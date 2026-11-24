package native

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

func project(t *testing.T, toml, main string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "wisp.toml"), []byte(toml), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "main.wsp"), []byte(main), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
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

func buildSharedLibrary(t *testing.T, dir, soname, file string) {
	t.Helper()
	cc, err := exec.LookPath("cc")
	if err != nil || runtime.GOOS != "linux" {
		t.Skip("needs cc on Linux")
	}
	src := filepath.Join(dir, "greet.c")
	if err := os.WriteFile(src, []byte("int greet_twice(int x) { return x * 2; }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "lib"), 0o755); err != nil {
		t.Fatal(err)
	}
	args := []string{"-shared", "-fPIC", "-o", filepath.Join(dir, "lib", file), src}
	if soname != "" {
		args = append(args[:2], append([]string{"-Wl,-soname," + soname}, args[2:]...)...)
	}
	if out, err := exec.Command(cc, args...).CombinedOutput(); err != nil {
		t.Fatalf("cc: %v\n%s", err, out)
	}
}

const program = `import "std/fmt";

extern func greet_twice(int32 x) int32;

func main() {
	fmt.printf("{}\n", greet_twice(21));
}
`

func TestProgramRunsWithASharedLibraryFromASearchFolder(t *testing.T) {
	dir := project(t, "[native]\nlibs = [\"greet\"]\nsearch = [\"lib\"]\n", program)
	buildSharedLibrary(t, dir, "", "libgreet.so")
	out, code := wisp(t, dir, "run")
	if code != 0 || !strings.Contains(out, "42") {
		t.Errorf("exit %d:\n%s", code, out)
	}
}

func TestOnlyTheVersionedLibraryIsEnough(t *testing.T) {
	dir := project(t, "[native]\nlibs = [\"greet\"]\nsearch = [\"lib\"]\n", program)
	buildSharedLibrary(t, dir, "libgreet.so.1", "libgreet.so.1")
	out, code := wisp(t, dir, "run")
	if code != 0 || !strings.Contains(out, "42") {
		t.Errorf("exit %d:\n%s", code, out)
	}
}

func TestSystemTablesAreChosenByTheTarget(t *testing.T) {
	dir := project(t, "[native.linux]\nlibs = [\"greet\"]\nsearch = [\"lib\"]\n[native.windows]\nlibs = [\"nothing\"]\n", program)
	buildSharedLibrary(t, dir, "", "libgreet.so")
	out, code := wisp(t, dir, "run")
	if code != 0 || !strings.Contains(out, "42") {
		t.Errorf("the linux table is the one that counts, exit %d:\n%s", code, out)
	}
}

func TestMissingLibraryAndBadKeysAreReported(t *testing.T) {
	dir := project(t, "[native]\nlibs = [\"nothing\"]\n", "func main() {}\n")
	out, code := wisp(t, dir, "build")
	if code == 0 || !strings.Contains(out, "the native library nothing was not found") || !strings.Contains(out, "search in [native]") {
		t.Errorf("exit %d:\n%s", code, out)
	}

	dir = project(t, "[native]\nlib = [\"x\"]\n[native.plan9]\nlibs = []\n", "func main() {}\n")
	out, code = wisp(t, dir, "build")
	if code == 0 || !strings.Contains(out, "did you mean libs") || !strings.Contains(out, `"plan9" is not a system`) {
		t.Errorf("exit %d:\n%s", code, out)
	}
}
