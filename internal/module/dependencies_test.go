package module

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func write(t *testing.T, root, name, content string) {
	t.Helper()
	path := filepath.Join(root, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func keys(mods []*Module) []string {
	var out []string
	for _, m := range mods {
		if !strings.HasPrefix(m.Path, "std/") {
			out = append(out, m.Path)
		}
	}
	sort.Strings(out)
	return out
}

func layout(t *testing.T) (app, geometry, mathx string) {
	t.Helper()
	base := t.TempDir()
	app, geometry, mathx = filepath.Join(base, "app"), filepath.Join(base, "geometry"), filepath.Join(base, "mathx")
	write(t, mathx, "mathx.wsp", "public func twice(int n) int => n * 2;\n")
	write(t, geometry, "wisp.toml", "[dependencies]\nmathx = { path = \"../mathx\" }\n")
	write(t, geometry, "geometry.wsp", "import \"util\";\nimport \"mathx\";\npublic func area(int w, h) int => util.mul(w, h);\n")
	write(t, geometry, "util.wsp", "public func mul(int a, b) int => a * b;\n")
	write(t, geometry, "shapes/shapes.wsp", "import \"geometry\";\npublic func f() int => geometry.area(1, 2);\n")
	write(t, app, "wisp.toml", "[dependencies]\ngeo = { path = \"../geometry\" }\nmathx = { path = \"../mathx\" }\n")
	write(t, app, "main.wsp", "import \"geo\";\nimport \"geo/shapes\";\nimport \"mathx\";\nfunc main() {}\n")
	return app, geometry, mathx
}

func TestDependenciesGetOneModuleEach(t *testing.T) {
	app, _, _ := layout(t)
	mods, err := DiscoverProject(app)
	if err != nil {
		t.Fatal(err)
	}
	got := keys(mods)
	want := []string{"geo", "geo/shapes/shapes", "geo/util", "main", "mathx"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("modules %v, want %v", got, want)
	}
	for _, m := range mods {
		if m.Dependency == "" && (m.Path == "geo" || m.Path == "mathx") {
			t.Errorf("%s should belong to a dependency", m.Path)
		}
		for path, message := range m.ImportErrors {
			t.Errorf("%s: import %s failed: %s", m.Path, path, message)
		}
	}
}

func TestDependencyNameCannotAlsoBeALocalFolder(t *testing.T) {
	app, _, _ := layout(t)
	write(t, app, "geo/geo.wsp", "public func x() int => 1;\n")
	mods, _ := DiscoverProject(app)
	for _, m := range mods {
		if m.Path == "main" {
			message := m.ImportErrors["geo"]
			if !strings.Contains(message, "ambiguous") || !strings.Contains(message, "dependency geo") {
				t.Errorf("the import should be ambiguous: %q", message)
			}
			return
		}
	}
	t.Error("main was not found")
}

func TestTwoPathsForTheSameDependencyNameIsAnError(t *testing.T) {
	app, geometry, _ := layout(t)
	other := filepath.Join(filepath.Dir(app), "other")
	write(t, other, "other.wsp", "public func twice(int n) int => n;\n")
	write(t, geometry, "wisp.toml", "[dependencies]\nmathx = { path = \"../other\" }\n")
	mods, _ := DiscoverProject(app)
	found := false
	for _, m := range mods {
		for _, message := range m.ImportErrors {
			if strings.Contains(message, "one project per name") {
				found = true
			}
		}
	}
	if !found {
		t.Error("two different folders for mathx should be refused")
	}
}

func TestMissingDependencyFolderAndEntryExplainThemselves(t *testing.T) {
	app, _, _ := layout(t)
	write(t, app, "wisp.toml", "[dependencies]\ngeo = { path = \"../nowhere\" }\n")
	write(t, app, "main.wsp", "import \"geo\";\nfunc main() {}\n")
	mods, _ := DiscoverProject(app)
	for _, m := range mods {
		if m.Path == "main" && !strings.Contains(m.ImportErrors["geo"], "which is not a folder") {
			t.Errorf("got %q", m.ImportErrors["geo"])
		}
	}

	app2, _, mathx := layout(t)
	os.Remove(filepath.Join(mathx, "mathx.wsp"))
	write(t, mathx, "other.wsp", "public func x() int => 1;\n")
	mods, _ = DiscoverProject(app2)
	for _, m := range mods {
		if m.Path == "main" && !strings.Contains(m.ImportErrors["mathx"], "has no module file mathx.wsp") {
			t.Errorf("got %q", m.ImportErrors["mathx"])
		}
	}
}

func TestDependenciesAreOfferedToAutoImport(t *testing.T) {
	app, _, _ := layout(t)
	var paths []string
	for _, c := range ImportablePaths(app) {
		if !strings.HasPrefix(c.Path, "std/") {
			paths = append(paths, c.Path)
		}
	}
	sort.Strings(paths)
	joined := strings.Join(paths, ",")
	for _, want := range []string{"geo", "geo/util", "mathx", "main"} {
		if !strings.Contains(","+joined+",", ","+want+",") {
			t.Errorf("%q is not offered: %s", want, joined)
		}
	}
}

func TestDependencyTestsAreNotRun(t *testing.T) {
	app, geometry, _ := layout(t)
	write(t, geometry, "geometry.wsp", "import \"std/testing\";\nimport \"util\";\nimport \"mathx\";\npublic func area(int w, h) int => util.mul(w, h);\n#test\nfunc depTest() ! { testing.expect(true)!; }\n")
	ConfigureTests(true)
	defer ConfigureTests(false)
	_, tests, _, err := DiscoverProjectWithTests(app, nil, func([]TestFunc) string { return "func main() {}\n" })
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range tests {
		if strings.HasPrefix(test.Module, "geo") {
			t.Errorf("a dependency's test ran: %+v", test)
		}
	}
}
