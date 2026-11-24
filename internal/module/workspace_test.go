package module

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func workspace(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	write(t, root, "wisp.toml", "[workspace]\nmembers = [\"geometry\", \"mathx\", \"tools\"]\n")
	write(t, root, "geometry/wisp.toml", "[project]\nname = \"geo\"\n")
	write(t, root, "geometry/geometry.wsp", "import \"mathx\";\npublic func double(int n) int => mathx.twice(n);\n")
	write(t, root, "mathx/wisp.toml", "")
	write(t, root, "mathx/mathx.wsp", "public func twice(int n) int => n * 2;\n")
	write(t, root, "tools/wisp.toml", "")
	write(t, root, "tools/main.wsp", "import \"geo\";\nimport \"mathx\";\nfunc main() {}\n")
	return root
}

func TestMembersImportEachOtherByProjectName(t *testing.T) {
	root := workspace(t)
	mods, err := DiscoverProject(filepath.Join(root, "tools"))
	if err != nil {
		t.Fatal(err)
	}
	got := strings.Join(keys(mods), ",")
	if got != "geo,main,mathx" {
		t.Errorf("modules %s", got)
	}
	for _, m := range mods {
		for path, message := range m.ImportErrors {
			t.Errorf("%s: import %s failed: %s", m.Path, path, message)
		}
	}
}

func TestProjectNameDefaultsToTheFolderAndCanBeOverridden(t *testing.T) {
	root := workspace(t)
	deps := DependencyRoots(filepath.Join(root, "tools"))
	if deps["geo"] != filepath.Join(root, "geometry") || deps["mathx"] != filepath.Join(root, "mathx") {
		t.Errorf("dependencies %v", deps)
	}
	if _, folderName := deps["geometry"]; folderName {
		t.Error("the project is called geo, not geometry")
	}
	if _, self := deps["tools"]; self {
		t.Error("a member does not depend on itself")
	}
}

func TestAnExplicitDependencyBeatsAMemberName(t *testing.T) {
	root := workspace(t)
	write(t, root, "other/other.wsp", "public func twice(int n) int => n;\n")
	write(t, root, "tools/wisp.toml", "[dependencies]\nmathx = { path = \"../other\" }\n")
	deps := DependencyRoots(filepath.Join(root, "tools"))
	if deps["mathx"] != filepath.Join(root, "other") {
		t.Errorf("mathx should be the explicit one: %v", deps["mathx"])
	}
}

func TestTheWorkspaceRootSeesItsMembersToo(t *testing.T) {
	root := workspace(t)
	if deps := DependencyRoots(root); deps["geo"] == "" || deps["mathx"] == "" || deps["tools"] == "" {
		t.Errorf("the root should see every member: %v", deps)
	}
	if wsRoot, ok := WorkspaceRootOf(filepath.Join(root, "tools")); !ok || wsRoot != root {
		t.Errorf("workspace of a member: %q, %v", wsRoot, ok)
	}
	if !HasOwnSources(filepath.Join(root, "tools")) || HasOwnSources(root) {
		t.Error("only the members have sources: the root is a plain workspace")
	}
}

func TestAProjectOutsideTheMembersIsNotPartOfTheWorkspace(t *testing.T) {
	root := workspace(t)
	write(t, root, "stranger/wisp.toml", "")
	write(t, root, "stranger/main.wsp", "func main() {}\n")
	if _, ok := WorkspaceRootOf(filepath.Join(root, "stranger")); ok {
		t.Error("stranger is not listed in members")
	}
	if deps := DependencyRoots(filepath.Join(root, "stranger")); len(deps) != 0 {
		t.Errorf("stranger sees %v", deps)
	}
}

func TestWorkspaceMistakesAreReported(t *testing.T) {
	for name, tc := range map[string]struct {
		toml  string
		extra map[string]string
		want  string
	}{
		"missing folder":    {"[workspace]\nmembers = [\"nowhere\"]\n", nil, "the member nowhere is not a folder"},
		"no wisp.toml":      {"[workspace]\nmembers = [\"bare\"]\n", map[string]string{"bare/x.wsp": ""}, "has no wisp.toml"},
		"invalid name":      {"[workspace]\nmembers = [\"my-app\"]\n", map[string]string{"my-app/wisp.toml": ""}, "cannot be imported as"},
		"duplicate names":   {"[workspace]\nmembers = [\"a\", \"b\"]\n", map[string]string{"a/wisp.toml": "[project]\nname = \"same\"\n", "b/wisp.toml": "[project]\nname = \"same\"\n"}, "are both called same"},
		"bad project name":  {"[project]\nname = \"Bad Name\"\n", nil, "is not a valid project name"},
		"unknown workspace": {"[workspace]\nmember = [\"a\"]\n", nil, "unknown key"},
	} {
		root := t.TempDir()
		write(t, root, "wisp.toml", tc.toml)
		for file, content := range tc.extra {
			write(t, root, file, content)
		}
		_, err := LoadConfig(root)
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: got %v, want %q", name, err, tc.want)
		}
	}
}

func TestWorkspacesThatPointAtEachOtherDoNotLoopForever(t *testing.T) {
	root := t.TempDir()
	write(t, root, "wisp.toml", "[workspace]\nmembers = [\"a\"]\n")
	write(t, root, "a/wisp.toml", "[workspace]\nmembers = [\"..\"]\n")
	_, _ = LoadConfig(root)
	DependencyRoots(filepath.Join(root, "a"))
}

func implicitWorkspace(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	write(t, root, "wisp.toml", "[workspace]\n")
	write(t, root, "geometry/wisp.toml", "[project]\nname = \"geo\"\n")
	write(t, root, "geometry/geometry.wsp", "public func area(int w, h) int => w * h;\n")
	write(t, root, "tools/wisp.toml", "")
	write(t, root, "tools/main.wsp", "func main() {}\n")
	write(t, root, "docs/readme.txt", "no project here\n")
	write(t, root, "_scratch/wisp.toml", "")
	write(t, root, ".hidden/wisp.toml", "")
	return root
}

func names(members []Member) string {
	var out []string
	for _, m := range members {
		out = append(out, m.Name)
	}
	sort.Strings(out)
	return strings.Join(out, ",")
}

func TestAnEmptyWorkspaceTakesEveryFolderThatIsAProject(t *testing.T) {
	root := implicitWorkspace(t)
	cfg, err := LoadConfig(root)
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.IsWorkspace() {
		t.Fatal("an empty [workspace] still makes a workspace")
	}
	if got := names(cfg.WorkspaceMembers(root)); got != "geo,tools" {
		t.Errorf("members %s: folders without a wisp.toml and hidden ones are not members", got)
	}
	if deps := DependencyRoots(filepath.Join(root, "tools")); deps["geo"] != filepath.Join(root, "geometry") {
		t.Errorf("tools should see geo: %v", deps)
	}
}

func TestMembersCanBeGlobsAndExcluded(t *testing.T) {
	root := t.TempDir()
	write(t, root, "wisp.toml", "[workspace]\nmembers = [\"crates/*\", \"app\"]\nexclude = [\"crates/old\"]\n")
	write(t, root, "crates/a/wisp.toml", "")
	write(t, root, "crates/b/wisp.toml", "")
	write(t, root, "crates/old/wisp.toml", "")
	write(t, root, "crates/notes/readme.txt", "")
	write(t, root, "app/wisp.toml", "")
	write(t, root, "other/wisp.toml", "")
	cfg, err := LoadConfig(root)
	if err != nil {
		t.Fatal(err)
	}
	if got := names(cfg.WorkspaceMembers(root)); got != "a,app,b" {
		t.Errorf("members %s", got)
	}
}

func TestANewFolderOrAChangedWorkspaceIsSeenWithoutRestarting(t *testing.T) {
	root := implicitWorkspace(t)
	tools := filepath.Join(root, "tools")
	if _, ok := DependencyRoots(tools)["extra"]; ok {
		t.Fatal("extra does not exist yet")
	}

	write(t, root, "extra/wisp.toml", "")
	if _, ok := DependencyRoots(tools)["extra"]; !ok {
		t.Error("a new folder with a wisp.toml should become a member")
	}

	write(t, root, "wisp.toml", "[workspace]\nexclude = [\"extra\"]\n")
	if _, ok := DependencyRoots(tools)["extra"]; ok {
		t.Error("editing the workspace file should drop extra")
	}

	if err := os.Remove(filepath.Join(root, "geometry", "wisp.toml")); err != nil {
		t.Fatal(err)
	}
	if _, ok := DependencyRoots(tools)["geo"]; ok {
		t.Error("a folder that stopped being a project should leave the workspace")
	}
}

func TestAMemberCalledStdOrNotImportableIsRefused(t *testing.T) {
	root := t.TempDir()
	write(t, root, "wisp.toml", "[workspace]\n")
	write(t, root, "std/wisp.toml", "")
	write(t, root, "my-app/wisp.toml", "")
	_, err := LoadConfig(root)
	if err == nil || !strings.Contains(err.Error(), "would be called std") || !strings.Contains(err.Error(), `cannot be imported as "my-app"`) {
		t.Errorf("got %v", err)
	}

	write(t, root, "wisp.toml", "[workspace]\nexclude = [\"std\", \"my-app\"]\n")
	if _, err := LoadConfig(root); err != nil {
		t.Errorf("exclude should settle both: %v", err)
	}
}
