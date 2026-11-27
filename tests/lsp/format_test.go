package lsp

import (
	"slices"
	"strings"
	"testing"
)

const formatHead = "import \"std/fmt\";\n\nstruct Point {\n  int x;\n  int y;\n}\n\nenum Color {\n  Red,\n  Green,\n}\n\nfunc add(int a, int b) int {\n  return a + b;\n}\n\n"

func formatSource(body string) string {
	return formatHead + "func show(int count, Point p) {\n  [3]int nums = .{1, 2, 3};\n" + body + "\n}\n"
}

func TestHoverInsideAFormatString(t *testing.T) {
	c := start(t)
	cases := []struct {
		name, src, want string
	}{
		{"a variable", `  fmt.println("n {cou¦nt}");`, "int count"},
		{"a member", `  fmt.println("x {p.¦x}");`, "int x"},
		{"a function", `  fmt.println("sum {ad¦d(count, 2)}");`, "func add"},
		{"an argument of a call", `  fmt.println("sum {add(co¦unt, 2)}");`, "int count"},
		{"after a spec was written", `  fmt.println("x {co¦unt:04}");`, "int count"},
		{"a second expression", `  fmt.println("a {count} b {p.¦y}");`, "int y"},
		{"after an escaped quote", `  fmt.println("q \"{co¦unt}\"");`, "int count"},
		{"in a raw string", "  fmt.println(`raw {co¦unt}`);", "int count"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := c.open(map[string]string{"main.wsp": formatSource(tc.src)}).hover()
			if !strings.Contains(got, tc.want) {
				t.Errorf("hover %q, want it to contain %q", got, tc.want)
			}
		})
	}

	for _, tc := range []struct{ name, src string }{
		{"the text of the string", `  fmt.println("he¦llo {count}");`},
		{"an opening brace", `  fmt.println("a ¦{count}");`},
		{"a spec", `  fmt.println("a {count:¦04}");`},
		{"a literal brace", `  fmt.println("a {{co¦unt}}");`},
	} {
		t.Run("nothing on "+tc.name, func(t *testing.T) {
			if got := c.open(map[string]string{"main.wsp": formatSource(tc.src)}).hover(); got != "" {
				t.Errorf("hover %q, want nothing", got)
			}
		})
	}
}

func TestDefinitionAndRenameInsideAFormatString(t *testing.T) {
	c := start(t)

	t.Run("definition goes to the declaration", func(t *testing.T) {
		p := c.open(map[string]string{"main.wsp": formatSource(`  fmt.println("n {cou¦nt}");`)})
		defs := p.definition()
		if len(defs) != 1 || defs[0].Range.Start.Line != 16 {
			t.Errorf("definition: %+v", defs)
		}
	})

	t.Run("rename changes the string too", func(t *testing.T) {
		p := c.open(map[string]string{"main.wsp": formatSource(`  fmt.println("n {cou¦nt} x {p.x} {count + 1:04}\t{{count}}");`)})
		got := p.renamed("total")
		want := formatSource(`  fmt.println("n {total} x {p.x} {total + 1:04}\t{{count}}");`)
		want = strings.Replace(want, "func show(int count,", "func show(int total,", 1)
		if got != want {
			t.Errorf("renamed wrongly\n--- got ---\n%s\n--- want ---\n%s", got, want)
		}
	})

	t.Run("rename keeps the escapes that come before", func(t *testing.T) {
		p := c.open(map[string]string{"main.wsp": formatSource(`  fmt.println("q \"{¦count}\" \\ {count}");`)})
		got := p.renamed("total")
		if !strings.Contains(got, `"q \"{total}\" \\ {total}"`) {
			t.Errorf("renamed wrongly:\n%s", got)
		}
	})

	t.Run("rename from the string reaches the declaration", func(t *testing.T) {
		p := c.open(map[string]string{"main.wsp": formatSource(`  fmt.println("a {¦count}");`)})
		got := p.renamed("total")
		if !strings.Contains(got, "func show(int total,") || !strings.Contains(got, `{total}`) {
			t.Errorf("renamed wrongly:\n%s", got)
		}
	})
}

func TestSemanticTokensInsideAFormatString(t *testing.T) {
	c := start(t)
	src := formatSource(`  fmt.println("n {count} sum {add(count, 2)} x {p.x:>4} first {nums[0]:03} ok {true} {{lit}} {:x}", 255);`)
	p := c.open(map[string]string{"main.wsp": src})
	line := strings.Index(src, "fmt.println(\"n") // used only to find the line below
	_ = line
	row := strings.Count(src[:strings.Index(src, `fmt.println("n`)], "\n")
	text := strings.Split(src, "\n")[row]

	type tok struct {
		text, kind string
	}
	var got []tok
	for _, tk := range p.semanticTokens() {
		if tk.line == row {
			got = append(got, tok{text[tk.col : tk.col+tk.length], tk.kind})
		}
	}
	for _, want := range []tok{
		{"{", "formatBrace"}, {"count", "formatPlain"}, {"}", "formatBrace"},
		{"add", "function"}, {"(", "formatPlain"}, {",", "formatPlain"}, {"2", "number"},
		{"p", "formatPlain"}, {".", "formatPlain"}, {"x", "property"},
		{":>4", "formatSpec"}, {"nums", "formatPlain"}, {"[", "formatPlain"}, {"]", "formatPlain"}, {":03", "formatSpec"},
		{"true", "keyword"}, {":x", "formatSpec"},
	} {
		if !slices.Contains(got, want) {
			t.Errorf("missing the token %+v in %+v", want, got)
		}
	}
	for _, tk := range got {
		if tk.text == "lit" || tk.text == "{lit}" {
			t.Errorf("an escaped brace was colored: %+v", tk)
		}
	}
}

func TestOperatorsInsideAFormatStringAreColoredAsOperators(t *testing.T) {
	c := start(t)
	src := formatSource(`  fmt.println("sum {count + 1 * 2} cmp {count >= 2 && true} ternary {count > 1 ? 1 : 0} range {nums[0] - -1}");`)
	p := c.open(map[string]string{"main.wsp": src})
	row := strings.Count(src[:strings.Index(src, `fmt.println("sum`)], "\n")
	text := strings.Split(src, "\n")[row]
	var operators []string
	for _, tk := range p.semanticTokens() {
		if tk.line == row && tk.kind == "operator" {
			operators = append(operators, text[tk.col:tk.col+tk.length])
		}
	}
	want := []string{"+", "*", ">=", "&&", ">", "?", ":", "-", "-"}
	if !slices.Equal(operators, want) {
		t.Errorf("operators %v, want %v", operators, want)
	}
}

func TestSemanticTokensAfterWideCharacters(t *testing.T) {
	c := start(t)
	src := formatSource(`  fmt.println("é ✓ {count} 𝄞 {count:03}");`)
	p := c.open(map[string]string{"main.wsp": src})
	row := strings.Count(src[:strings.Index(src, `fmt.println("é`)], "\n")
	var braces []int
	for _, tk := range p.semanticTokens() {
		if tk.line == row && tk.kind == "formatBrace" {
			braces = append(braces, tk.col)
		}
	}
	// The line is two spaces, `fmt.println("`, `é ✓ `, `{count}`, ` 𝄞 ` and `{count:03}`. The text
	// before the first brace takes 19 columns, and 𝄞 takes two UTF-16 units, which is what the editor counts.
	want := []int{19, 25, 30, 39}
	if !slices.Equal(braces, want) {
		t.Errorf("braces at %v, want %v", braces, want)
	}
}

func TestCompletionInsideAFormatString(t *testing.T) {
	c := start(t)

	has := func(labels []string, want ...string) bool {
		for _, w := range want {
			if !slices.Contains(labels, w) {
				return false
			}
		}
		return true
	}

	t.Run("a name", func(t *testing.T) {
		got := c.open(map[string]string{"main.wsp": formatSource(`  fmt.println("n {cou¦}");`)}).completion()
		if !has(got, "count", "p", "add", "nums") {
			t.Errorf("completion: %v", got)
		}
	})
	t.Run("a name right after the brace", func(t *testing.T) {
		got := c.open(map[string]string{"main.wsp": formatSource(`  fmt.println("n {¦}");`)}).completion()
		if !has(got, "count", "p") {
			t.Errorf("completion: %v", got)
		}
	})
	t.Run("the fields after a dot", func(t *testing.T) {
		got := c.open(map[string]string{"main.wsp": formatSource(`  fmt.println("x {p.¦}");`)}).completion()
		if !has(got, "x", "y") {
			t.Errorf("completion: %v", got)
		}
	})
	t.Run("the fields while typing one", func(t *testing.T) {
		got := c.open(map[string]string{"main.wsp": formatSource(`  fmt.println("x {p.x¦}");`)}).completion()
		if !has(got, "x", "y") {
			t.Errorf("completion: %v", got)
		}
	})
	t.Run("an argument of a call", func(t *testing.T) {
		got := c.open(map[string]string{"main.wsp": formatSource(`  fmt.println("x {add(co¦, 2)}");`)}).completion()
		if !has(got, "count") {
			t.Errorf("completion: %v", got)
		}
	})
	t.Run("the second brace", func(t *testing.T) {
		got := c.open(map[string]string{"main.wsp": formatSource(`  fmt.println("a {count} b {p.¦}");`)}).completion()
		if !has(got, "x", "y") {
			t.Errorf("completion: %v", got)
		}
	})
	t.Run("a module member", func(t *testing.T) {
		src := "import \"std/fmt\";\nimport \"std/math\";\n\nfunc main() {\n  fmt.println(\"pi {math.¦}\");\n}\n"
		got := c.open(map[string]string{"main.wsp": src}).completion()
		if !has(got, "max", "PI") {
			t.Errorf("completion: %v", got)
		}
	})

	for _, tc := range []struct{ name, src string }{
		{"the text of the string", `  fmt.println("n co¦");`},
		{"after the spec colon", `  fmt.println("n {count:¦}");`},
		{"a string that is not a format", `  string s = "n {cou¦}";`},
		{"a literal brace", `  fmt.println("n {{cou¦}}");`},
	} {
		t.Run("nothing in "+tc.name, func(t *testing.T) {
			if got := c.open(map[string]string{"main.wsp": formatSource(tc.src)}).completion(); len(got) != 0 {
				t.Errorf("completion: %v", got[:min(len(got), 8)])
			}
		})
	}
}

func TestAnErrorInAFormatExpressionPointsAtIt(t *testing.T) {
	c := start(t)
	src := formatSource(`  fmt.println("a {count} b {cuont + 1} c");`)
	p := c.open(map[string]string{"main.wsp": src})
	row := strings.Count(src[:strings.Index(src, `fmt.println("a`)], "\n")
	text := strings.Split(src, "\n")[row]
	found := false
	for _, d := range p.diagnostics() {
		if d.Severity != 1 || d.Range.Start.Line != row {
			continue
		}
		found = true
		got := text[d.Range.Start.Character:d.Range.End.Character]
		if !strings.HasPrefix(got, "cuont") {
			t.Errorf("the error covers %q: %s", got, d.Message)
		}
	}
	if !found {
		t.Error("no error was reported")
	}
}

func TestHoverIsQuietWhereItOnlyRepeatsTheType(t *testing.T) {
	c := start(t)

	for _, tc := range []struct{ name, line string }{
		{"a string", `  string name = "wi¦sp";`},
		{"an integer", `  int total = add(¦2, 3);`},
		{"a float", `  float ratio = 2.¦5;`},
		{"a character", `  byte letter = 'a¦';`},
		{"the opening bracket of an index", `  int first = nums¦[0];`},
		{"the closing bracket of an index", `  int first = nums[0¦];`},
		{"the opening bracket of a slice", `  []int part = nums¦[1..3];`},
		{"the opening parenthesis of a call", `  int total = add¦(2, 3);`},
		{"the closing parenthesis of a call", `  int total = add(2, 3¦);`},
	} {
		t.Run("nothing on "+tc.name, func(t *testing.T) {
			if got := c.open(map[string]string{"main.wsp": formatSource(tc.line)}).hover(); got != "" {
				t.Errorf("hover %q, want nothing", got)
			}
		})
	}

	for _, tc := range []struct{ name, line, want string }{
		{"the callee of a call", `  int total = ad¦d(2, 3);`, "func add"},
		{"a variable in an index", `  int first = nu¦ms[0];`, "[3]int"},
		{"a variable in a slice", `  []int part = nu¦ms[1..3];`, "[3]int"},
		{"an argument of a call", `  int total = add(cou¦nt, 3);`, "int count"},
		{"the second dot of a slice", `  []int part = nums[1.¦.3];`, "[]int"},
		{"the first dot of a slice", `  []int part = nums[1¦..3];`, "[]int"},
		{"an inclusive slice", `  []int part = nums[1..¦=2];`, "[]int"},
	} {
		t.Run(tc.name+" keeps its hover", func(t *testing.T) {
			if got := c.open(map[string]string{"main.wsp": formatSource(tc.line)}).hover(); !strings.Contains(got, tc.want) {
				t.Errorf("hover %q, want it to contain %q", got, tc.want)
			}
		})
	}

	t.Run("a range keeps its type", func(t *testing.T) {
		src := "func main() {\n\tRange<int> r = 0.¦.9;\n}\n"
		if got := c.open(map[string]string{"main.wsp": src}).hover(); !strings.Contains(got, "Range<int>") {
			t.Errorf("hover %q", got)
		}
	})
}

func TestHoverDoesNotReadComments(t *testing.T) {
	c := start(t)
	for _, tc := range []struct{ name, line string }{
		{"a name in a comment", `  int total = 2 + 3;  // the length is le¦n(name) here`},
		{"a builtin in a comment", `  // len¦ measures a string`},
		{"a name in a doc comment", `  /// le¦n measures`},
		{"inside a block comment", `  /* count¦ is the number of things */`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := c.open(map[string]string{"main.wsp": formatSource(tc.line)}).hover(); got != "" {
				t.Errorf("hover %q, want nothing", got)
			}
		})
	}

	t.Run("code before a comment on the same line keeps its hover", func(t *testing.T) {
		got := c.open(map[string]string{"main.wsp": formatSource(`  int total = add(cou¦nt, 2);  // the sum`)}).hover()
		if !strings.Contains(got, "int count") {
			t.Errorf("hover %q", got)
		}
	})
}
