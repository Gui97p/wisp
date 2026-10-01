package target

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

type Target struct {
	Name string
	Arch string
	OS   string

	ABI         string
	Format      string
	Entry       string
	Runtime     string
	StartSource string
	LinkArgs    []string

	Builtin bool
}

const linuxStart = `global _start
extern main

section .text
_start:
	xor ebp, ebp
	call main
	mov rdi, rax
	mov eax, 60
	syscall

section .note.GNU-stack noalloc noexec nowrite progbits
`

const sineStart = `global _start
extern main

section .text
_start:
	xor ebp, ebp
	call main
	mov rax, 19
	mov rdi, 3
	syscall

section .note.GNU-stack noalloc noexec nowrite progbits
`

var builtin = []Target{
	{Name: "linux_x64", Arch: "x64", OS: "linux", ABI: "sysv", Format: "elf64", Entry: "_start", Runtime: "std/runtime", StartSource: linuxStart, LinkArgs: []string{"-z", "noexecstack"}, Builtin: true},
	{Name: "sine_x64", Arch: "x64", OS: "sine", ABI: "sysv", Format: "elf64", Entry: "_start", Runtime: "std/runtime", StartSource: sineStart, LinkArgs: []string{"-z", "noexecstack"}, Builtin: true},
	{Name: "lua", Arch: "lua", Builtin: true},
}

var project []Target

var (
	namePattern  = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)
	entryPattern = regexp.MustCompile(`^[A-Za-z_.$][A-Za-z0-9_.$]*$`)
)

func all() []Target {
	return append(append([]Target{}, builtin...), project...)
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
	if t.Runtime == "" {
		t.Runtime = "std/runtime"
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

	return Target{}, fmt.Errorf("unknown target %q%s", name, hint(name, Names()))
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

	return fmt.Errorf("unknown target %q%s", selector, hint(selector, Selectors()))
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

func hint(name string, candidates []string) string {
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
