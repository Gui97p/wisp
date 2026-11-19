package lsp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

const cursor = "¦"

type message struct {
	ID     *json.RawMessage `json:"id,omitempty"`
	Method string           `json:"method,omitempty"`
	Params json.RawMessage  `json:"params,omitempty"`
	Result json.RawMessage  `json:"result,omitempty"`
	Error  *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

type client struct {
	t       *testing.T
	cmd     *exec.Cmd
	in      io.WriteCloser
	mu      sync.Mutex
	nextID  int
	waiting map[int]chan message
	diags   map[string][]diagnostic
	changed chan struct{}
}

type diagnostic struct {
	Message  string `json:"message"`
	Severity int    `json:"severity"`
	Range    struct {
		Start struct{ Line, Character int }
	} `json:"range"`
}

type position struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

type location struct {
	URI   string `json:"uri"`
	Range struct {
		Start position `json:"start"`
	} `json:"range"`
}

func repoRoot() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

var binary string

func TestMain(m *testing.M) {
	root := repoRoot()
	binary = filepath.Join(root, "bin", "wisp")
	build := exec.Command("go", "build", "-o", binary, "./cmd/main")
	build.Dir = root
	if out, err := build.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "build failed: %v\n%s", err, out)
		os.Exit(1)
	}
	os.Exit(m.Run())
}

func start(t *testing.T) *client {
	t.Helper()
	cmd := exec.Command(binary, "lsp")
	in, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	out, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	c := &client{t: t, cmd: cmd, in: in, waiting: map[int]chan message{}, diags: map[string][]diagnostic{}, changed: make(chan struct{}, 64)}
	go c.read(bufio.NewReader(out))
	t.Cleanup(func() {
		in.Close()
		cmd.Process.Kill()
		cmd.Wait()
	})

	dir := t.TempDir()
	c.call("initialize", map[string]any{"processId": nil, "rootUri": "file://" + dir, "capabilities": map[string]any{}})
	c.notify("initialized", map[string]any{})
	return c
}

func (c *client) read(r *bufio.Reader) {
	for {
		length := 0
		for {
			line, err := r.ReadString('\n')
			if err != nil {
				return
			}
			line = strings.TrimSpace(line)
			if line == "" {
				break
			}
			if v, ok := strings.CutPrefix(line, "Content-Length:"); ok {
				length, _ = strconv.Atoi(strings.TrimSpace(v))
			}
		}
		body := make([]byte, length)
		if _, err := io.ReadFull(r, body); err != nil {
			return
		}
		var msg message
		if json.Unmarshal(body, &msg) != nil {
			continue
		}
		switch {
		case msg.Method == "textDocument/publishDiagnostics":
			var p struct {
				URI         string       `json:"uri"`
				Diagnostics []diagnostic `json:"diagnostics"`
			}
			json.Unmarshal(msg.Params, &p)
			c.mu.Lock()
			c.diags[p.URI] = p.Diagnostics
			c.mu.Unlock()
			select {
			case c.changed <- struct{}{}:
			default:
			}
		case msg.Method != "" && msg.ID != nil:
			c.send(map[string]any{"jsonrpc": "2.0", "id": json.RawMessage(*msg.ID), "result": nil})
		case msg.ID != nil:
			var id int
			json.Unmarshal(*msg.ID, &id)
			c.mu.Lock()
			ch := c.waiting[id]
			delete(c.waiting, id)
			c.mu.Unlock()
			if ch != nil {
				ch <- msg
			}
		}
	}
}

func (c *client) send(v any) {
	body, _ := json.Marshal(v)
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "Content-Length: %d\r\n\r\n", len(body))
	buf.Write(body)
	c.mu.Lock()
	defer c.mu.Unlock()
	c.in.Write(buf.Bytes())
}

func (c *client) notify(method string, params any) {
	c.send(map[string]any{"jsonrpc": "2.0", "method": method, "params": params})
}

func (c *client) call(method string, params any) json.RawMessage {
	c.t.Helper()
	c.mu.Lock()
	c.nextID++
	id := c.nextID
	ch := make(chan message, 1)
	c.waiting[id] = ch
	c.mu.Unlock()
	c.send(map[string]any{"jsonrpc": "2.0", "id": id, "method": method, "params": params})
	select {
	case msg := <-ch:
		if msg.Error != nil {
			c.t.Fatalf("%s failed: %s", method, msg.Error.Message)
		}
		return msg.Result
	case <-time.After(20 * time.Second):
		c.t.Fatalf("%s timed out", method)
	}
	return nil
}

type project struct {
	c    *client
	dir  string
	path string
	text string
	line int
	char int
}

func (c *client) open(files map[string]string) *project {
	return c.openAs("main.wsp", files)
}

func (c *client) openAs(primary string, files map[string]string) *project {
	c.t.Helper()
	dir := c.t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "wisp.toml"), []byte("[output]\nbin = \"bin\"\n"), 0o644); err != nil {
		c.t.Fatal(err)
	}
	p := &project{c: c, dir: dir}
	for name, text := range files {
		clean, line, char := stripCursor(text)
		if err := os.WriteFile(filepath.Join(dir, name), []byte(clean), 0o644); err != nil {
			c.t.Fatal(err)
		}
		if name == primary {
			p.path, p.text, p.line, p.char = filepath.Join(dir, name), clean, line, char
		}
	}
	for name := range files {
		path := filepath.Join(dir, name)
		data, _ := os.ReadFile(path)
		c.notify("textDocument/didOpen", map[string]any{"textDocument": map[string]any{"uri": "file://" + path, "languageId": "wisp", "version": 1, "text": string(data)}})
	}
	return p
}

func stripCursor(text string) (string, int, int) {
	i := strings.Index(text, cursor)
	if i < 0 {
		return text, 0, 0
	}
	clean := text[:i] + text[i+len(cursor):]
	line := strings.Count(clean[:i], "\n")
	char := i - (strings.LastIndex(clean[:i], "\n") + 1)
	return clean, line, char
}

func (p *project) uri() string { return "file://" + p.path }

func (p *project) at() map[string]any {
	return map[string]any{"textDocument": map[string]any{"uri": p.uri()}, "position": position{p.line, p.char}}
}

func (p *project) hover() string {
	p.c.t.Helper()
	raw := p.c.call("textDocument/hover", p.at())
	var h struct {
		Contents struct {
			Value string `json:"value"`
		} `json:"contents"`
	}
	json.Unmarshal(raw, &h)
	return h.Contents.Value
}

func (p *project) definition() []location {
	p.c.t.Helper()
	raw := p.c.call("textDocument/definition", p.at())
	var locations []location
	if json.Unmarshal(raw, &locations) != nil {
		var one location
		if json.Unmarshal(raw, &one) == nil && one.URI != "" {
			locations = []location{one}
		}
	}
	return locations
}

func (p *project) completion() []string {
	p.c.t.Helper()
	params := p.at()
	params["context"] = map[string]any{"triggerKind": 1}
	raw := p.c.call("textDocument/completion", params)
	var items []struct {
		Label string `json:"label"`
	}
	if json.Unmarshal(raw, &items) != nil {
		var list struct {
			Items []struct {
				Label string `json:"label"`
			} `json:"items"`
		}
		json.Unmarshal(raw, &list)
		items = list.Items
	}
	labels := make([]string, len(items))
	for i, it := range items {
		labels[i] = it.Label
	}
	return labels
}

func (p *project) diagnostics() []diagnostic {
	p.c.t.Helper()
	deadline := time.After(20 * time.Second)
	for {
		p.c.mu.Lock()
		d, ok := p.c.diags[p.uri()]
		p.c.mu.Unlock()
		if ok {
			return d
		}
		select {
		case <-p.c.changed:
		case <-deadline:
			p.c.t.Fatal("no diagnostics were published")
		}
	}
}

func (p *project) formatted() (string, bool) {
	p.c.t.Helper()
	raw := p.c.call("textDocument/formatting", map[string]any{
		"textDocument": map[string]any{"uri": p.uri()},
		"options":      map[string]any{"tabSize": 4, "insertSpaces": false},
	})
	var edits []struct {
		Range struct {
			Start position `json:"start"`
			End   position `json:"end"`
		} `json:"range"`
		NewText string `json:"newText"`
	}
	if json.Unmarshal(raw, &edits) != nil || len(edits) == 0 {
		return p.text, false
	}
	return edits[0].NewText, true
}

type textEdit struct {
	Range struct {
		Start position `json:"start"`
		End   position `json:"end"`
	} `json:"range"`
	NewText string `json:"newText"`
}

type completionItem struct {
	Label               string     `json:"label"`
	Detail              string     `json:"detail"`
	AdditionalTextEdits []textEdit `json:"additionalTextEdits"`
}

func (p *project) completionItems() []completionItem {
	p.c.t.Helper()
	params := p.at()
	params["context"] = map[string]any{"triggerKind": 1}
	raw := p.c.call("textDocument/completion", params)
	var items []completionItem
	if json.Unmarshal(raw, &items) != nil {
		var list struct {
			Items []completionItem `json:"items"`
		}
		json.Unmarshal(raw, &list)
		items = list.Items
	}
	return items
}

type codeAction struct {
	Title string `json:"title"`
	Kind  string `json:"kind"`
	Edit  struct {
		Changes map[string][]textEdit `json:"changes"`
	} `json:"edit"`
}

func (p *project) codeActions(only ...string) []codeAction {
	p.c.t.Helper()
	diagnostics := []diagnostic{}
	if len(only) == 0 {
		diagnostics = p.diagnostics()
	}
	raw := p.c.call("textDocument/codeAction", map[string]any{
		"textDocument": map[string]any{"uri": p.uri()},
		"range":        map[string]any{"start": position{0, 0}, "end": position{0, 0}},
		"context":      map[string]any{"diagnostics": p.rawDiagnostics(diagnostics), "only": only},
	})
	var actions []codeAction
	json.Unmarshal(raw, &actions)
	return actions
}

func (p *project) rawDiagnostics(list []diagnostic) []map[string]any {
	out := []map[string]any{}
	for _, d := range list {
		out = append(out, map[string]any{
			"message":  d.Message,
			"severity": d.Severity,
			"range": map[string]any{
				"start": position{d.Range.Start.Line, d.Range.Start.Character},
				"end":   position{d.Range.Start.Line, d.Range.Start.Character + 1},
			},
		})
	}
	return out
}

func (a codeAction) apply(p *project, text string) string {
	edits := a.Edit.Changes[p.uri()]
	lines := strings.Split(text, "\n")
	offset := func(pos position) int {
		n := 0
		for i := 0; i < pos.Line && i < len(lines); i++ {
			n += len(lines[i]) + 1
		}
		return n + pos.Character
	}
	sorted := append([]textEdit(nil), edits...)
	for i := range sorted {
		for j := i + 1; j < len(sorted); j++ {
			if offset(sorted[j].Range.Start) > offset(sorted[i].Range.Start) {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}
	for _, e := range sorted {
		start, end := offset(e.Range.Start), offset(e.Range.End)
		text = text[:start] + e.NewText + text[end:]
	}
	return text
}

type semanticToken struct {
	line, col, length int
	kind              string
}

func (p *project) semanticTokens() []semanticToken {
	p.c.t.Helper()
	raw := p.c.call("textDocument/semanticTokens/full", map[string]any{"textDocument": map[string]any{"uri": p.uri()}})
	var result struct {
		Data []int `json:"data"`
	}
	json.Unmarshal(raw, &result)
	kinds := []string{"property", "enumMember", "type", "typeParameter"}
	var tokens []semanticToken
	line, col := 0, 0
	for i := 0; i+4 < len(result.Data); i += 5 {
		line += result.Data[i]
		if result.Data[i] == 0 {
			col += result.Data[i+1]
		} else {
			col = result.Data[i+1]
		}
		tokens = append(tokens, semanticToken{line, col, result.Data[i+2], kinds[result.Data[i+3]]})
	}
	return tokens
}

func (p *project) renamed(newName string) string {
	p.c.t.Helper()
	params := p.at()
	params["newName"] = newName
	raw := p.c.call("textDocument/rename", params)
	var action codeAction
	if err := json.Unmarshal(raw, &struct {
		Changes *map[string][]textEdit `json:"changes"`
	}{&action.Edit.Changes}); err != nil {
		p.c.t.Fatal(err)
	}
	return action.apply(p, p.text)
}
