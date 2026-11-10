package module

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/Gui97p/wisp/internal/diag"
	"github.com/Gui97p/wisp/internal/target"
)

type Config struct {
	Entry   string                  `toml:"entry" doc:"Entry file of the project, relative to the project root. Defaults to main.wsp."`
	OS      string                  `toml:"os" doc:"The system to build for: linux, cc, or a name declared in [targets.<name>]. Defaults to the system of the machine."`
	Arch    string                  `toml:"arch" doc:"The architecture to build for. Only needed when the system has more than one; otherwise it is a check."`
	Std     string                  `toml:"std" doc:"The standard library. Leave it out for the one that ships with the compiler, write a folder to replace it, or \"none\" to build without any (a folder called none is written ./none)."`
	Link    []string                `toml:"link" doc:"Extra arguments passed to the linker (libraries, object files, flags)."`
	Libc    bool                    `toml:"libc" doc:"Link against the C library instead of the Wisp runtime start code."`
	Output  OutputConfig            `toml:"output" doc:"Where build artifacts are written."`
	Run     RunConfig               `toml:"run" doc:"How wisp run starts the program."`
	Targets map[string]TargetConfig `toml:"targets" doc:"Custom targets, declared as [targets.<name>]. Used for kernels and other bare-metal environments."`
}

type RunConfig struct {
	Env map[string]any `toml:"env" doc:"Environment variables for the program under wisp run, written NAME = \"value\". A variable already set in the shell wins."`
}

type EnvVar struct {
	Name  string
	Value string
}

func (c *Config) RunEnv() []EnvVar {
	names := make([]string, 0, len(c.Run.Env))
	for name := range c.Run.Env {
		names = append(names, name)
	}
	sort.Strings(names)

	vars := make([]EnvVar, 0, len(names))
	for _, name := range names {
		if value, ok := c.Run.Env[name].(string); ok {
			vars = append(vars, EnvVar{Name: name, Value: value})
		}
	}
	return vars
}

type TargetConfig struct {
	Arch    string       `toml:"arch" doc:"CPU architecture of the target. Only x64 is supported."`
	ABI     string       `toml:"abi" doc:"Calling convention. Only sysv is supported, and it is the default."`
	Format  string       `toml:"format" doc:"Object format. Only elf64 is supported, and it is the default."`
	Entry   string       `toml:"entry" doc:"Symbol where the linker starts the program. Defaults to _start."`
	Runtime string       `toml:"runtime" doc:"Assembly file with the runtime routines (allocation, panic, I/O) for this target."`
	Start   string       `toml:"start" doc:"Assembly file that calls main and leaves the program. Required for custom targets."`
	Link    []string     `toml:"link" doc:"Extra linker arguments for this target, such as -T linker.ld. On a built-in target they are added to the built-in ones."`
	Libc    *bool        `toml:"libc" doc:"Link against the C library for this target. Overrides the top-level libc."`
	Output  OutputConfig `toml:"output" doc:"Where this target writes its artifacts. Overrides the top-level [output]."`
}

type OutputConfig = target.Output

const StdNone = "none"

type ConfigField struct {
	Key   string
	Type  string
	Doc   string
	Table bool
}

func ConfigFields(section string) []ConfigField {
	var t reflect.Type
	switch section {
	case "":
		t = reflect.TypeOf(Config{})
	case "output":
		t = reflect.TypeOf(OutputConfig{})
	case "run":
		t = reflect.TypeOf(RunConfig{})
	case "targets":
		t = reflect.TypeOf(TargetConfig{})
	case "targets.output":
		t = reflect.TypeOf(OutputConfig{})
	default:
		return nil
	}

	fields := make([]ConfigField, 0, t.NumField())
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		key := f.Tag.Get("toml")
		if key == "" {
			continue
		}

		field := ConfigField{Key: key, Doc: f.Tag.Get("doc")}
		kind := f.Type.Kind()
		if kind == reflect.Ptr {
			kind = f.Type.Elem().Kind()
		}
		switch kind {
		case reflect.Bool:
			field.Type = "bool"
		case reflect.Slice:
			field.Type = "array of strings"
		case reflect.String:
			field.Type = "string"
		default:
			field.Type = "table"
			field.Table = true
		}
		fields = append(fields, field)
	}
	return fields
}

func ConfigFieldNamed(section, key string) (ConfigField, bool) {
	for _, f := range ConfigFields(section) {
		if f.Key == key {
			return f, true
		}
	}
	return ConfigField{}, false
}

func LoadConfig(root string) (*Config, error) {
	path := filepath.Join(root, "wisp.toml")

	if _, err := os.Stat(path); err != nil {
		target.ResetProject()
		return &Config{}, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return loadConfigText(root, path, string(data))
}

func LoadConfigSource(root, text string) (*Config, error) {
	return loadConfigText(root, filepath.Join(root, "wisp.toml"), text)
}

func loadConfigText(root, path, text string) (*Config, error) {
	cfg, err := decodeConfigSource(path, text)
	if err != nil {
		return nil, err
	}
	if err := cfg.registerTargets(root); err != nil {
		var fieldErr *target.FieldError
		if !errors.As(err, &fieldErr) {
			return nil, err
		}
		line := locateKey(text, toml.Key{"targets", fieldErr.Target, fieldErr.Field})
		if line == 0 {
			line = locateKey(text, toml.Key{"targets", fieldErr.Target})
		}
		return nil, &ConfigError{File: path, Source: text, Issues: []ConfigIssue{{Line: line, Message: fieldErr.Summary(), Help: fieldErr.Help}}}
	}
	return cfg, nil
}

func (c *Config) registerTargets(root string) error {
	target.ResetProject()

	names := make([]string, 0, len(c.Targets))
	for name := range c.Targets {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		tc := c.Targets[name]

		var start, runtimeSource string
		if tc.Start != "" {
			data, err := os.ReadFile(filepath.Join(root, tc.Start))
			if err != nil {
				return &target.FieldError{Target: name, Field: "start", Message: "cannot read the start file", Help: fmt.Sprintf("%s could not be opened, the system reports %s", tc.Start, diag.Reason(err))}
			}
			start = string(data)
		}
		if tc.Runtime != "" {
			data, err := os.ReadFile(filepath.Join(root, tc.Runtime))
			if err != nil {
				return &target.FieldError{Target: name, Field: "runtime", Message: "cannot read the runtime file", Help: fmt.Sprintf("%s could not be opened, the system reports %s", tc.Runtime, diag.Reason(err))}
			}
			runtimeSource = string(data)
		}

		if target.IsBuiltin(name) {
			err := target.Override(name, target.Patch{
				Arch:          tc.Arch,
				ABI:           tc.ABI,
				Format:        tc.Format,
				Entry:         tc.Entry,
				StartSource:   start,
				RuntimeSource: runtimeSource,
				LinkArgs:      tc.Link,
				Libc:          tc.Libc,
				Output:        tc.Output,
			})
			if err != nil {
				return err
			}
			continue
		}

		err := target.Register(target.Target{
			Name:          name,
			Arch:          tc.Arch,
			ABI:           tc.ABI,
			Format:        tc.Format,
			Entry:         tc.Entry,
			StartSource:   start,
			RuntimeSource: runtimeSource,
			LinkArgs:      tc.Link,
			Libc:          tc.Libc,
			Output:        tc.Output,
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *Config) StdlibSettings(root string) (string, bool, error) {
	if c.Std == StdNone {
		return "", true, nil
	}

	path := c.Std
	if path != "" && !filepath.IsAbs(path) {
		path = filepath.Join(root, path)
	}
	return path, false, nil
}

func LoadStdlibSettings(root string) (string, bool, error) {
	path := filepath.Join(root, "wisp.toml")
	if _, err := os.Stat(path); err != nil {
		return "", false, nil
	}

	cfg, err := decodeConfig(path)
	if err != nil {
		return "", false, err
	}
	return cfg.StdlibSettings(root)
}

type ConfigIssue struct {
	Line    int
	Message string
	Help    string
}

type ConfigError struct {
	File   string
	Source string
	Issues []ConfigIssue
}

func (e *ConfigError) Error() string {
	lines := make([]string, len(e.Issues))
	for i, issue := range e.Issues {
		text := issue.Message
		if issue.Help != "" {
			text += " (" + issue.Help + ")"
		}
		if issue.Line > 0 {
			lines[i] = fmt.Sprintf("%s:%d: %s", filepath.Base(e.File), issue.Line, text)
		} else {
			lines[i] = fmt.Sprintf("%s: %s", filepath.Base(e.File), text)
		}
	}
	return strings.Join(lines, "\n")
}

func (e *ConfigError) Diagnostics() diag.List {
	source := strings.Split(e.Source, "\n")
	list := make(diag.List, 0, len(e.Issues))
	for _, issue := range e.Issues {
		d := diag.Diagnostic{File: e.File, Message: issue.Message, Help: issue.Help, Line: max(issue.Line, 1), Col: 1}
		for d.Line > 1 && d.Line-1 < len(source) && strings.TrimSpace(source[d.Line-1]) == "" {
			d.Line--
		}
		if d.Line-1 < len(source) {
			text := strings.TrimRight(source[d.Line-1], " \t\r")
			d.Col = len(text) - len(strings.TrimLeft(text, " \t")) + 1
			d.EndLine, d.EndCol = d.Line, max(len(text), d.Col)
		}
		list = append(list, d)
	}
	return list
}

func decodeConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return decodeConfigSource(path, string(data))
}

func decodeConfigSource(path, text string) (*Config, error) {
	var cfg Config
	md, err := toml.Decode(text, &cfg)
	if err != nil {
		if parseErr, ok := errors.AsType[toml.ParseError](err); ok {
			return nil, &ConfigError{File: path, Source: text, Issues: []ConfigIssue{{Line: parseErr.Position.Line, Message: parseErr.Message}}}
		}
		return nil, &ConfigError{File: path, Source: text, Issues: []ConfigIssue{{Message: diag.Reason(err)}}}
	}

	var issues []ConfigIssue
	var reported []toml.Key
	for _, key := range md.Undecoded() {
		if coveredBy(key, reported) {
			continue
		}
		reported = append(reported, key)

		if len(key) >= 3 && key[0] == "targets" && slices.Contains(target.Architectures(), key[2]) {
			issues = append(issues, ConfigIssue{
				Line:    locateKey(text, key),
				Message: fmt.Sprintf("a table per architecture is not available yet: %q", key.String()),
				Help:    fmt.Sprintf("write these keys in [targets.%s]: while %s has a single architecture, they apply to all of it", key[1], key[1]),
			})
			continue
		}

		known := knownKeys(key)
		issues = append(issues, ConfigIssue{
			Line:    locateKey(text, key),
			Message: fmt.Sprintf("unknown key %q", key.String()),
			Help:    target.Suggestion(key[len(key)-1], known, "keys"),
		})
	}
	for name, value := range cfg.Run.Env {
		key := toml.Key{"run", "env", name}
		switch {
		case name == "" || strings.ContainsAny(name, "=\x00"):
			issues = append(issues, ConfigIssue{Line: locateKey(text, key), Message: fmt.Sprintf("%q is not a valid environment variable name", name)})
		default:
			if _, isString := value.(string); !isString {
				issues = append(issues, ConfigIssue{Line: locateKey(text, key), Message: fmt.Sprintf("run.env.%s must be a string", name), Help: fmt.Sprintf("write it as %s = \"...\"", name)})
			}
		}
	}
	sort.Slice(issues, func(i, j int) bool { return issues[i].Line < issues[j].Line })
	if len(issues) > 0 {
		return nil, &ConfigError{File: path, Source: text, Issues: issues}
	}

	return &cfg, nil
}

func coveredBy(key toml.Key, reported []toml.Key) bool {
	for _, r := range reported {
		if len(r) <= len(key) && strings.Join(r, "\x00") == strings.Join(key[:len(r)], "\x00") {
			return true
		}
	}
	return false
}

func knownKeys(key toml.Key) []string {
	switch {
	case len(key) >= 4 && key[0] == "targets" && key[2] == "output":
		return tomlTags(reflect.TypeOf(OutputConfig{}))
	case len(key) >= 3 && key[0] == "targets":
		return tomlTags(reflect.TypeOf(TargetConfig{}))
	case len(key) >= 2 && key[0] == "output":
		return tomlTags(reflect.TypeOf(OutputConfig{}))
	case len(key) >= 2 && key[0] == "run":
		return tomlTags(reflect.TypeOf(RunConfig{}))
	}
	return tomlTags(reflect.TypeOf(Config{}))
}

func tomlTags(t reflect.Type) []string {
	tags := make([]string, 0, t.NumField())
	for i := 0; i < t.NumField(); i++ {
		if tag := t.Field(i).Tag.Get("toml"); tag != "" {
			tags = append(tags, tag)
		}
	}
	sort.Strings(tags)
	return tags
}

var (
	tableHeader = regexp.MustCompile(`^\s*\[\[?\s*([^\]]+?)\s*\]\]?\s*(#.*)?$`)
	keyLine     = regexp.MustCompile(`^\s*("[^"]*"|[A-Za-z0-9_-]+)\s*=`)
)

func locateKey(src string, key toml.Key) int {
	var table []string
	want := strings.Join(key, ".")

	for i, line := range strings.Split(src, "\n") {
		if m := tableHeader.FindStringSubmatch(line); m != nil {
			table = splitTable(m[1])
			if strings.Join(table, ".") == want {
				return i + 1
			}
			continue
		}
		if m := keyLine.FindStringSubmatch(line); m != nil {
			name := strings.Trim(m[1], `"`)
			if strings.Join(append(append([]string{}, table...), name), ".") == want {
				return i + 1
			}
		}
	}
	return 0
}

func splitTable(header string) []string {
	parts := strings.Split(header, ".")
	for i, part := range parts {
		parts[i] = strings.Trim(strings.TrimSpace(part), `"`)
	}
	return parts
}
