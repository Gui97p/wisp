package runpath

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

func tree(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	write(t, root, "src/main.wsp", "import \"std/fmt\";\nfunc main() { fmt.printf(\"{}\\n\", twice(21)); }\n")
	write(t, root, "src/util.wsp", "func twice(int n) int => n * 2;\n")
	write(t, root, "tools/main.wsp", "import \"std/os\";\nimport \"std/fmt\";\nfunc main() { fmt.printf(\"{}\\n\", len(os.args())); }\n")
	write(t, root, "other.wsp", "#standalone\nimport \"std/fmt\";\nfunc main() { fmt.println(\"other\"); }\n")
	return root
}

func TestAFileRunsWithoutAProjectFile(t *testing.T) {
	root := tree(t)
	out, code := wisp(t, root, "run", "src/main.wsp")
	if code != 0 || !strings.Contains(out, "42") {
		t.Errorf("exit %d:\n%s", code, out)
	}
	out, code = wisp(t, root, "run", "other.wsp")
	if code != 0 || !strings.Contains(out, "other") {
		t.Errorf("a file at the root, exit %d:\n%s", code, out)
	}
}

func TestAFolderRunsItsMain(t *testing.T) {
	root := tree(t)
	out, code := wisp(t, root, "run", "tools/", "--", "a", "b")
	if code != 0 || !strings.Contains(out, "3") {
		t.Errorf("the arguments after -- reach the program, exit %d:\n%s", code, out)
	}
}

func TestAPathFindsTheProjectOfTheFile(t *testing.T) {
	root := tree(t)
	write(t, root, "wisp.toml", "[output]\nbin = \"bin\"\n")
	out, code := wisp(t, filepath.Join(root, "tools"), "run", "../src/main.wsp")
	if code != 0 || !strings.Contains(out, "42") {
		t.Errorf("exit %d:\n%s", code, out)
	}
}

func TestBuildTakesAPathToo(t *testing.T) {
	root := tree(t)
	out, code := wisp(t, root, "build", "src/main.wsp")
	if code != 0 || !strings.Contains(out, "Finished") {
		t.Errorf("exit %d:\n%s", code, out)
	}
}

func TestBadPathsExplainThemselves(t *testing.T) {
	root := tree(t)
	write(t, root, "empty/readme.txt", "")
	for _, tc := range []struct{ arg, want string }{
		{"nothing.wsp", "nothing.wsp does not exist"},
		{"empty/", "has no main.wsp or empty.wsp to start from"},
		{"empty/readme.txt", "is not a Wisp file"},
	} {
		out, code := wisp(t, root, "run", tc.arg)
		if code == 0 || !strings.Contains(out, tc.want) {
			t.Errorf("%s: exit %d:\n%s", tc.arg, code, out)
		}
	}
}

func TestAProgramNameStillWorks(t *testing.T) {
	root := t.TempDir()
	write(t, root, "wisp.toml", "[program.hello]\nentry = \"hello.wsp\"\n")
	write(t, root, "hello.wsp", "import \"std/fmt\";\nfunc main() { fmt.println(\"hello\"); }\n")
	out, code := wisp(t, root, "run", "hello")
	if code != 0 || !strings.Contains(out, "hello") {
		t.Errorf("exit %d:\n%s", code, out)
	}
}
