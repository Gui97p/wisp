package target

import (
	"strings"
	"sync"
)

type Profile struct {
	Name     string
	Trace    bool
	Snippets bool
	Strip    bool
	Optimize bool
	Checks   bool
}

const DefaultProfile = "debug"

var builtinProfiles = []Profile{
	{Name: "debug", Trace: true, Snippets: true, Checks: true},
	{Name: "release", Strip: true, Optimize: true, Checks: true},
	{Name: "fast", Strip: true, Optimize: true},
}

func Profiles() []string {
	names := make([]string, len(builtinProfiles))
	for i, p := range builtinProfiles {
		names[i] = p.Name
	}
	return names
}

func IsProfile(name string) bool {
	_, ok := BuiltinProfile(name)
	return ok
}

func BuiltinProfile(name string) (Profile, bool) {
	for _, p := range builtinProfiles {
		if p.Name == name {
			return p, true
		}
	}
	return Profile{}, false
}

func ValidateProfile(name string) error {
	if IsProfile(name) {
		return nil
	}
	return &UnknownError{Name: name, Help: Suggestion(name, Profiles(), "profiles")}
}

func (p Profile) CompileKey() string {
	var key strings.Builder
	key.WriteString(p.Name)
	for _, on := range []bool{p.Trace, p.Optimize, p.Checks} {
		if on {
			key.WriteString("1")
		} else {
			key.WriteString("0")
		}
	}
	return key.String()
}

var (
	profileMu   sync.RWMutex
	activeFacts = builtinProfiles[0]
)

func SetActiveProfile(p Profile) {
	profileMu.Lock()
	defer profileMu.Unlock()
	activeFacts = p
}

func ActiveProfile() Profile {
	profileMu.RLock()
	defer profileMu.RUnlock()
	return activeFacts
}
