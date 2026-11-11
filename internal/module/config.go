package module

import (
	"errors"
	"fmt"
	"maps"
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
	Default   string                   `toml:"default" doc:"The program that wisp run and wisp env use when no name is given. Without it a project with several programs asks for the name."`
	OS        string                   `toml:"os" doc:"The system to build for: linux, cc, or a name declared in [targets.<name>]. Defaults to the system of the machine."`
	Arch      string                   `toml:"arch" doc:"The architecture to build for. Only needed when the system has more than one; otherwise it is a check."`
	Std       string                   `toml:"std" doc:"The standard library. Leave it out for the one that ships with the compiler, write a folder to replace it, or \"none\" to build without any (a folder called none is written ./none)."`
	Link      []string                 `toml:"link" doc:"Extra arguments passed to the linker (libraries, object files, flags)."`
	Libc      bool                     `toml:"libc" doc:"Link against the C library instead of the Wisp runtime start code."`
	Output    OutputConfig             `toml:"output" doc:"Where build artifacts are written."`
	Run       RunConfig                `toml:"run" doc:"How wisp run starts the program."`
	Programs  map[string]ProgramConfig `toml:"program" doc:"The programs of the project, declared as [program.<name>]. Without any, the project is one program named after its folder."`
	Libraries map[string]LibraryConfig `toml:"library" doc:"The libraries of the project, declared as [library.<name>]. A library has no entry: it is the modules it lists and everything they import."`
	Targets   map[string]TargetConfig  `toml:"targets" doc:"Custom targets, declared as [targets.<name>]. Used for kernels and other bare-metal environments."`
	Profile   string                   `toml:"profile" doc:"The profile to build with: debug (the default) or release. A program or library can choose its own, and --profile overrides it."`
	Profiles  map[string]ProfileConfig `toml:"profiles" doc:"Adjusts what debug or release turn on, declared as [profiles.debug] or [profiles.release]. Only the keys that differ from the built-in profile are written."`

	Entry         string       `toml:"-"`
	ProgramName   string       `toml:"-"`
	BinaryName    string       `toml:"-"`
	ProgramLink   []string     `toml:"-"`
	ProgramLibc   *bool        `toml:"-"`
	ProgramOutput OutputConfig `toml:"-"`
}

type ProfileConfig struct {
	Trace    *bool `toml:"trace" doc:"Keep the call stack information that panics and unhandled errors print as a stack trace (x86-64)."`
	Snippets *bool `toml:"snippets" doc:"Show the line of source code in the blocks that report an error at run time."`
	Strip    *bool `toml:"strip" doc:"Remove the symbol table from the linked binary (x86-64)."`
	Optimize *bool `toml:"optimize" doc:"Let the compiler spend effort on faster code, such as keeping locals in registers."`
	Checks   *bool `toml:"checks" doc:"Keep the run-time safety checks: index bounds, division by zero and null pointers."`
}

type LibraryConfig struct {
	Modules []string     `toml:"modules" doc:"The modules the library is made of, by import path, such as \"geometry\". Whatever they import goes along."`
	OS      string       `toml:"os" doc:"The system to build this library for. Defaults to the top-level os."`
	Arch    string       `toml:"arch" doc:"The architecture to build this library for."`
	Output  OutputConfig `toml:"output" doc:"Where this library writes its artifacts. {profile}, {target}, {os}, {arch} and {name} are replaced."`
	Profile string       `toml:"profile" doc:"The profile to build this library with. Defaults to the top-level profile."`
}

type ProgramConfig struct {
	Name    string       `toml:"name" doc:"Name of the binary this program produces. Defaults to the name of the program."`
	Entry   string       `toml:"entry" doc:"Entry file of this program, relative to the project root."`
	OS      string       `toml:"os" doc:"The system to build this program for. Defaults to the top-level os."`
	Arch    string       `toml:"arch" doc:"The architecture to build this program for."`
	Link    []string     `toml:"link" doc:"Extra linker arguments for this program, added after the ones of the system."`
	Libc    *bool        `toml:"libc" doc:"Link against the C library for this program."`
	Output  OutputConfig `toml:"output" doc:"Where this program writes its artifacts. {profile}, {target}, {os}, {arch} and {name} are replaced."`
	Run     RunConfig    `toml:"run" doc:"How wisp run starts this program. Its env adds to the top-level one."`
	Profile string       `toml:"profile" doc:"The profile to build this program with. Defaults to the top-level profile."`

	Systems map[string]SystemConfig `toml:"-"`
}

type Overrides struct {
	Name   string       `toml:"name" doc:"Name of the binary on this system, instead of the one of the program."`
	Link   []string     `toml:"link" doc:"Extra linker arguments on this system, added after the ones of the program."`
	Libc   *bool        `toml:"libc" doc:"Link against the C library on this system."`
	Output OutputConfig `toml:"output" doc:"Where the program writes on this system. Only the folders written here change."`
	Run    RunConfig    `toml:"run" doc:"Environment of wisp run on this system. It adds to the one of the program."`
}

type SystemConfig struct {
	Overrides
	Archs map[string]Overrides `toml:"-"`
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
	Arch    string   `toml:"arch" doc:"CPU architecture of a system of your own that has just one. A system with several writes a [targets.<os>.<arch>] table for each instead."`
	ABI     string   `toml:"abi" doc:"Calling convention. Only sysv is supported, and it is the default."`
	Format  string   `toml:"format" doc:"Object format. Only elf64 is supported, and it is the default."`
	Entry   string   `toml:"entry" doc:"Symbol where the linker starts the program. Defaults to _start."`
	Runtime string   `toml:"runtime" doc:"Assembly file with the runtime routines (allocation, panic, I/O) for this system."`
	Start   string   `toml:"start" doc:"Assembly file that calls main and leaves the program. Required for systems of your own."`
	Link    []string `toml:"link" doc:"Extra linker arguments, such as -T linker.ld. On a built-in system they are added to the built-in ones."`
	Libc    *bool    `toml:"libc" doc:"Link against the C library for this system. Overrides the top-level libc."`

	Archs map[string]ArchConfig `toml:"-"`
}

type ArchConfig struct {
	ABI     string   `toml:"abi" doc:"Calling convention. Only sysv is supported, and it is the default."`
	Format  string   `toml:"format" doc:"Object format. Only elf64 is supported, and it is the default."`
	Entry   string   `toml:"entry" doc:"Symbol where the linker starts the program. Defaults to _start."`
	Runtime string   `toml:"runtime" doc:"Assembly file with the runtime routines for this architecture of the system."`
	Start   string   `toml:"start" doc:"Assembly file that calls main and leaves the program, for this architecture."`
	Link    []string `toml:"link" doc:"Extra linker arguments for this architecture, added to the ones of the system."`
	Libc    *bool    `toml:"libc" doc:"Link against the C library for this architecture."`
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
		t = reflect.TypeFor[Config]()
	case "output":
		t = reflect.TypeFor[OutputConfig]()
	case "run", "program.run", "program.system.run":
		t = reflect.TypeFor[RunConfig]()
	case "library":
		t = reflect.TypeFor[LibraryConfig]()
	case "library.output":
		t = reflect.TypeFor[OutputConfig]()
	case "profiles":
		t = reflect.TypeFor[ProfileConfig]()
	case "program":
		t = reflect.TypeFor[ProgramConfig]()
	case "program.output", "program.system.output":
		t = reflect.TypeFor[OutputConfig]()
	case "program.system":
		t = reflect.TypeFor[Overrides]()
	case "targets":
		t = reflect.TypeFor[TargetConfig]()
	case "targets.arch":
		t = reflect.TypeFor[ArchConfig]()
	default:
		return nil
	}

	fields := make([]ConfigField, 0, t.NumField())
	for f := range t.Fields() {
		f := f
		key := f.Tag.Get("toml")
		if key == "" || key == "-" {
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
	if err := cfg.checkSystems(text, path); err != nil {
		return nil, err
	}
	return cfg, nil
}

func readSource(root, name, field, file string) (string, error) {
	if file == "" {
		return "", nil
	}
	path := file
	if !filepath.IsAbs(path) {
		path = filepath.Join(root, file)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", &target.FieldError{Target: name, Field: field, Message: fmt.Sprintf("cannot read the %s file", field), Help: fmt.Sprintf("%s could not be opened, the system reports %s", file, diag.Reason(err))}
	}
	return string(data), nil
}

type layer struct {
	abi, format, entry, start, runtime string
	link                               []string
	libc                               *bool
}

func (c *Config) layerOf(root, name string, abi, format, entry, start, runtimeFile string, link []string, libc *bool) (layer, error) {
	startSource, err := readSource(root, name, "start", start)
	if err != nil {
		return layer{}, err
	}
	runtimeSource, err := readSource(root, name, "runtime", runtimeFile)
	if err != nil {
		return layer{}, err
	}
	return layer{abi: abi, format: format, entry: entry, start: startSource, runtime: runtimeSource, link: link, libc: libc}, nil
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

		system, err := c.layerOf(root, name, tc.ABI, tc.Format, tc.Entry, tc.Start, tc.Runtime, tc.Link, tc.Libc)
		if err != nil {
			return err
		}

		archLayers := map[string]layer{}
		for arch, ac := range tc.Archs {
			l, err := c.layerOf(root, name+"."+arch, ac.ABI, ac.Format, ac.Entry, ac.Start, ac.Runtime, ac.Link, ac.Libc)
			if err != nil {
				return err
			}
			archLayers[arch] = l
		}

		if target.IsBuiltinOS(name) {
			return &target.FieldError{Target: name, Message: "is a built-in system and is not adjusted here", Help: "link and libc can be set at the top level or in [program.<name>]; to change how it starts, define a system of your own under another name"}
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
				LinkArgs: l.link, Libc: l.libc,
			})
			if err != nil {
				return err
			}
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

		if len(key) == 1 && key[0] == "entry" {
			issues = append(issues, ConfigIssue{
				Line:    locateKey(text, key),
				Message: "entry belongs to a program now",
				Help:    "write [program.<name>] with entry = \"...\", and default = \"<name>\" to choose the one wisp run starts; a project with a single main needs neither",
			})
			continue
		}
		if isTargetOutput(key) {
			issues = append(issues, ConfigIssue{
				Line:    locateKey(text, key),
				Message: "output is not set per system",
				Help:    "where a program writes is up to the program: use [output] with {os} and {arch} in the paths, as in bin = \"dist/{os}/{arch}/{name}\", or [program.<name>.output]",
			})
			continue
		}
		if isArchTable(md, key) || isSystemTable(md, key) {
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
	decodeSystemTables(text, md, &cfg, &issues)

	if cfg.Default != "" {
		if _, declared := cfg.Programs[cfg.Default]; !declared {
			names := make([]string, 0, len(cfg.Programs))
			for n := range cfg.Programs {
				names = append(names, n)
			}
			sort.Strings(names)
			help := "declare it as [program." + cfg.Default + "]"
			if len(names) > 0 {
				help = "the programs are " + strings.Join(names, ", ")
			}
			issues = append(issues, ConfigIssue{Line: locateKey(text, toml.Key{"default"}), Message: fmt.Sprintf("default names the program %q, which is not declared", cfg.Default), Help: help})
		}
	}

	if cfg.Profile != "" {
		if err := target.ValidateProfile(cfg.Profile); err != nil {
			issues = append(issues, ConfigIssue{Line: locateKey(text, toml.Key{"profile"}), Message: fmt.Sprintf("%q is not a profile", cfg.Profile), Help: profileHelp(err)})
		}
	}
	for name := range cfg.Profiles {
		if err := target.ValidateProfile(name); err != nil {
			issues = append(issues, ConfigIssue{Line: locateKey(text, toml.Key{"profiles", name}), Message: fmt.Sprintf("%q is not a profile", name), Help: profileHelp(err)})
		}
	}
	for name, program := range cfg.Programs {
		if program.Profile != "" {
			if err := target.ValidateProfile(program.Profile); err != nil {
				issues = append(issues, ConfigIssue{Line: locateKey(text, toml.Key{"program", name, "profile"}), Message: fmt.Sprintf("%q is not a profile", program.Profile), Help: profileHelp(err)})
			}
		}
	}

	for name, lib := range cfg.Libraries {
		if lib.Profile != "" {
			if err := target.ValidateProfile(lib.Profile); err != nil {
				issues = append(issues, ConfigIssue{Line: locateKey(text, toml.Key{"library", name, "profile"}), Message: fmt.Sprintf("%q is not a profile", lib.Profile), Help: profileHelp(err)})
			}
		}
		if !programNamePattern.MatchString(name) {
			issues = append(issues, ConfigIssue{Line: locateKey(text, toml.Key{"library", name}), Message: fmt.Sprintf("%q is not a valid library name", name), Help: "use lowercase letters, digits and underscores, starting with a letter"})
		}
		if _, clash := cfg.Programs[name]; clash {
			issues = append(issues, ConfigIssue{Line: locateKey(text, toml.Key{"library", name}), Message: fmt.Sprintf("%q is both a program and a library", name), Help: "programs and libraries share their names, pick another one"})
		}
		if len(lib.Modules) == 0 {
			issues = append(issues, ConfigIssue{Line: locateKey(text, toml.Key{"library", name}), Message: fmt.Sprintf("the library %s lists no modules", name), Help: "write modules = [\"geometry\"], with the import paths of the modules it is made of"})
		}
	}

	for name, bin := range cfg.Programs {
		if bin.Name != "" && !binaryNamePattern.MatchString(bin.Name) {
			issues = append(issues, ConfigIssue{Line: locateKey(text, toml.Key{"program", name, "name"}), Message: fmt.Sprintf("%q is not a valid name for a binary", bin.Name), Help: "use letters, digits, underscores, dots and dashes, not starting with a dot or a dash"})
		}
		if !programNamePattern.MatchString(name) {
			issues = append(issues, ConfigIssue{Line: locateKey(text, toml.Key{"program", name}), Message: fmt.Sprintf("%q is not a valid program name", name), Help: "use lowercase letters, digits and underscores, starting with a letter"})
		}
		for envName, value := range bin.Run.Env {
			if _, isString := value.(string); !isString || envName == "" || strings.ContainsAny(envName, "=\x00") {
				issues = append(issues, ConfigIssue{Line: locateKey(text, toml.Key{"program", name, "run", "env", envName}), Message: fmt.Sprintf("program.%s.run.env.%s must be a string", name, envName), Help: fmt.Sprintf("write it as %s = \"...\"", envName)})
			}
		}
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
	case len(key) >= 3 && key[0] == "targets":
		return tomlTags(reflect.TypeOf(TargetConfig{}))
	case len(key) >= 2 && key[0] == "output":
		return tomlTags(reflect.TypeOf(OutputConfig{}))
	case len(key) >= 2 && key[0] == "run":
		return tomlTags(reflect.TypeOf(RunConfig{}))
	case len(key) >= 4 && key[0] == "library" && key[2] == "output":
		return tomlTags(reflect.TypeOf(OutputConfig{}))
	case len(key) >= 3 && key[0] == "library":
		return tomlTags(reflect.TypeOf(LibraryConfig{}))
	case len(key) >= 4 && key[0] == "program" && key[2] == "output":
		return tomlTags(reflect.TypeOf(OutputConfig{}))
	case len(key) >= 4 && key[0] == "program" && key[2] == "run":
		return tomlTags(reflect.TypeOf(RunConfig{}))
	case len(key) >= 3 && key[0] == "program":
		return tomlTags(reflect.TypeOf(ProgramConfig{}))
	}
	return tomlTags(reflect.TypeOf(Config{}))
}

func tomlTags(t reflect.Type) []string {
	tags := make([]string, 0, t.NumField())
	for i := 0; i < t.NumField(); i++ {
		if tag := t.Field(i).Tag.Get("toml"); tag != "" && tag != "-" {
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

func isTargetField(name string) bool {
	return slices.Contains(tomlTags(reflect.TypeOf(TargetConfig{})), name)
}

func isArchTable(md toml.MetaData, key toml.Key) bool {
	if len(key) < 3 || key[0] != "targets" || key[2] == "output" || isTargetField(key[2]) {
		return false
	}
	if len(key) >= 4 {
		return true
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

	candidates := map[string]bool{}
	for _, key := range md.Undecoded() {
		if isArchTable(md, key) {
			candidates[key[1]+"."+key[2]] = true
		}
	}

	tables := map[string]bool{}
	for osName, entries := range raw.Targets {
		for key, prim := range entries {
			if !candidates[osName+"."+key] {
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
		if isTargetOutput(key) {
			*issues = append(*issues, ConfigIssue{
				Line:    locateKey(text, key),
				Message: "output is not set per system",
				Help:    "where a program writes is up to the program: use [output] with {os} and {arch} in the paths, as in bin = \"dist/{os}/{arch}/{name}\", or [program.<name>.output]",
			})
			continue
		}
		known := tomlTags(reflect.TypeOf(ArchConfig{}))
		*issues = append(*issues, ConfigIssue{
			Line:    locateKey(text, key),
			Message: fmt.Sprintf("unknown key %q", key.String()),
			Help:    target.Suggestion(key[len(key)-1], known, "keys"),
		})
	}
	return nil
}

func profileHelp(err error) string {
	var unknown *target.UnknownError
	if errors.As(err, &unknown) && unknown.Help != "" {
		return unknown.Help
	}
	return "the profiles are " + target.Enumerate(target.Profiles())
}

func (c *Config) ProfileFacts(name string) target.Profile {
	facts, _ := target.BuiltinProfile(name)
	over, ok := c.Profiles[name]
	if !ok {
		return facts
	}
	set := func(dst *bool, v *bool) {
		if v != nil {
			*dst = *v
		}
	}
	set(&facts.Trace, over.Trace)
	set(&facts.Snippets, over.Snippets)
	set(&facts.Strip, over.Strip)
	set(&facts.Optimize, over.Optimize)
	set(&facts.Checks, over.Checks)
	return facts
}

func isProgramField(name string) bool {
	return slices.Contains(tomlTags(reflect.TypeOf(ProgramConfig{})), name)
}

func isSystemTable(md toml.MetaData, key toml.Key) bool {
	if len(key) < 3 || key[0] != "program" || isProgramField(key[2]) {
		return false
	}
	if len(key) >= 4 {
		return true
	}
	return md.Type(key[0], key[1], key[2]) == "Hash"
}

func decodeSystemTables(text string, md toml.MetaData, cfg *Config, issues *[]ConfigIssue) {
	var raw struct {
		Program map[string]map[string]toml.Primitive
	}
	rawMeta, err := toml.Decode(text, &raw)
	if err != nil {
		return
	}

	candidates := map[string]bool{}
	for _, key := range md.Undecoded() {
		if isSystemTable(md, key) {
			candidates[key[1]+"."+key[2]] = true
		}
	}

	known := tomlTags(reflect.TypeOf(Overrides{}))
	tables := map[string]bool{}
	for name, entries := range raw.Program {
		for osName, prim := range entries {
			if !candidates[name+"."+osName] {
				continue
			}
			var sc SystemConfig
			if err := rawMeta.PrimitiveDecode(prim, &sc); err != nil {
				*issues = append(*issues, ConfigIssue{Line: locateKey(text, toml.Key{"program", name, osName}), Message: diag.Reason(err)})
				continue
			}
			tables[name+"."+osName] = true

			var inner map[string]toml.Primitive
			if err := rawMeta.PrimitiveDecode(prim, &inner); err != nil {
				continue
			}
			for key, sub := range inner {
				if slices.Contains(known, key) {
					continue
				}
				path := toml.Key{"program", name, osName, key}
				if rawMeta.Type(path...) != "Hash" {
					*issues = append(*issues, ConfigIssue{Line: locateKey(text, path), Message: fmt.Sprintf("unknown key %q", path.String()), Help: target.Suggestion(key, known, "keys")})
					continue
				}
				var o Overrides
				if err := rawMeta.PrimitiveDecode(sub, &o); err != nil {
					*issues = append(*issues, ConfigIssue{Line: locateKey(text, path), Message: diag.Reason(err)})
					continue
				}
				if sc.Archs == nil {
					sc.Archs = map[string]Overrides{}
				}
				sc.Archs[key] = o
				tables[name+"."+osName+"."+key] = true
			}

			program := cfg.Programs[name]
			if program.Systems == nil {
				program.Systems = map[string]SystemConfig{}
			}
			program.Systems[osName] = sc
			cfg.Programs[name] = program
		}
	}

	var reported []toml.Key
	for _, key := range rawMeta.Undecoded() {
		if len(key) < 4 || key[0] != "program" || coveredBy(key, reported) {
			continue
		}
		if !tables[key[1]+"."+key[2]] && !tables[key[1]+"."+key[2]+"."+key[3]] {
			continue
		}
		reported = append(reported, key)
		*issues = append(*issues, ConfigIssue{Line: locateKey(text, key), Message: fmt.Sprintf("unknown key %q", key.String()), Help: target.Suggestion(key[len(key)-1], known, "keys")})
	}
}

func (c *Config) checkSystems(text, path string) error {
	var issues []ConfigIssue
	known := target.OSNames()
	for name, program := range c.Programs {
		for osName, sc := range program.Systems {
			if !slices.Contains(known, osName) {
				issues = append(issues, ConfigIssue{Line: locateKey(text, toml.Key{"program", name, osName}), Message: fmt.Sprintf("%q is not a system", osName), Help: target.Suggestion(osName, append(known, tomlTags(reflect.TypeOf(ProgramConfig{}))...), "systems and keys")})
				continue
			}
			archs := target.ArchsOf(osName)
			for arch := range sc.Archs {
				if !slices.Contains(archs, arch) {
					issues = append(issues, ConfigIssue{Line: locateKey(text, toml.Key{"program", name, osName, arch}), Message: fmt.Sprintf("the system %s has no architecture %q", osName, arch), Help: target.Suggestion(arch, archs, "architectures")})
				}
			}
		}
	}
	if len(issues) == 0 {
		return nil
	}
	sort.Slice(issues, func(i, j int) bool { return issues[i].Line < issues[j].Line })
	return &ConfigError{File: path, Source: text, Issues: issues}
}

var programNamePattern = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

var binaryNamePattern = regexp.MustCompile(`^[A-Za-z0-9_][A-Za-z0-9_.-]*$`)

func (c *Config) ProgramNames(project string) []string {
	if len(c.Programs) == 0 {
		if len(c.Libraries) > 0 {
			return nil
		}
		return []string{project}
	}
	names := make([]string, 0, len(c.Programs))
	for name := range c.Programs {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (c *Config) DefaultProgram(project string) string {
	if c.Default != "" {
		return c.Default
	}
	if names := c.ProgramNames(project); len(names) == 1 {
		return names[0]
	}
	return ""
}

func (c *Config) HasProgram(name, project string) bool {
	return slices.Contains(c.ProgramNames(project), name)
}

func (c *Config) HasLibrary(name string) bool {
	_, ok := c.Libraries[name]
	return ok
}

func (c *Config) ForProgram(name string) *Config {
	eff := *c
	eff.ProgramName = name
	eff.BinaryName = name

	bin, declared := c.Programs[name]
	if !declared {
		return &eff
	}

	if bin.Name != "" {
		eff.BinaryName = bin.Name
	}
	if bin.Entry != "" {
		eff.Entry = bin.Entry
	}
	if bin.OS != "" {
		eff.OS = bin.OS
		eff.Arch = ""
	}
	if bin.Arch != "" {
		eff.Arch = bin.Arch
	}
	if bin.Profile != "" {
		eff.Profile = bin.Profile
	}
	eff.ProgramLink = bin.Link
	eff.ProgramLibc = bin.Libc
	eff.ProgramOutput = bin.Output

	env := map[string]any{}
	for k, v := range c.Run.Env {
		env[k] = v
	}
	for k, v := range bin.Run.Env {
		env[k] = v
	}
	eff.Run = RunConfig{Env: env}

	return &eff
}

func (c *Config) ForSystem(osName, arch string) *Config {
	program, declared := c.Programs[c.ProgramName]
	if !declared {
		return c
	}
	sc, ok := program.Systems[osName]
	if !ok {
		return c
	}
	eff := *c
	eff.apply(sc.Overrides)
	if o, ok := sc.Archs[arch]; ok {
		eff.apply(o)
	}
	return &eff
}

func (c *Config) apply(o Overrides) {
	if o.Name != "" {
		c.BinaryName = o.Name
	}
	c.ProgramLink = append(append([]string{}, c.ProgramLink...), o.Link...)
	if o.Libc != nil {
		c.ProgramLibc = o.Libc
	}
	if o.Output.Bin != "" {
		c.ProgramOutput.Bin = o.Output.Bin
	}
	if o.Output.Obj != "" {
		c.ProgramOutput.Obj = o.Output.Obj
	}
	if o.Output.Asm != "" {
		c.ProgramOutput.Asm = o.Output.Asm
	}
	env := map[string]any{}
	maps.Copy(env, c.Run.Env)
	maps.Copy(env, o.Run.Env)
	c.Run = RunConfig{Env: env}
}

func isTargetOutput(key toml.Key) bool {
	return len(key) >= 3 && key[0] == "targets" && (key[2] == "output" || (len(key) >= 4 && key[3] == "output"))
}

func (c *Config) LibraryNames() []string {
	names := make([]string, 0, len(c.Libraries))
	for name := range c.Libraries {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (c *Config) ForLibrary(name string) *Config {
	eff := *c
	eff.ProgramName = name
	eff.BinaryName = name
	eff.Entry = ""
	eff.ProgramLink = nil
	eff.ProgramLibc = nil

	lib := c.Libraries[name]
	if lib.OS != "" {
		eff.OS = lib.OS
		eff.Arch = ""
	}
	if lib.Arch != "" {
		eff.Arch = lib.Arch
	}
	if lib.Profile != "" {
		eff.Profile = lib.Profile
	}
	eff.ProgramOutput = lib.Output
	return &eff
}
