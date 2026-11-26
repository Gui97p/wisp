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
	write(t, root, "src/main.wsp", "import \"std/fmt\";\nfunc main() { fmt.println(\"{}\", twice(21)); }\n")
	write(t, root, "src/util.wsp", "func twice(int n) int => n * 2;\n")
	write(t, root, "tools/main.wsp", "import \"std/os\";\nimport \"std/fmt\";\nfunc main() { fmt.println(\"{}\", len(os.args())); }\n")
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

func programsTree(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	say := func(word string) string {
		return "import \"std/fmt\";\nfunc main() { fmt.println(\"" + word + "\"); }\n"
	}
	write(t, root, "main.wsp", say("root"))
	write(t, root, "server/main.wsp", say("server"))
	write(t, root, "tools/tools.wsp", say("tools"))
	write(t, root, "script.wsp", "#standalone\n"+say("script"))
	write(t, root, "lib/lib.wsp", "public func helper() int => 1;\n")
	return root
}

func TestProgramsAreTheFoldersAndFilesThatDefineMain(t *testing.T) {
	root := programsTree(t)
	name := filepath.Base(root)

	out, _ := wisp(t, root, "env", "programs")
	if got := strings.TrimSpace(out); got != name+", script, server, tools" && !strings.HasSuffix(got, ", script, server, tools") {
		t.Errorf("programs: %q", got)
	}
	if strings.Contains(out, "lib") {
		t.Errorf("a folder without main is not a program: %q", out)
	}

	for program, want := range map[string]string{"server": "server", "tools": "tools", "script": "script", name: "root"} {
		out, code := wisp(t, root, "run", program)
		if code != 0 || !strings.Contains(out, want) {
			t.Errorf("run %s: exit %d:\n%s", program, code, out)
		}
	}
}

func TestSeveralProgramsAskForAName(t *testing.T) {
	root := programsTree(t)
	out, code := wisp(t, root, "run")
	if code == 0 || !strings.Contains(out, "the project has several programs") || !strings.Contains(out, "script, server, tools") {
		t.Errorf("exit %d:\n%s", code, out)
	}
}

func TestBuildMakesEveryProgram(t *testing.T) {
	root := programsTree(t)
	write(t, root, "wisp.toml", "[output]\nbin = \"bin\"\n")
	if out, code := wisp(t, root, "build"); code != 0 {
		t.Fatalf("exit %d:\n%s", code, out)
	}
	for _, program := range []string{"server", "tools", "script"} {
		matches, _ := filepath.Glob(filepath.Join(root, "bin", program))
		if len(matches) == 0 {
			t.Errorf("no binary for %s", program)
		}
	}
}

func TestTheTomlOnlyCustomizesPrograms(t *testing.T) {
	root := programsTree(t)

	write(t, root, "wisp.toml", "[program.server]\nname = \"srv\"\n[output]\nbin = \"bin\"\n")
	if out, code := wisp(t, root, "build", "server"); code != 0 {
		t.Fatalf("exit %d:\n%s", code, out)
	}
	if matches, _ := filepath.Glob(filepath.Join(root, "bin", "srv")); len(matches) == 0 {
		t.Error("the binary should take the name from wisp.toml")
	}

	write(t, root, "wisp.toml", "default = \"tools\"\n")
	if out, code := wisp(t, root, "run"); code != 0 || !strings.Contains(out, "tools") {
		t.Errorf("default names a program that was found: exit %d:\n%s", code, out)
	}

	write(t, root, "wisp.toml", "[program.nothing]\nlink = []\n")
	out, code := wisp(t, root, "run", "server")
	if code == 0 || !strings.Contains(out, `"nothing" is not a program of this project`) || !strings.Contains(out, "script, server, tools") {
		t.Errorf("a declared program that does not exist: exit %d:\n%s", code, out)
	}

	write(t, root, "wisp.toml", "default = \"nothing\"\n")
	if out, code := wisp(t, root, "run"); code == 0 || !strings.Contains(out, `default names "nothing"`) {
		t.Errorf("default must be a program: exit %d:\n%s", code, out)
	}
}

func TestAnEntryStillPicksTheFileByHand(t *testing.T) {
	root := programsTree(t)
	write(t, root, "wisp.toml", "[program.old]\nentry = \"script.wsp\"\n")
	if out, code := wisp(t, root, "run", "old"); code != 0 || !strings.Contains(out, "script") {
		t.Errorf("exit %d:\n%s", code, out)
	}
	out, _ := wisp(t, root, "env", "programs")
	if strings.Contains(out, "script") {
		t.Errorf("the declared program is that file: it is not listed twice: %q", out)
	}
}

func TestProgramsWithTheSameFolderNameAreToldApartByPath(t *testing.T) {
	root := t.TempDir()
	write(t, root, "a/tool/main.wsp", "func main() {}\n")
	write(t, root, "b/tool/main.wsp", "func main() {}\n")
	out, _ := wisp(t, root, "env", "programs")
	if !strings.Contains(out, "a_tool") || !strings.Contains(out, "b_tool") {
		t.Errorf("programs: %q", out)
	}
}
