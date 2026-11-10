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
	Arch    string       `toml:"arch" doc:"CPU architecture of a system of your own that has just one. A system with several writes a [targets.<os>.<arch>] table for each instead."`
	ABI     string       `toml:"abi" doc:"Calling convention. Only sysv is supported, and it is the default."`
	Format  string       `toml:"format" doc:"Object format. Only elf64 is supported, and it is the default."`
	Entry   string       `toml:"entry" doc:"Symbol where the linker starts the program. Defaults to _start."`
	Runtime string       `toml:"runtime" doc:"Assembly file with the runtime routines (allocation, panic, I/O) for this system."`
	Start   string       `toml:"start" doc:"Assembly file that calls main and leaves the program. Required for systems of your own."`
	Link    []string     `toml:"link" doc:"Extra linker arguments, such as -T linker.ld. On a built-in system they are added to the built-in ones."`
	Libc    *bool        `toml:"libc" doc:"Link against the C library for this system. Overrides the top-level libc."`
	Output  OutputConfig `toml:"output" doc:"Where this system writes its artifacts. Overrides the top-level [output]."`

	Archs map[string]ArchConfig `toml:"-"`
}

type ArchConfig struct {
	ABI     string       `toml:"abi" doc:"Calling convention. Only sysv is supported, and it is the default."`
	Format  string       `toml:"format" doc:"Object format. Only elf64 is supported, and it is the default."`
	Entry   string       `toml:"entry" doc:"Symbol where the linker starts the program. Defaults to _start."`
	Runtime string       `toml:"runtime" doc:"Assembly file with the runtime routines for this architecture of the system."`
	Start   string       `toml:"start" doc:"Assembly file that calls main and leaves the program, for this architecture."`
	Link    []string     `toml:"link" doc:"Extra linker arguments for this architecture, added to the ones of the system."`
	Libc    *bool        `toml:"libc" doc:"Link against the C library for this architecture."`
	Output  OutputConfig `toml:"output" doc:"Where this architecture writes its artifacts."`
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
	case "targets.output", "targets.arch.output":
		t = reflect.TypeOf(OutputConfig{})
	case "targets.arch":
		t = reflect.TypeOf(ArchConfig{})
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

func readSource(root, name, field, file string) (string, error) {
	if file == "" {
		return "", nil
	}
	data, err := os.ReadFile(filepath.Join(root, file))
	if err != nil {
		return "", &target.FieldError{Target: name, Field: field, Message: fmt.Sprintf("cannot read the %s file", field), Help: fmt.Sprintf("%s could not be opened, the system reports %s", file, diag.Reason(err))}
	}
	return string(data), nil
}

type layer struct {
	abi, format, entry, start, runtime string
	link                               []string
	libc                               *bool
	output                             OutputConfig
}

func (c *Config) layerOf(root, name string, abi, format, entry, start, runtimeFile string, link []string, libc *bool, output OutputConfig) (layer, error) {
	startSource, err := readSource(root, name, "start", start)
	if err != nil {
		return layer{}, err
	}
	runtimeSource, err := readSource(root, name, "runtime", runtimeFile)
	if err != nil {
		return layer{}, err
	}
	return layer{abi: abi, format: format, entry: entry, start: startSource, runtime: runtimeSource, link: link, libc: libc, output: output}, nil
}

func merge(base, top layer) layer {
	out := base
	if top.abi != "" {
		out.abi = top.abi
	}
	if top.format != "" {
		out.format = top.format
	}
	if top.entry != "" {
		out.entry = top.entry
	}
	if top.start != "" {
		out.start = top.start
	}
	if top.runtime != "" {
		out.runtime = top.runtime
	}
	out.link = append(append([]string{}, base.link...), top.link...)
	if top.libc != nil {
		out.libc = top.libc
	}
	if top.output.Bin != "" {
		out.output.Bin = top.output.Bin
	}
	if top.output.Obj != "" {
		out.output.Obj = top.output.Obj
	}
	if top.output.Asm != "" {
		out.output.Asm = top.output.Asm
	}
	return out
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

		system, err := c.layerOf(root, name, tc.ABI, tc.Format, tc.Entry, tc.Start, tc.Runtime, tc.Link, tc.Libc, tc.Output)
		if err != nil {
			return err
		}

		archLayers := map[string]layer{}
		for arch, ac := range tc.Archs {
			l, err := c.layerOf(root, name+"."+arch, ac.ABI, ac.Format, ac.Entry, ac.Start, ac.Runtime, ac.Link, ac.Libc, ac.Output)
			if err != nil {
				return err
			}
			archLayers[arch] = l
		}

		if target.IsBuiltinOS(name) {
			if tc.Arch != "" {
				return &target.FieldError{Target: name, Field: "arch", Message: "the architecture of a built-in system cannot be set here", Help: "use [targets." + name + ".<arch>] to adjust one of its architectures"}
			}
			known := target.BuiltinArchsOf(name)
			for arch := range archLayers {
				if !slices.Contains(known, arch) {
					return archError(name, arch, known)
				}
			}
			for _, arch := range known {
				l := merge(system, archLayers[arch])
				err := target.Override(name, arch, target.Patch{
					ABI: l.abi, Format: l.format, Entry: l.entry,
					StartSource: l.start, RuntimeSource: l.runtime,
					LinkArgs: l.link, Libc: l.libc, Output: l.output,
				})
				if err != nil {
					return err
				}
			}
			continue
		}

		archs := make([]string, 0, len(archLayers)+1)
		for arch := range archLayers {
			archs = append(archs, arch)
		}
		if tc.Arch != "" && !slices.Contains(archs, tc.Arch) {
			archs = append(archs, tc.Arch)
		}
		sort.Strings(archs)
		if len(archs) == 0 {
			return &target.FieldError{Target: name, Field: "arch", Message: "\"arch\" is required", Help: "write arch = \"x64\", or a [targets." + name + ".<arch>] table for each architecture"}
		}

		for _, arch := range archs {
			l := merge(system, archLayers[arch])
			err := target.Register(target.Target{
				OS: name, Arch: arch,
				ABI: l.abi, Format: l.format, Entry: l.entry,
				StartSource: l.start, RuntimeSource: l.runtime,
				LinkArgs: l.link, Libc: l.libc, Output: l.output,
			})
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func archError(osName, arch string, known []string) error {
	if !slices.Contains(target.Architectures(), arch) {
		return &target.FieldError{Target: osName, Field: arch, Message: fmt.Sprintf("there is no backend for the architecture %q", arch), Help: fmt.Sprintf("the architectures that exist are %s", target.Enumerate(target.Architectures()))}
	}
	return &target.FieldError{Target: osName, Field: arch, Message: fmt.Sprintf("the %s system has no %s target", osName, arch), Help: fmt.Sprintf("it has %s", target.Enumerate(known))}
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

		if isArchTable(md, key) {
			continue
		}

		known := knownKeys(key)
		issues = append(issues, ConfigIssue{
			Line:    locateKey(text, key),
			Message: fmt.Sprintf("unknown key %q", key.String()),
			Help:    target.Suggestion(key[len(key)-1], known, "keys"),
		})
	}
	if err := decodeArchTables(text, md, &cfg, &issues); err != nil {
		return nil, err
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

func isArchTable(md toml.MetaData, key toml.Key) bool {
	if len(key) < 3 || key[0] != "targets" {
		return false
	}
	if slices.Contains(tomlTags(reflect.TypeOf(TargetConfig{})), key[2]) {
		return false
	}
	return md.Type(key[0], key[1], key[2]) == "Hash"
}

func decodeArchTables(text string, md toml.MetaData, cfg *Config, issues *[]ConfigIssue) error {
	var raw struct {
		Targets map[string]map[string]toml.Primitive
	}
	rawMeta, err := toml.Decode(text, &raw)
	if err != nil {
		return nil
	}

	tables := map[string]bool{}
	for osName, entries := range raw.Targets {
		for key, prim := range entries {
			if !isArchTable(md, toml.Key{"targets", osName, key}) {
				continue
			}
			var ac ArchConfig
			if err := rawMeta.PrimitiveDecode(prim, &ac); err != nil {
				*issues = append(*issues, ConfigIssue{Line: locateKey(text, toml.Key{"targets", osName, key}), Message: diag.Reason(err)})
				continue
			}
			tc := cfg.Targets[osName]
			if tc.Archs == nil {
				tc.Archs = map[string]ArchConfig{}
			}
			tc.Archs[key] = ac
			cfg.Targets[osName] = tc
			tables[osName+"."+key] = true
		}
	}

	var reported []toml.Key
	for _, key := range rawMeta.Undecoded() {
		if len(key) < 4 || key[0] != "targets" || !tables[key[1]+"."+key[2]] || coveredBy(key, reported) {
			continue
		}
		reported = append(reported, key)
		*issues = append(*issues, ConfigIssue{
			Line:    locateKey(text, key),
			Message: fmt.Sprintf("unknown key %q", key.String()),
			Help:    target.Suggestion(key[len(key)-1], tomlTags(reflect.TypeOf(ArchConfig{})), "keys"),
		})
	}
	return nil
}
