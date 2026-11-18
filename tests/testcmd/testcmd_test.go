package testcmd

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

func copyTree(t *testing.T, from, to string) {
	t.Helper()
	err := filepath.WalkDir(from, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(from, path)
		target := filepath.Join(to, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}
}

func run(t *testing.T, project string, args ...string) (string, int) {
	t.Helper()
	dir := t.TempDir()
	copyTree(t, filepath.Join(repoRoot(), "tests", "testcmd", "projects", project), dir)
	cmd := exec.Command(binary, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	code := 0
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		code = exit.ExitCode()
	} else if err != nil {
		t.Fatal(err)
	}
	return string(out), code
}

func expect(t *testing.T, out string, wants ...string) {
	t.Helper()
	for _, want := range wants {
		if !strings.Contains(out, want) {
			t.Errorf("the output lacks %q:\n%s", want, out)
		}
	}
}

func refuse(t *testing.T, out string, unwanted ...string) {
	t.Helper()
	for _, bad := range unwanted {
		if strings.Contains(out, bad) {
			t.Errorf("the output has %q:\n%s", bad, out)
		}
	}
}

func TestRunsEveryTest(t *testing.T) {
	out, code := run(t, "basic", "test")
	if code != 1 {
		t.Errorf("exit code %d, want 1", code)
	}
	expect(t, out,
		"Running 5 tests",
		"ok    calc.test: an internal function is visible to the file next to it",
		"ok    calc: add works in the file itself",
		"ok    tests/blackbox: public api from another folder",
		"FAIL  calc.test: failsWithError",
		"FAIL  calc.test: an index out of range aborts",
		"exit status 4",
		"error[std/errors/kind.Kind]: Kind.Invalid",
		"panic: index out of range",
		"calc.test.wsp:19:3",
		"Failed:",
		"Test result: FAILED. 3 passed, 2 failed",
	)
	refuse(t, out, ".wisp-test-main")
}

func TestFilterByName(t *testing.T) {
	out, code := run(t, "basic", "test", "--run", "internal")
	if code != 0 {
		t.Errorf("exit code %d, want 0", code)
	}
	expect(t, out, "Running 1 test", "an internal function is visible", "Test result: ok. 1 passed, 0 failed")
	refuse(t, out, "failsWithError", "blackbox")
}

func TestFilterByPath(t *testing.T) {
	out, code := run(t, "basic", "test", "tests")
	if code != 0 {
		t.Errorf("exit code %d, want 0", code)
	}
	expect(t, out, "Running 1 test", "tests/blackbox: public api from another folder")
	refuse(t, out, "calc.test")

	out, code = run(t, "basic", "test", "calc.wsp")
	if code != 0 {
		t.Errorf("exit code %d, want 0", code)
	}
	expect(t, out, "Running 1 test", "calc: add works in the file itself")
}

func TestUnknownPathPointsToRun(t *testing.T) {
	out, code := run(t, "basic", "test", "nothing")
	if code == 0 {
		t.Errorf("exit code 0, want a failure")
	}
	expect(t, out, "not a file or folder of the project", "--run nothing")
}

func TestListDoesNotRun(t *testing.T) {
	out, code := run(t, "basic", "test", "--list")
	if code != 0 {
		t.Errorf("exit code %d, want 0", code)
	}
	expect(t, out, "calc.test: failsWithError  (", "calc: add works in the file itself  (", "tests/blackbox: public api from another folder  (")
	refuse(t, out, "Running", "Test result")
}

func TestNoTests(t *testing.T) {
	out, code := run(t, "empty", "test")
	if code != 0 {
		t.Errorf("exit code %d, want 0", code)
	}
	expect(t, out, "There are no tests")
}

func TestNoMatch(t *testing.T) {
	out, code := run(t, "basic", "test", "--run", "zzzz")
	if code != 0 {
		t.Errorf("exit code %d, want 0", code)
	}
	expect(t, out, "No test matches")
}

func TestTimeout(t *testing.T) {
	out, code := run(t, "hang", "test", "--timeout", "1s")
	if code != 1 {
		t.Errorf("exit code %d, want 1", code)
	}
	expect(t, out, "FAIL  hang.test: never ends", "timed out after 1s", "ok    hang.test: ends at once", "1 passed, 1 failed")
}

func TestShowOutput(t *testing.T) {
	out, _ := run(t, "basic", "test", "--run", "internal", "--show-output")
	expect(t, out, "ok    calc.test: an internal function")
}

func TestOrdinaryBuildLeavesTestsOut(t *testing.T) {
	out, code := run(t, "withmain", "run")
	if code != 0 {
		t.Errorf("a broken test file stopped the build (exit %d):\n%s", code, out)
	}
	expect(t, out, "4")

	out, code = run(t, "withmain", "check")
	if code == 0 {
		t.Errorf("check ignored a broken test file:\n%s", out)
	}
	expect(t, out, "calc.test.wsp")
}

func TestBrokenTestFileIsReportedOnce(t *testing.T) {
	out, code := run(t, "withmain", "test")
	if code == 0 {
		t.Errorf("exit code 0, want a failure")
	}
	expect(t, out, "calc.test.wsp")
	refuse(t, out, ".wisp-test-main", "has errors")
}

func TestOnlyWhatTheTestsNeedIsAnalysed(t *testing.T) {
	out, code := run(t, "basic", "test", "tests")
	if code != 0 {
		t.Errorf("exit code %d, want 0", code)
	}
	refuse(t, out, "noisy", "never used", "warning")

	out, _ = run(t, "basic", "test", "noisy")
	expect(t, out, "No test matches")
	refuse(t, out, "noisy", "warning")
}

func TestTestingModule(t *testing.T) {
	out, code := run(t, "testing", "test")
	if code != 1 {
		t.Errorf("exit code %d, want 1", code)
	}
	expect(t, out,
		"ok    t.test: expect passes",
		"ok    t.test: expectKind",
		"ok    t.test: expectApprox and strings",
		"ok    t.test: freed memory is fine",
		"FAIL  t.test: expect fails with a message",
		"expected 4, got 5",
		"at t.test.wsp:13:3: in t.test.failsWithMessage",
		"strings differ at byte 6",
		`want: "hello world"`,
		"expected 1, got 2: the difference 1 is more than 0.5",
		"FAIL  t.test: a leak is a failure",
		"leak: 1 blocks, 64 bytes",
		"memory leaked",
		"skip  t.test: skipped on purpose",
		"not for this system",
		"ok    t.test: aborts as expected",
		"stopped the program, as expected",
		"FAIL  t.test: should abort but returns",
		"was expected to stop the program, and returned",
		"FAIL  t.test: aborts with other text",
		`what it printed lacks "boom"`,
		"Test result: FAILED. 5 passed, 6 failed, 1 skipped",
	)
}
