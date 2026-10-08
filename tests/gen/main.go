package main

import (
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type ty struct {
	name   string
	bits   int
	signed bool
}

var types = []ty{
	{"int8", 8, true}, {"int16", 16, true}, {"int32", 32, true}, {"int64", 64, true},
	{"uint8", 8, false}, {"uint16", 16, false}, {"uint32", 32, false}, {"uint64", 64, false},
}

func pow2(n int) *big.Int { return new(big.Int).Lsh(big.NewInt(1), uint(n)) }

func (t ty) min() *big.Int {
	if !t.signed {
		return big.NewInt(0)
	}
	return new(big.Int).Neg(pow2(t.bits - 1))
}

func (t ty) max() *big.Int {
	if !t.signed {
		return new(big.Int).Sub(pow2(t.bits), big.NewInt(1))
	}
	return new(big.Int).Sub(pow2(t.bits-1), big.NewInt(1))
}

func (t ty) wrap(v *big.Int) *big.Int {
	r := new(big.Int).Mod(v, pow2(t.bits))
	if t.signed && r.Cmp(pow2(t.bits-1)) >= 0 {
		r.Sub(r, pow2(t.bits))
	}
	return r
}

func (t ty) samples() []*big.Int {
	one := big.NewInt(1)
	mid := new(big.Int).Add(new(big.Int).Rsh(t.max(), 1), one)
	list := []*big.Int{
		big.NewInt(0), big.NewInt(1), t.max(), new(big.Int).Sub(t.max(), one), mid,
	}
	if t.signed {
		list = append(list, t.min(), big.NewInt(-1))
	}
	seen := map[string]bool{}
	var out []*big.Int
	for _, v := range list {
		if !seen[v.String()] {
			seen[v.String()] = true
			out = append(out, v)
		}
	}
	return out
}

func tag(v *big.Int) string {
	if v.Sign() < 0 {
		return "m" + new(big.Int).Neg(v).String()
	}
	return v.String()
}

const header = `func check(bool ok, string name) {
    if ok {
        emit(name);
    } else {
        emit("FAIL");
        emit(name);
    }
}

func checkEq(bool c, bool want, string name) {
    if want {
        check(c, name);
    } else {
        if c {
            emit("FAIL");
            emit(name);
        } else {
            emit(name);
        }
    }
}

`

const chunk = 15

type gen struct {
	b      strings.Builder
	names  []string
	cases  int
	funcs  int
	suffix string
}

var localName = regexp.MustCompile(`\b(a|b|r|e|n|c|l|q|el|er|en|ec)\b`)

func (g *gen) rename(text string) string {
	parts := strings.Split(text, "\"")
	for i := 0; i < len(parts); i += 2 {
		parts[i] = localName.ReplaceAllString(parts[i], "${1}"+g.suffix)
	}
	return strings.Join(parts, "\"")
}

func (g *gen) line(format string, args ...any) {
	g.b.WriteString("    ")
	g.b.WriteString(g.rename(fmt.Sprintf(format, args...)))
	g.b.WriteString("\n")
}

func (g *gen) open() {
	if g.cases%chunk == 0 {
		if g.cases > 0 {
			g.b.WriteString("}\n\n")
		}
		fmt.Fprintf(&g.b, "func t%d() {\n", g.funcs)
		g.funcs++
	}
	g.cases++
	g.suffix = fmt.Sprint(g.cases % chunk)
}

func (g *gen) source() string {
	var out strings.Builder
	out.WriteString(header)
	out.WriteString(g.b.String())
	if g.cases > 0 {
		out.WriteString("}\n\n")
	}
	out.WriteString("func main() {\n")
	for i := 0; i < g.funcs; i++ {
		fmt.Fprintf(&out, "    t%d();\n", i)
	}
	out.WriteString("}\n")
	return out.String()
}

func (g *gen) decl(t ty, name string, v *big.Int) {
	if v.Sign() >= 0 {
		g.line("%s %s = %s;", t.name, name, v)
		return
	}
	g.line("%s %s = 0;", t.name, name)
	rest := new(big.Int).Neg(v)
	limit := t.max()
	for rest.Sign() > 0 {
		chunk := new(big.Int).Set(rest)
		if chunk.Cmp(limit) > 0 {
			chunk.Set(limit)
		}
		g.line("%s -= %s;", name, chunk)
		rest.Sub(rest, chunk)
	}
}

func (g *gen) check(cond, name string) {
	g.names = append(g.names, name)
	g.line("check(%s, %q);", cond, name)
}

func (g *gen) checkEq(cond string, want bool, name string) {
	g.names = append(g.names, name)
	g.line("bool c = %s;", cond)
	g.line("checkEq(c, %t, %q);", want, name)
}

type testcase struct {
	dir      string
	src      string
	expected []string
	abort    string
	x64Only  bool
	files    map[string]string
}

var cases []testcase

func emitCase(dir string, g *gen) {
	cases = append(cases, testcase{dir: dir, src: g.source(), expected: g.names})
}

func opCase(feature, op string, t ty, build func(g *gen)) {
	g := &gen{}
	build(g)
	emitCase(feature+"_"+op+"_"+t.name, g)
}

func arith(t ty) {
	for _, op := range []string{"+", "-", "*"} {
		name := map[string]string{"+": "add", "-": "sub", "*": "mul"}[op]
		opCase("arith", name, t, func(g *gen) {
			for _, a := range t.samples() {
				for _, b := range t.samples() {
					var r *big.Int
					switch op {
					case "+":
						r = new(big.Int).Add(a, b)
					case "-":
						r = new(big.Int).Sub(a, b)
					default:
						r = new(big.Int).Mul(a, b)
					}
					g.open()
					g.decl(t, "a", a)
					g.decl(t, "b", b)
					g.line("%s r = a %s b;", t.name, op)
					g.decl(t, "e", t.wrap(r))
					g.check("r == e", fmt.Sprintf("arith_%s %s %s %s", t.name, name, tag(a), tag(b)))
				}
			}
		})
	}
}

func compare(t ty) {
	names := map[string]string{"<": "lt", "<=": "le", ">": "gt", ">=": "ge", "==": "eq", "!=": "ne"}
	for _, op := range []string{"<", "<=", ">", ">=", "==", "!="} {
		opCase("cmp", names[op], t, func(g *gen) {
			for _, a := range t.samples() {
				for _, b := range t.samples() {
					c := a.Cmp(b)
					var want bool
					switch op {
					case "<":
						want = c < 0
					case "<=":
						want = c <= 0
					case ">":
						want = c > 0
					case ">=":
						want = c >= 0
					case "==":
						want = c == 0
					default:
						want = c != 0
					}
					g.open()
					g.decl(t, "a", a)
					g.decl(t, "b", b)
					g.checkEq("a "+op+" b", want, fmt.Sprintf("cmp_%s %s %s %s", t.name, names[op], tag(a), tag(b)))
				}
			}
		})
	}
}

func divmod(t ty) {
	for _, op := range []string{"/", "%"} {
		name := map[string]string{"/": "div", "%": "mod"}[op]
		opCase("divmod", name, t, func(g *gen) {
			for _, a := range t.samples() {
				for _, b := range t.samples() {
					if b.Sign() == 0 {
						continue
					}
					if t.signed && a.Cmp(t.min()) == 0 && b.Cmp(big.NewInt(-1)) == 0 {
						continue
					}
					var r *big.Int
					if op == "/" {
						r = new(big.Int).Quo(a, b)
					} else {
						r = new(big.Int).Rem(a, b)
					}
					g.open()
					g.decl(t, "a", a)
					g.decl(t, "b", b)
					g.line("%s r = a %s b;", t.name, op)
					g.decl(t, "e", t.wrap(r))
					g.check("r == e", fmt.Sprintf("divmod_%s %s %s %s", t.name, name, tag(a), tag(b)))
				}
			}
		})
	}
}

func divOverflow(t ty) {
	opCase("divovf", "div", t, func(g *gen) {
		g.open()
		g.decl(t, "a", t.min())
		g.decl(t, "b", big.NewInt(-1))
		g.line("%s q = a / b;", t.name)
		g.decl(t, "e", t.min())
		g.check("q == e", fmt.Sprintf("divovf_%s min div m1 wraps to min", t.name))
	})
	opCase("divovf", "mod", t, func(g *gen) {
		g.open()
		g.decl(t, "a", t.min())
		g.decl(t, "b", big.NewInt(-1))
		g.line("%s r = a %% b;", t.name)
		g.decl(t, "e", big.NewInt(0))
		g.check("r == e", fmt.Sprintf("divovf_%s min mod m1 is zero", t.name))
	})
}

func shift(t ty) {
	counts := []int{0, 1, 2, t.bits - 1}
	type variant struct {
		op, name string
		lit      bool
	}
	for _, v := range []variant{{"<<", "shlvar", false}, {"<<", "shllit", true}, {">>", "shrvar", false}, {">>", "shrlit", true}} {
		opCase("shift", v.name, t, func(g *gen) {
			for _, a := range t.samples() {
				for _, n := range counts {
					var want *big.Int
					if v.op == "<<" {
						want = t.wrap(new(big.Int).Lsh(a, uint(n)))
					} else {
						want = new(big.Int).Rsh(a, uint(n))
					}
					g.open()
					g.decl(t, "a", a)
					if v.lit {
						g.line("%s r = a %s %d;", t.name, v.op, n)
					} else {
						g.decl(t, "n", big.NewInt(int64(n)))
						g.line("%s r = a %s n;", t.name, v.op)
					}
					g.decl(t, "e", want)
					g.check("r == e", fmt.Sprintf("shift_%s %s %s by %d", t.name, v.name, tag(a), n))
				}
			}
		})
	}
}

func unary(t ty) {
	opCase("unary", "neg", t, func(g *gen) {
		for _, a := range t.samples() {
			g.open()
			g.decl(t, "a", a)
			g.line("%s n = -a;", t.name)
			g.decl(t, "en", t.wrap(new(big.Int).Neg(a)))
			g.check("n == en", fmt.Sprintf("unary_%s neg %s", t.name, tag(a)))
		}
	})
	opCase("unary", "not", t, func(g *gen) {
		for _, a := range t.samples() {
			g.open()
			g.decl(t, "a", a)
			g.line("%s c = ~a;", t.name)
			g.decl(t, "ec", t.wrap(new(big.Int).Not(a)))
			g.check("c == ec", fmt.Sprintf("unary_%s not %s", t.name, tag(a)))
		}
	})
}

func bitwise(t ty) {
	for _, op := range []string{"&", "|", "^"} {
		name := map[string]string{"&": "and", "|": "or", "^": "xor"}[op]
		opCase("bitwise", name, t, func(g *gen) {
			for _, a := range t.samples() {
				for _, b := range t.samples() {
					var r *big.Int
					switch op {
					case "&":
						r = new(big.Int).And(a, b)
					case "|":
						r = new(big.Int).Or(a, b)
					default:
						r = new(big.Int).Xor(a, b)
					}
					g.open()
					g.decl(t, "a", a)
					g.decl(t, "b", b)
					g.line("%s r = a %s b;", t.name, op)
					g.decl(t, "e", t.wrap(r))
					g.check("r == e", fmt.Sprintf("bitwise_%s %s %s %s", t.name, name, tag(a), tag(b)))
				}
			}
		})
	}
}

func compound(t ty) {
	s := t.samples()
	if len(s) > 5 {
		s = s[:5]
	}
	names := map[string]string{"+=": "add", "-=": "sub", "*=": "mul", "/=": "div", "%=": "mod", "&=": "and", "|=": "or", "^=": "xor"}
	for _, op := range []string{"+=", "-=", "*=", "/=", "%=", "&=", "|=", "^="} {
		opCase("compound", names[op], t, func(g *gen) {
			for _, a := range s {
				for _, b := range s {
					if (op == "/=" || op == "%=") && (b.Sign() == 0 || (t.signed && a.Cmp(t.min()) == 0 && b.Cmp(big.NewInt(-1)) == 0)) {
						continue
					}
					var r *big.Int
					switch op {
					case "+=":
						r = new(big.Int).Add(a, b)
					case "-=":
						r = new(big.Int).Sub(a, b)
					case "*=":
						r = new(big.Int).Mul(a, b)
					case "/=":
						r = new(big.Int).Quo(a, b)
					case "%=":
						r = new(big.Int).Rem(a, b)
					case "&=":
						r = new(big.Int).And(a, b)
					case "|=":
						r = new(big.Int).Or(a, b)
					default:
						r = new(big.Int).Xor(a, b)
					}
					g.open()
					g.decl(t, "a", a)
					g.decl(t, "b", b)
					g.line("a %s b;", op)
					g.decl(t, "e", t.wrap(r))
					g.check("a == e", fmt.Sprintf("compound_%s %s %s %s", t.name, names[op], tag(a), tag(b)))
				}
			}
		})
	}
	for _, left := range []bool{true, false} {
		name, op := "shr", ">>="
		if left {
			name, op = "shl", "<<="
		}
		opCase("compound", name, t, func(g *gen) {
			for _, a := range s {
				for _, n := range []int{1, t.bits - 1} {
					var want *big.Int
					if left {
						want = t.wrap(new(big.Int).Lsh(a, uint(n)))
					} else {
						want = new(big.Int).Rsh(a, uint(n))
					}
					g.open()
					g.decl(t, "a", a)
					g.decl(t, "n", big.NewInt(int64(n)))
					g.line("a %s n;", op)
					g.decl(t, "e", want)
					g.check("a == e", fmt.Sprintf("compound_%s %s %s by %d", t.name, name, tag(a), n))
				}
			}
		})
	}
}

func incdec(t ty) {
	opCase("incdec", "inc", t, func(g *gen) {
		for _, a := range t.samples() {
			g.open()
			g.decl(t, "a", a)
			g.line("a++;")
			g.decl(t, "e", t.wrap(new(big.Int).Add(a, big.NewInt(1))))
			g.check("a == e", fmt.Sprintf("incdec_%s inc %s", t.name, tag(a)))
		}
	})
	opCase("incdec", "dec", t, func(g *gen) {
		for _, a := range t.samples() {
			g.open()
			g.decl(t, "a", a)
			g.line("a--;")
			g.decl(t, "e", t.wrap(new(big.Int).Sub(a, big.NewInt(1))))
			g.check("a == e", fmt.Sprintf("incdec_%s dec %s", t.name, tag(a)))
		}
	})
}

func cast(src ty) {
	for _, dst := range types {
		opCase("cast", dst.name, src, func(g *gen) {
			for _, v := range src.samples() {
				g.open()
				g.decl(src, "a", v)
				g.line("%s r = %s(a);", dst.name, dst.name)
				g.decl(dst, "e", dst.wrap(v))
				g.check("r == e", fmt.Sprintf("cast_%s to %s %s", src.name, dst.name, tag(v)))
			}
		})
	}
}

var checkName = regexp.MustCompile(`(?m)^\s*(?:check|checkEq)\(.*"([^"]+)"\);\s*$`)

func main() {
	root := "tests/numeric"
	if len(os.Args) > 1 {
		root = os.Args[1]
	}

	for _, t := range types {
		arith(t)
		compare(t)
		divmod(t)
		shift(t)
		unary(t)
		bitwise(t)
		compound(t)
		incdec(t)
		cast(t)
		if t.signed {
			divOverflow(t)
		}
	}

	for _, m := range manual {
		exp := m.expected
		if exp == nil {
			body := m.src[strings.Index(m.src, "func main()"):]
			for _, match := range checkName.FindAllStringSubmatch(body, -1) {
				exp = append(exp, match[1])
			}
		}
		cases = append(cases, testcase{dir: m.dir, src: header + m.src, expected: exp, abort: m.abort, x64Only: m.x64Only, files: m.files})
	}

	os.RemoveAll(root)
	for _, c := range cases {
		dir := filepath.Join(root, c.dir)
		if err := os.MkdirAll(dir, 0755); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if err := os.WriteFile(filepath.Join(dir, "main.wsp"), []byte(c.src), 0644); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		expected := strings.Join(c.expected, "\n")
		if expected != "" {
			expected += "\n"
		}
		if err := os.WriteFile(filepath.Join(dir, "expected.txt"), []byte(expected), 0644); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		for name, content := range c.files {
			if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
		}
		if c.x64Only {
			if err := os.WriteFile(filepath.Join(dir, "x64-only"), nil, 0644); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
		}
		if c.abort != "" {
			if err := os.WriteFile(filepath.Join(dir, "abort.txt"), []byte(c.abort+"\n"), 0644); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
		}
	}
	fmt.Printf("generated %d cases in %s\n", len(cases), root)
}
