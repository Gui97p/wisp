package workspace

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

var binary string

func repoRoot() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func TestMain(m *testing.M) {
	root := repoRoot()
	binary = filepath.Join(root, "bin", "wisp")
	build := exec.Command("go", "build", "-o", binary, "./cmd/main")
	build.Dir = root
	if out, err := build.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "build failed: %v\n%s", err, out)
		os.Exit(1)
	}
	os.Exit(m.Run())
}

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

func workspace(t *testing.T, toolsMain string) string {
	t.Helper()
	root := t.TempDir()
	write(t, root, "wisp.toml", "[workspace]\nmembers = [\"geometry\", \"mathx\", \"tools\"]\n")
	write(t, root, "geometry/wisp.toml", "[project]\nname = \"geo\"\n")
	write(t, root, "geometry/geometry.wsp", "import \"mathx\";\npublic func area(int w, h) int => w * h;\npublic func doubled(int n) int => mathx.twice(n);\n")
	write(t, root, "mathx/wisp.toml", "")
	write(t, root, "mathx/mathx.wsp", "public func twice(int n) int => n * 2;\n")
	write(t, root, "tools/wisp.toml", "[output]\nbin = \"bin\"\n")
	write(t, root, "tools/main.wsp", toolsMain)
	return root
}

const goodTools = `import "std/fmt";
import "geo";
import "mathx";

func main() {
	fmt.println("{} {}", geo.area(3, 4), mathx.twice(geo.doubled(5)));
}
`

func wisp(t *testing.T, dir string, args ...string) (string, int) {
	t.Helper()
	cmd := exec.Command(binary, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		return string(out), exit.ExitCode()
	}
	if err != nil {
		t.Fatal(err)
	}
	return string(out), 0
}

func TestAMemberRunsWithItsSiblingsImportedByName(t *testing.T) {
	root := workspace(t, goodTools)
	out, code := wisp(t, filepath.Join(root, "tools"), "run")
	if code != 0 || !strings.Contains(out, "12 20") {
		t.Errorf("exit %d:\n%s", code, out)
	}
}

func TestCheckAtTheRootChecksEveryMember(t *testing.T) {
	root := workspace(t, goodTools)
	out, code := wisp(t, root, "check")
	if code != 0 {
		t.Errorf("exit %d:\n%s", code, out)
	}
	for _, want := range []string{"== geo (geometry) ==", "== mathx (mathx) ==", "== tools (tools) =="} {
		if !strings.Contains(out, want) {
			t.Errorf("the output lacks %q:\n%s", want, out)
		}
	}
}

func TestABrokenMemberFailsTheWholeCommand(t *testing.T) {
	root := workspace(t, "func main( {\n")
	out, code := wisp(t, root, "check")
	if code != 1 || !strings.Contains(out, "1 of 3 projects failed") {
		t.Errorf("exit %d:\n%s", code, out)
	}
}

func TestRunAtTheRootExplainsWhereToGo(t *testing.T) {
	root := workspace(t, goodTools)
	out, code := wisp(t, root, "run")
	if code == 0 || !strings.Contains(out, "run a program from one of its members (geo, mathx, tools)") {
		t.Errorf("exit %d:\n%s", code, out)
	}
}

func TestAWorkspaceWithCodeOfItsOwnIsAnOrdinaryProject(t *testing.T) {
	root := workspace(t, goodTools)
	write(t, root, "main.wsp", "import \"std/fmt\";\nimport \"geo\";\nfunc main() { fmt.println(\"{}\", geo.area(2, 5)); }\n")
	out, code := wisp(t, root, "run")
	if code != 0 || !strings.Contains(out, "10") || strings.Contains(out, "==") {
		t.Errorf("exit %d:\n%s", code, out)
	}
}

func TestAnEmptyWorkspaceFileIsEnough(t *testing.T) {
	root := workspace(t, goodTools)
	write(t, root, "wisp.toml", "[workspace]\n")
	write(t, root, "docs/readme.txt", "not a project\n")

	out, code := wisp(t, root, "check")
	if code != 0 {
		t.Errorf("exit %d:\n%s", code, out)
	}
	for _, want := range []string{"== geo (geometry) ==", "== mathx (mathx) ==", "== tools (tools) =="} {
		if !strings.Contains(out, want) {
			t.Errorf("the output lacks %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "docs") {
		t.Errorf("a folder without a wisp.toml is not a member:\n%s", out)
	}

	write(t, root, "later/wisp.toml", "")
	write(t, root, "later/later.wsp", "public func one() int => 1;\n")
	out, _ = wisp(t, root, "check")
	if !strings.Contains(out, "== later (later) ==") {
		t.Errorf("a project added afterwards joins by itself:\n%s", out)
	}
}
