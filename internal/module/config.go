package module

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/Gui97p/wisp/internal/target"
)

type Config struct {
	Entry        string                  `toml:"entry"`
	Target       string                  `toml:"target"`
	Std          string                  `toml:"std"`
	Freestanding bool                    `toml:"freestanding"`
	Link         []string                `toml:"link"`
	Libc         bool                    `toml:"libc"`
	Output       OutputConfig            `toml:"output"`
	Targets      map[string]TargetConfig `toml:"targets"`
}

type TargetConfig struct {
	Arch    string   `toml:"arch"`
	OS      string   `toml:"os"`
	ABI     string   `toml:"abi"`
	Format  string   `toml:"format"`
	Entry   string   `toml:"entry"`
	Runtime string   `toml:"runtime"`
	Start   string   `toml:"start"`
	Link    []string `toml:"link"`
}

type OutputConfig struct {
	Bin string `toml:"bin"`
	Obj string `toml:"obj"`
	Asm string `toml:"asm"`
	Lua string `toml:"lua"`
}

func LoadConfig(root string) (*Config, error) {
	path := filepath.Join(root, "wisp.toml")

	if _, err := os.Stat(path); err != nil {
		return &Config{}, nil
	}

	cfg, err := decodeConfig(path)
	if err != nil {
		return nil, err
	}
	if err := cfg.registerTargets(root); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *Config) registerTargets(root string) error {
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
				return fmt.Errorf("target %q: cannot read start file: %w", name, err)
			}
			start = string(data)
		}
		if tc.Runtime != "" {
			data, err := os.ReadFile(filepath.Join(root, tc.Runtime))
			if err != nil {
				return fmt.Errorf("target %q: cannot read runtime file: %w", name, err)
			}
			runtimeSource = string(data)
		}

		err := target.Register(target.Target{
			Name:          name,
			Arch:          tc.Arch,
			OS:            tc.OS,
			ABI:           tc.ABI,
			Format:        tc.Format,
			Entry:         tc.Entry,
			StartSource:   start,
			RuntimeSource: runtimeSource,
			LinkArgs:      tc.Link,
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *Config) StdlibSettings(root string) (string, bool, error) {
	if c.Std != "" && c.Freestanding {
		return "", false, fmt.Errorf("wisp.toml: std and freestanding cannot be used together")
	}

	path := c.Std
	if path != "" && !filepath.IsAbs(path) {
		path = filepath.Join(root, path)
	}
	return path, c.Freestanding, nil
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
}

type ConfigError struct {
	File   string
	Issues []ConfigIssue
}

func (e *ConfigError) Error() string {
	lines := make([]string, len(e.Issues))
	for i, issue := range e.Issues {
		if issue.Line > 0 {
			lines[i] = fmt.Sprintf("%s:%d: %s", filepath.Base(e.File), issue.Line, issue.Message)
		} else {
			lines[i] = fmt.Sprintf("%s: %s", filepath.Base(e.File), issue.Message)
		}
	}
	return strings.Join(lines, "\n")
}

func decodeConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	md, err := toml.Decode(string(data), &cfg)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", filepath.Base(path), err)
	}

	var issues []ConfigIssue
	var reported []toml.Key
	for _, key := range md.Undecoded() {
		if coveredBy(key, reported) {
			continue
		}
		reported = append(reported, key)

		known := knownKeys(key)
		issues = append(issues, ConfigIssue{
			Line:    locateKey(string(data), key),
			Message: fmt.Sprintf("unknown key %q%s", key.String(), target.Hint(key[len(key)-1], known)),
		})
	}
	if len(issues) > 0 {
		return nil, &ConfigError{File: path, Issues: issues}
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
