package target

import (
	_ "embed"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

type Target struct {
	Name string
	Arch string
	OS   string

	ABI           string
	Format        string
	Entry         string
	StartSource   string
	RuntimeSource string
	LinkArgs      []string
	Libc          *bool
	Output        Output

	Builtin bool
}

type Output struct {
	Bin string `toml:"bin" doc:"Path of the linked executable."`
	Obj string `toml:"obj" doc:"Path of the object file."`
	Asm string `toml:"asm" doc:"Path of the generated assembly."`
	Lua string `toml:"lua" doc:"Path of the generated Lua file."`
}

type Patch struct {
	Arch          string
	OS            string
	ABI           string
	Format        string
	Entry         string
	StartSource   string
	RuntimeSource string
	LinkArgs      []string
	Libc          *bool
	Output        Output
}

//go:embed asm/linux_x64_start.asm
var linuxStart string

//go:embed asm/linux_x64_runtime.asm
var linuxRuntime string

var builtin = []Target{
	{Name: "linux_x64", Arch: "x64", OS: "linux", ABI: "sysv", Format: "elf64", Entry: "_start", StartSource: linuxStart, RuntimeSource: linuxRuntime, LinkArgs: []string{"-z", "noexecstack"}, Builtin: true},
	{Name: "lua", Arch: "lua", Builtin: true},
}

var project []Target

var (
	namePattern  = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)
	entryPattern = regexp.MustCompile(`^[A-Za-z_.$][A-Za-z0-9_.$]*$`)
)

func all() []Target {
	out := make([]Target, 0, len(builtin)+len(project))
	for _, b := range builtin {
		out = append(out, b)
		for _, p := range project {
			if p.Name == b.Name {
				out[len(out)-1] = p
			}
		}
	}
	for _, p := range project {
		if !isBuiltinName(p.Name) {
			out = append(out, p)
		}
	}
	return out
}

func isBuiltinName(name string) bool {
	for _, b := range builtin {
		if b.Name == name {
			return true
		}
	}
	return false
}

func IsBuiltin(name string) bool {
	return isBuiltinName(name)
}

func ResetProject() {
	project = nil
}

func Override(name string, patch Patch) error {
	var base Target
	found := false
	for _, b := range builtin {
		if b.Name == name {
			base, found = b, true
		}
	}
	if !found {
		return fmt.Errorf("target %q is not a built-in target", name)
	}

	if patch.Arch != "" && patch.Arch != base.Arch {
		return fmt.Errorf("target %q: the arch of a built-in target cannot be changed", name)
	}

	if base.Arch != "x64" {
		if patch.OS != "" || patch.ABI != "" || patch.Format != "" || patch.Entry != "" || patch.StartSource != "" || patch.RuntimeSource != "" || len(patch.LinkArgs) > 0 || patch.Libc != nil {
			return fmt.Errorf("target %q only accepts \"output\"", name)
		}
	}

	t := base
	if patch.OS != "" {
		if !namePattern.MatchString(patch.OS) {
			return fmt.Errorf("target %q: invalid os %q (lowercase letters, digits and underscores)", name, patch.OS)
		}
		t.OS = patch.OS
	}
	if patch.ABI != "" {
		if patch.ABI != "sysv" {
			return fmt.Errorf("target %q: unsupported abi %q (only \"sysv\")", name, patch.ABI)
		}
		t.ABI = patch.ABI
	}
	if patch.Format != "" {
		if patch.Format != "elf64" {
			return fmt.Errorf("target %q: unsupported format %q (only \"elf64\")", name, patch.Format)
		}
		t.Format = patch.Format
	}
	if patch.Entry != "" {
		if !entryPattern.MatchString(patch.Entry) {
			return fmt.Errorf("target %q: invalid entry symbol %q", name, patch.Entry)
		}
		t.Entry = patch.Entry
	}
	if patch.StartSource != "" {
		t.StartSource = patch.StartSource
	}
	if patch.RuntimeSource != "" {
		t.RuntimeSource = patch.RuntimeSource
	}
	t.LinkArgs = append(append([]string{}, base.LinkArgs...), patch.LinkArgs...)
	if patch.Libc != nil {
		t.Libc = patch.Libc
	}
	t.Output = patch.Output

	for i, p := range project {
		if p.Name == name {
			project[i] = t
			return nil
		}
	}
	project = append(project, t)

	return nil
}

func CheckName(name string) error {
	if !namePattern.MatchString(name) {
		return fmt.Errorf("target %q: the name must be lowercase letters, digits and underscores, starting with a letter", name)
	}

	for _, b := range builtin {
		if name == b.Name || name == b.Arch || (b.OS != "" && name == b.OS) {
			return fmt.Errorf("target %q: the name is already used by a built-in target or selector", name)
		}
	}

	return nil
}

func Register(t Target) error {
	if err := CheckName(t.Name); err != nil {
		return err
	}
	if t.OS != "" && !namePattern.MatchString(t.OS) {
		return fmt.Errorf("target %q: invalid os %q (lowercase letters, digits and underscores)", t.Name, t.OS)
	}
	if t.Arch != "x64" {
		return fmt.Errorf("target %q: unsupported arch %q (only \"x64\")", t.Name, t.Arch)
	}
	if t.ABI == "" {
		t.ABI = "sysv"
	}
	if t.ABI != "sysv" {
		return fmt.Errorf("target %q: unsupported abi %q (only \"sysv\")", t.Name, t.ABI)
	}
	if t.Format == "" {
		t.Format = "elf64"
	}
	if t.Format != "elf64" {
		return fmt.Errorf("target %q: unsupported format %q (only \"elf64\")", t.Name, t.Format)
	}
	if t.Entry == "" {
		t.Entry = "_start"
	}
	if !entryPattern.MatchString(t.Entry) {
		return fmt.Errorf("target %q: invalid entry symbol %q", t.Name, t.Entry)
	}
	if strings.TrimSpace(t.StartSource) == "" {
		return fmt.Errorf("target %q: \"start\" is required, the assembly that calls main and leaves the program for this kernel", t.Name)
	}

	t.Builtin = false
	for i, p := range project {
		if p.Name == t.Name {
			project[i] = t
			return nil
		}
	}
	project = append(project, t)

	return nil
}

func (t Target) CompileKey() string {
	return strings.Join([]string{t.Name, t.Arch, t.OS, t.ABI, t.Format}, "|")
}

func Lookup(name string) (Target, error) {
	for _, t := range all() {
		if t.Name == name {
			return t, nil
		}
	}

	return Target{}, fmt.Errorf("unknown target %q%s", name, Hint(name, Names()))
}

func (t Target) Matches(selector string) bool {
	return selector == t.Name || selector == t.Arch || (t.OS != "" && selector == t.OS)
}

func CheckNative(targetName string, selectors []string) error {
	if len(selectors) == 0 {
		return nil
	}

	var named []string
	for _, sel := range selectors {
		if sel == "" || MatchesName(targetName, sel) {
			return nil
		}
		named = append(named, sel)
	}

	return fmt.Errorf("has native blocks for [%s] but none for target %q", strings.Join(named, ", "), targetName)
}

func MatchesName(targetName, selector string) bool {
	t, err := Lookup(targetName)
	if err != nil {
		return false
	}

	return t.Matches(selector)
}

func ValidateSelector(selector string) error {
	for _, t := range all() {
		if t.Matches(selector) {
			return nil
		}
	}

	return fmt.Errorf("unknown target %q%s", selector, Hint(selector, Selectors()))
}

func Names() []string {
	names := make([]string, 0, len(all()))
	for _, t := range all() {
		names = append(names, t.Name)
	}
	sort.Strings(names)

	return names
}

func Selectors() []string {
	seen := map[string]bool{}
	var selectors []string
	for _, t := range all() {
		for _, s := range []string{t.Name, t.Arch, t.OS} {
			if s != "" && !seen[s] {
				seen[s] = true
				selectors = append(selectors, s)
			}
		}
	}
	sort.Strings(selectors)

	return selectors
}

func Hint(name string, candidates []string) string {
	best, bestDist := "", 3
	for _, c := range candidates {
		if d := distance(name, c); d < bestDist {
			best, bestDist = c, d
		}
	}

	msg := fmt.Sprintf(" (known: %s)", strings.Join(candidates, ", "))
	if best != "" {
		msg = fmt.Sprintf(" (did you mean %q? known: %s)", best, strings.Join(candidates, ", "))
	}

	return msg
}

func distance(a, b string) int {
	prev := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}

	for i := 1; i <= len(a); i++ {
		cur := make([]int, len(b)+1)
		cur[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			cur[j] = min(prev[j]+1, cur[j-1]+1, prev[j-1]+cost)
		}
		prev = cur
	}

	return prev[len(b)]
}
