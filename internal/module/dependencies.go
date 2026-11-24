package module

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/BurntSushi/toml"
)

type stamp struct {
	modified time.Time
	size     int64
	exists   bool
}

func stampOf(path string) stamp {
	info, err := os.Stat(path)
	if err != nil {
		return stamp{}
	}
	return stamp{modified: info.ModTime(), size: info.Size(), exists: true}
}

type watcher map[string]stamp

func (w watcher) add(path string) {
	if w != nil {
		w[path] = stampOf(path)
	}
}

type depsCacheEntry struct {
	watched watcher
	deps    map[string]string
}

func (e depsCacheEntry) valid() bool {
	for path, was := range e.watched {
		if now := stampOf(path); now.exists != was.exists || !now.modified.Equal(was.modified) || now.size != was.size {
			return false
		}
	}
	return true
}

var (
	depsCache   = map[string]depsCacheEntry{}
	depsCacheMu sync.Mutex
)

func dependenciesOf(root string) map[string]string {
	key := filepath.Clean(root)
	depsCacheMu.Lock()
	cached, ok := depsCache[key]
	depsCacheMu.Unlock()
	if ok && cached.valid() {
		return cached.deps
	}

	watched := watcher{}
	deps := mergedDependencies(key, watched)

	depsCacheMu.Lock()
	depsCache[key] = depsCacheEntry{watched: watched, deps: deps}
	depsCacheMu.Unlock()
	return deps
}

func (c *Config) DependencyRoots(root string) map[string]string {
	if len(c.Dependencies) == 0 {
		return nil
	}
	roots := make(map[string]string, len(c.Dependencies))
	for name, dep := range c.Dependencies {
		path := dep.Path
		if !filepath.IsAbs(path) {
			path = filepath.Join(root, path)
		}
		roots[name] = filepath.Clean(path)
	}
	return roots
}

func (c *Config) DependencyNames() []string {
	names := make([]string, 0, len(c.Dependencies))
	for name := range c.Dependencies {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func DependencyRoots(projectRoot string) map[string]string {
	return dependenciesOf(projectRoot)
}

func cachedConfig(path string) *Config {
	now := stampOf(path)
	if !now.exists {
		return nil
	}

	configCacheMu.Lock()
	cached, ok := configCache[path]
	configCacheMu.Unlock()
	if ok && cached.stamp == now {
		return cached.cfg
	}

	configCacheMu.Lock()
	if decoding[path] {
		configCacheMu.Unlock()
		return nil
	}
	decoding[path] = true
	configCacheMu.Unlock()

	var cfg *Config
	if decoded, err := decodeConfig(path); err == nil {
		cfg = decoded
	}

	configCacheMu.Lock()
	delete(decoding, path)
	configCache[path] = configCacheEntry{stamp: now, cfg: cfg}
	configCacheMu.Unlock()
	return cfg
}

type configCacheEntry struct {
	stamp stamp
	cfg   *Config
}

var (
	configCache   = map[string]configCacheEntry{}
	configCacheMu sync.Mutex
	decoding      = map[string]bool{}
)

func (c *Config) ProjectName(root string) string {
	if c != nil && c.Project.Name != "" {
		return c.Project.Name
	}
	return filepath.Base(root)
}

func (c *Config) IsWorkspace() bool {
	return c.Workspace.defined || len(c.Workspace.Members) > 0
}

type Member struct {
	Name string
	Dir  string
}

func hasGlob(pattern string) bool {
	return strings.ContainsAny(pattern, "*?[")
}

func hiddenFolder(name string) bool {
	return strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_")
}

func (c *Config) memberDirs(root string, w watcher) []string {
	patterns := c.Workspace.Members
	if len(patterns) == 0 {
		patterns = []string{"*"}
	}

	excluded := func(dir string) bool {
		rel, err := filepath.Rel(root, dir)
		if err != nil {
			return false
		}
		rel = filepath.ToSlash(rel)
		for _, pattern := range c.Workspace.Exclude {
			if ok, _ := filepath.Match(pattern, rel); ok {
				return true
			}
			if ok, _ := filepath.Match(pattern, filepath.Base(dir)); ok && !strings.Contains(pattern, "/") {
				return true
			}
		}
		return false
	}

	seen := map[string]bool{}
	var dirs []string
	add := func(dir string) {
		dir = filepath.Clean(dir)
		if !seen[dir] && !excluded(dir) {
			seen[dir] = true
			dirs = append(dirs, dir)
		}
	}

	for _, pattern := range patterns {
		abs := pattern
		if !filepath.IsAbs(abs) {
			abs = filepath.Join(root, abs)
		}
		if !hasGlob(pattern) {
			w.add(filepath.Join(abs, "wisp.toml"))
			add(abs)
			continue
		}
		w.add(filepath.Dir(abs))
		matches, _ := filepath.Glob(abs)
		sort.Strings(matches)
		for _, match := range matches {
			if hiddenFolder(filepath.Base(match)) {
				continue
			}
			if info, err := os.Stat(match); err != nil || !info.IsDir() {
				continue
			}
			w.add(filepath.Join(match, "wisp.toml"))
			if _, err := os.Stat(filepath.Join(match, "wisp.toml")); err == nil {
				add(match)
			}
		}
	}
	return dirs
}

func (c *Config) members(root string, w watcher) []Member {
	dirs := c.memberDirs(root, w)
	members := make([]Member, 0, len(dirs))
	for _, dir := range dirs {
		toml := filepath.Join(dir, "wisp.toml")
		w.add(toml)
		members = append(members, Member{Name: cachedConfig(toml).ProjectName(dir), Dir: dir})
	}
	return members
}

func (c *Config) WorkspaceMembers(root string) []Member {
	return c.members(root, nil)
}

func workspaceFor(root string, w watcher) (string, *Config) {
	root = filepath.Clean(root)
	for dir := root; ; dir = filepath.Dir(dir) {
		toml := filepath.Join(dir, "wisp.toml")
		w.add(toml)
		if cfg := cachedConfig(toml); cfg != nil && cfg.IsWorkspace() {
			if dir == root {
				return dir, cfg
			}
			for _, member := range cfg.members(dir, w) {
				if member.Dir == root {
					return dir, cfg
				}
			}
			return "", nil
		}
		if parent := filepath.Dir(dir); parent == dir {
			return "", nil
		}
	}
}

func mergedDependencies(root string, w watcher) map[string]string {
	deps := map[string]string{}
	if wsRoot, ws := workspaceFor(root, w); ws != nil {
		for _, member := range ws.members(wsRoot, w) {
			if member.Dir != root && programNamePattern.MatchString(member.Name) {
				deps[member.Name] = member.Dir
			}
		}
	}
	if cfg := cachedConfig(filepath.Join(root, "wisp.toml")); cfg != nil {
		for name, dir := range cfg.DependencyRoots(root) {
			deps[name] = dir
		}
	}
	if len(deps) == 0 {
		return nil
	}
	return deps
}

func checkWorkspace(text, root string, cfg *Config) []ConfigIssue {
	var issues []ConfigIssue
	line := locateKey(text, toml.Key{"workspace", "members"})
	if line == 0 {
		line = locateKey(text, toml.Key{"workspace"})
	}

	for _, folder := range cfg.Workspace.Members {
		if hasGlob(folder) {
			continue
		}
		dir := folder
		if !filepath.IsAbs(dir) {
			dir = filepath.Join(root, dir)
		}
		if info, err := os.Stat(dir); err != nil || !info.IsDir() {
			issues = append(issues, ConfigIssue{Line: line, Message: fmt.Sprintf("the member %s is not a folder", folder), Help: "list the folders of the projects, relative to this wisp.toml, or leave members out to take every folder that has a wisp.toml"})
		} else if _, err := os.Stat(filepath.Join(dir, "wisp.toml")); err != nil {
			issues = append(issues, ConfigIssue{Line: line, Message: fmt.Sprintf("the member %s has no wisp.toml", folder), Help: "every member is a project of its own: run wisp init inside it"})
		}
	}

	seen := map[string]string{}
	for _, dir := range cfg.memberDirs(root, nil) {
		folder, _ := filepath.Rel(root, dir)
		folder = filepath.ToSlash(folder)
		memberToml := filepath.Join(dir, "wisp.toml")
		if _, err := os.Stat(memberToml); err != nil {
			continue
		}
		name := cachedConfig(memberToml).ProjectName(dir)
		switch other, dup := seen[name]; {
		case name == "std":
			issues = append(issues, ConfigIssue{Line: line, Message: fmt.Sprintf("the member %s would be called std, which is the standard library", folder), Help: "give it another name with [project] name, or leave it out with exclude"})
		case !programNamePattern.MatchString(name):
			issues = append(issues, ConfigIssue{Line: line, Message: fmt.Sprintf("the member %s cannot be imported as %q", folder, name), Help: "write [project] name = \"...\" in its wisp.toml, with lowercase letters, digits and underscores, or leave it out with exclude"})
		case dup:
			issues = append(issues, ConfigIssue{Line: line, Message: fmt.Sprintf("the members %s and %s are both called %s", other, folder, name), Help: "give one of them another name with [project] name"})
		default:
			seen[name] = folder
		}
	}
	return issues
}

func HasOwnSources(root string) bool {
	roots, err := discoverModuleRoots(root, nil)
	return err == nil && len(roots) > 0
}

func WorkspaceRootOf(root string) (string, bool) {
	wsRoot, ws := workspaceFor(root, nil)
	return wsRoot, ws != nil
}
