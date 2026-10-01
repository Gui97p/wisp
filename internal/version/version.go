package version

import (
	_ "embed"
	"fmt"
	"regexp"
	"runtime"
	"runtime/debug"
	"strings"
)

//go:embed VERSION
var raw string

var semver = regexp.MustCompile(`^\d+\.\d+\.\d+(-[0-9A-Za-z.]+)?$`)

type Info struct {
	Version  string
	Codename string
}

type Build struct {
	Commit   string
	Time     string
	Modified bool
	Go       string
	Platform string
}

func Parse(content string) (Info, error) {
	fields := map[string]string{}
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			return Info{}, fmt.Errorf("malformed line %q, expected key = value", line)
		}
		fields[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}

	info := Info{Version: fields["version"], Codename: fields["codename"]}
	if !semver.MatchString(info.Version) {
		return Info{}, fmt.Errorf("invalid version %q, expected major.minor.patch", info.Version)
	}
	if info.Codename == "" {
		return Info{}, fmt.Errorf("missing codename")
	}

	return info, nil
}

func current() Info {
	info, err := Parse(raw)
	if err != nil {
		return Info{Version: "0.0.0-invalid", Codename: "unknown"}
	}

	return info
}

func Version() string {
	return current().Version
}

func Codename() string {
	return current().Codename
}

func String() string {
	info := current()
	return fmt.Sprintf("%s %q", info.Version, info.Codename)
}

func BuildInfo() Build {
	b := Build{Go: runtime.Version(), Platform: runtime.GOOS + "/" + runtime.GOARCH}

	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return b
	}

	for _, s := range bi.Settings {
		switch s.Key {
		case "vcs.revision":
			b.Commit = s.Value
		case "vcs.time":
			b.Time = s.Value
		case "vcs.modified":
			b.Modified = s.Value == "true"
		}
	}

	return b
}

func Env() []string {
	info := current()
	return []string{"WISP_VERSION=" + info.Version, "WISP_CODENAME=" + info.Codename}
}
