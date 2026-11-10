package target

import (
	_ "embed"
	"fmt"
	"regexp"
	"runtime"
	"slices"
	"sort"
	"strings"
	"sync"
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
	{Name: "linux/x64", Arch: "x64", OS: "linux", ABI: "sysv", Format: "elf64", Entry: "_start", StartSource: linuxStart, RuntimeSource: linuxRuntime, LinkArgs: []string{"-z", "noexecstack"}, Builtin: true},
	{Name: "cc/lua", Arch: "lua", OS: "cc", Builtin: true},
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
		if !IsBuiltinPair(p.OS, p.Arch) {
			out = append(out, p)
		}
	}
	return out
}

func List() []Target {
	return all()
}

func IsBuiltinPair(osName, arch string) bool {
	name := pairName(osName, arch)
	for _, b := range builtin {
		if b.Name == name {
			return true
		}
	}
	return false
}

func IsBuiltinOS(osName string) bool {
	for _, b := range builtin {
		if b.OS == osName {
			return true
		}
	}
	return false
}

func ArchsOf(osName string) []string {
	var archs []string
	for _, t := range all() {
		if t.OS == osName {
			archs = append(archs, t.Arch)
		}
	}
	sort.Strings(archs)
	return archs
}

func BuiltinArchsOf(osName string) []string {
	var archs []string
	for _, b := range builtin {
		if b.OS == osName {
			archs = append(archs, b.Arch)
		}
	}
	sort.Strings(archs)
	return archs
}

func ResetProject() {
	project = nil
}

func pairName(osName, arch string) string {
	return osName + "/" + arch
}

func Override(osName, arch string, patch Patch) error {
	name := pairName(osName, arch)
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

	if base.Arch == "lua" {
		if patch.ABI != "" || patch.Format != "" || patch.Entry != "" || patch.StartSource != "" || patch.RuntimeSource != "" || len(patch.LinkArgs) > 0 || patch.Libc != nil {
			return &FieldError{Target: name, Message: "only accepts \"output\""}
		}
	}

	t := base
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
		if name == b.OS || name == b.Arch {
			return &FieldError{Target: name, Message: "the name is already used by a built-in system or architecture", Help: "pick another name"}
		}
	}

	return nil
}

func Register(t Target) error {
	if err := CheckName(t.OS); err != nil {
		return err
	}
	t.Name = pairName(t.OS, t.Arch)
	if !slices.Contains(assemblyArchs, t.Arch) {
		return &FieldError{Target: t.Name, Field: "arch", Message: fmt.Sprintf("there is no backend for the architecture %q", t.Arch), Help: fmt.Sprintf("the architectures a system of your own can use are %s", Enumerate(assemblyArchs))}
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

var assemblyArchs = []string{"x64"}

func hostArch() string {
	switch runtime.GOARCH {
	case "amd64":
		return "x64"
	}
	return runtime.GOARCH
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

func Resolve(osName, arch string) (Target, error) {
	var candidates []Target
	for _, t := range all() {
		if t.OS == osName {
			candidates = append(candidates, t)
		}
	}
	if len(candidates) == 0 {
		return Target{}, &UnknownError{Name: osName, Help: Suggestion(osName, OSNames(), "systems")}
	}

	if arch != "" {
		for _, t := range candidates {
			if t.Arch == arch {
				return t, nil
			}
		}
		return Target{}, fmt.Errorf("the %s system has no %s target (it has %s)", osName, arch, Enumerate(ArchsOf(osName)))
	}

	if len(candidates) == 1 {
		return candidates[0], nil
	}
	for _, t := range candidates {
		if t.Arch == hostArch() {
			return t, nil
		}
	}
	return Target{}, fmt.Errorf("the %s system has more than one architecture (%s): choose one with arch or --arch", osName, Enumerate(ArchsOf(osName)))
}

func (t Target) Matches(selector string) bool {
	return selector == t.Arch || selector == t.OS
}

func MatchesName(targetName, selector string) bool {
	return MatchesText(targetName, selector)
}

func ValidateSelector(selector string) error {
	for _, t := range all() {
		if t.Matches(selector) {
			return nil
		}
	}

	return &UnknownError{Name: selector, Help: Suggestion(selector, Selectors(), "architectures and systems")}
}

func Architectures() []string {
	seen := map[string]bool{}
	var archs []string
	for _, t := range all() {
		if t.Arch != "" && !seen[t.Arch] {
			seen[t.Arch] = true
			archs = append(archs, t.Arch)
		}
	}
	sort.Strings(archs)

	return archs
}

func ValidateArch(arch string) error {
	archs := Architectures()
	for _, a := range archs {
		if a == arch {
			return nil
		}
	}

	return &UnknownError{Name: arch, Help: Suggestion(arch, archs, "architectures")}
}

func ActiveArch() string {
	t, err := Lookup(Active())
	if err != nil {
		return ""
	}
	return t.Arch
}

var (
	activeMu   sync.RWMutex
	activeName string
)

func SetActive(name string) {
	activeMu.Lock()
	defer activeMu.Unlock()
	activeName = name
}

func Active() string {
	activeMu.RLock()
	defer activeMu.RUnlock()
	if activeName == "" {
		return pairName(runtime.GOOS, hostArch())
	}
	return activeName
}

func Names() []string {
	names := make([]string, 0, len(all()))
	for _, t := range all() {
		names = append(names, t.Name)
	}
	sort.Strings(names)

	return names
}

func OSNames() []string {
	seen := map[string]bool{}
	var names []string
	for _, t := range all() {
		if !seen[t.OS] {
			seen[t.OS] = true
			names = append(names, t.OS)
		}
	}
	sort.Strings(names)

	return names
}

func Selectors() []string {
	seen := map[string]bool{}
	var selectors []string
	for _, t := range all() {
		for _, s := range []string{t.Arch, t.OS} {
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
	return fmt.Sprintf("unknown system %q (%s)", e.Name, e.Help)
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
