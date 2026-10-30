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

	Builtin  bool
	Modified bool
}

type FieldError struct {
	Target  string
	Field   string
	Message string
	Help    string
}

func (e *FieldError) Summary() string {
	return fmt.Sprintf("target %q: %s", e.Target, e.Message)
}

func (e *FieldError) Error() string {
	if e.Help == "" {
		return e.Summary()
	}
	return fmt.Sprintf("%s (%s)", e.Summary(), e.Help)
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
	{Name: "cc", Arch: "lua", Builtin: true},
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

func List() []Target {
	return all()
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
		return &FieldError{Target: name, Message: "is not a built-in target"}
	}

	if patch.Arch != "" && patch.Arch != base.Arch {
		return &FieldError{Target: name, Field: "arch", Message: "the arch of a built-in target cannot be changed"}
	}

	if base.Arch != "x64" {
		if patch.OS != "" || patch.ABI != "" || patch.Format != "" || patch.Entry != "" || patch.StartSource != "" || patch.RuntimeSource != "" || len(patch.LinkArgs) > 0 || patch.Libc != nil {
			return &FieldError{Target: name, Message: "only accepts \"output\""}
		}
	}

	t := base
	if patch.OS != "" {
		if !namePattern.MatchString(patch.OS) {
			return &FieldError{Target: name, Field: "os", Message: fmt.Sprintf("invalid os %q", patch.OS), Help: "use lowercase letters, digits and underscores"}
		}
		t.OS = patch.OS
	}
	if patch.ABI != "" {
		if patch.ABI != "sysv" {
			return &FieldError{Target: name, Field: "abi", Message: fmt.Sprintf("unsupported abi %q", patch.ABI), Help: "only sysv exists"}
		}
		t.ABI = patch.ABI
	}
	if patch.Format != "" {
		if patch.Format != "elf64" {
			return &FieldError{Target: name, Field: "format", Message: fmt.Sprintf("unsupported format %q", patch.Format), Help: "only elf64 exists"}
		}
		t.Format = patch.Format
	}
	if patch.Entry != "" {
		if !entryPattern.MatchString(patch.Entry) {
			return &FieldError{Target: name, Field: "entry", Message: fmt.Sprintf("invalid entry symbol %q", patch.Entry), Help: "use letters, digits and the characters _ . and $, not starting with a digit"}
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
	t.Modified = true

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
		return &FieldError{Target: name, Message: "the name is not valid", Help: "use lowercase letters, digits and underscores, starting with a letter"}
	}

	for _, b := range builtin {
		if name == b.Name || name == b.Arch || (b.OS != "" && name == b.OS) {
			return &FieldError{Target: name, Message: "the name is already used by a built-in target or selector", Help: "pick another name"}
		}
	}

	return nil
}

func Register(t Target) error {
	if err := CheckName(t.Name); err != nil {
		return err
	}
	if t.OS != "" && !namePattern.MatchString(t.OS) {
		return &FieldError{Target: t.Name, Field: "os", Message: fmt.Sprintf("invalid os %q", t.OS), Help: "use lowercase letters, digits and underscores"}
	}
	if t.Arch != "x64" {
		return &FieldError{Target: t.Name, Field: "arch", Message: fmt.Sprintf("unsupported arch %q", t.Arch), Help: "only x64 exists for now"}
	}
	if t.ABI == "" {
		t.ABI = "sysv"
	}
	if t.ABI != "sysv" {
		return &FieldError{Target: t.Name, Field: "abi", Message: fmt.Sprintf("unsupported abi %q", t.ABI), Help: "only sysv exists"}
	}
	if t.Format == "" {
		t.Format = "elf64"
	}
	if t.Format != "elf64" {
		return &FieldError{Target: t.Name, Field: "format", Message: fmt.Sprintf("unsupported format %q", t.Format), Help: "only elf64 exists"}
	}
	if t.Entry == "" {
		t.Entry = "_start"
	}
	if !entryPattern.MatchString(t.Entry) {
		return &FieldError{Target: t.Name, Field: "entry", Message: fmt.Sprintf("invalid entry symbol %q", t.Entry), Help: "use letters, digits and the characters _ . and $, not starting with a digit"}
	}
	if strings.TrimSpace(t.StartSource) == "" {
		return &FieldError{Target: t.Name, Field: "start", Message: "\"start\" is required", Help: "it is the assembly file that calls main and leaves the program the way this kernel requires"}
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

	return Target{}, &UnknownError{Name: name, Help: Suggestion(name, Names(), "targets")}
}

func (t Target) Matches(selector string) bool {
	return selector == t.Name || selector == t.Arch || (t.OS != "" && selector == t.OS)
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

	return &UnknownError{Name: selector, Help: Suggestion(selector, Selectors(), "targets")}
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

func Enumerate(names []string) string {
	switch len(names) {
	case 0:
		return ""
	case 1:
		return names[0]
	}
	return strings.Join(names[:len(names)-1], ", ") + " and " + names[len(names)-1]
}

func Suggestion(name string, candidates []string, noun string) string {
	best, bestDist := "", 3
	for _, c := range candidates {
		if d := distance(name, c); d < bestDist {
			best, bestDist = c, d
		}
	}

	known := fmt.Sprintf("the known %s are %s", noun, Enumerate(candidates))
	if best != "" {
		return fmt.Sprintf("did you mean %s? %s", best, known)
	}
	return known
}

type UnknownError struct {
	Name string
	Help string
}

func (e *UnknownError) Error() string {
	return fmt.Sprintf("unknown target %q (%s)", e.Name, e.Help)
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
