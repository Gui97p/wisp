package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/Gui97p/wisp/internal/ast"
	"github.com/Gui97p/wisp/internal/module"
	"github.com/Gui97p/wisp/internal/target"
	"github.com/spf13/cobra"
)

var (
	testRun        bool
	testFilter     string
	testJobs       int
	testTimeout    time.Duration
	testList       bool
	testShowOutput bool
	testReport     bool
)

var (
	errNoTests = errors.New("the project has no tests")
	errNoMatch = errors.New("no test matches")
	testKeep   func(module.TestFunc) bool
)

var testCmd = &cobra.Command{
	Use:   "test [path...]",
	Short: "Compile and run the tests of the project",
	Long: `Compile the project with its tests and run them, each one in a process of its own.

A test is a function marked #test, with no parameters and no result, or only an error:

    #test("adding two numbers")
    func addition() ! { ... }

It can sit in the file it tests, in a file next to it named name.test.wsp, which sees what is internal, or in any other folder of the project. A test passes when it returns, and fails when it returns an error or stops the program, as a panic does.

Without arguments every test runs. A path (a folder or a file) restricts the run to the tests under it.`,
	Args:         cobra.ArbitraryArgs,
	RunE:         runTests,
	SilenceUsage: true,
}

func init() {
	f := testCmd.Flags()
	f.StringVarP(&testFilter, "run", "r", "", "run only the tests whose module and name match this regular expression")
	f.IntVarP(&testJobs, "jobs", "j", 0, "tests running at once (the number of processors by default)")
	f.DurationVar(&testTimeout, "timeout", 30*time.Second, "time limit of each test")
	f.BoolVar(&testList, "list", false, "list the tests without running them")
	f.BoolVar(&testReport, "report", false, "print the place and time of every test, and what a failure says right under it")
	f.BoolVar(&testShowOutput, "show-output", false, "print what passing tests wrote too")
	f.StringVar(&osOpt, "os", "", "system to test on (linux, windows)")
	f.StringVar(&archOpt, "arch", "", "architecture to test on (x64)")
	f.StringArrayVar(&linkOpt, "link", nil, "extra argument for the linker, repeatable")
	f.BoolVar(&libcOpt, "libc", false, "link the system C library")
	addProfileFlags(testCmd)
	addProjectFlags(testCmd)
	addOutputFlags(testCmd)
}

func generateTestMain(tests []module.TestFunc) string {
	aliases := map[string]string{}
	var order []string
	for _, test := range tests {
		if _, seen := aliases[test.Module]; !seen {
			aliases[test.Module] = fmt.Sprintf("t%d", len(order))
			order = append(order, test.Module)
		}
	}

	var b strings.Builder
	b.WriteString("import \"std/os\";\nimport \"std/fmt\";\nimport \"std/testing\";\n")
	for _, path := range order {
		fmt.Fprintf(&b, "import %q as %s;\n", path, aliases[path])
	}
	b.WriteString("\nfunc main() ! {\n")
	b.WriteString("  []string args = os.args();\n")
	b.WriteString("  if len(args) != 2 {\n    fmt.fprintln(2, \"usage: a test binary takes the id of one test\");\n    os.exit(64);\n  }\n")
	b.WriteString("  switch args[1] {\n")
	for _, test := range tests {
		bang := ""
		if test.Fallible {
			bang = "!"
		}
		fmt.Fprintf(&b, "    case %q => {\n      %s.%s()%s;\n      testing.finish()!;\n    }\n", test.ID(), aliases[test.Module], test.Name, bang)
	}
	b.WriteString("    default => {\n      fmt.fprint(2, \"there is no test \");\n      fmt.fprintln(2, args[1]);\n      os.exit(64);\n    }\n  }\n}\n")
	return b.String()
}

func findTestMain(compiled []compiledModule) (*ast.FuncDecl, *compiledModule, int) {
	for i := range compiled {
		if compiled[i].mod.Path != module.TestMainPath {
			continue
		}
		for _, decl := range compiled[i].merged.Declarations {
			if fd, ok := decl.(*ast.FuncDecl); ok && fd.Receiver == nil && fd.Name == "main" {
				return fd, &compiled[i], 1
			}
		}
	}
	return nil, nil, 0
}

func testTargets(root string, tgt target.Target) buildTargets {
	base := filepath.Join(root, "build", profileDir(tgt), "test")
	return buildTargets{
		asmDir:  filepath.Join(base, "asm"),
		objDir:  filepath.Join(base, "obj"),
		binPath: filepath.Join(base, executableName(tgt, projectName(root)+".test")),
		wantBin: true,
	}
}

func testName(t module.TestFunc) string {
	return t.Module + ": " + t.Display
}

func testPlace(t module.TestFunc) string {
	return fmt.Sprintf("%s:%d", relPath(t.File), t.Line)
}

func testSelector(args []string) (func(module.TestFunc) bool, error) {
	var filter *regexp.Regexp
	if testFilter != "" {
		var err error
		if filter, err = regexp.Compile(testFilter); err != nil {
			return nil, fmt.Errorf("--run %q is not a regular expression: %v", testFilter, err)
		}
	}

	var scopes []string
	for _, arg := range args {
		abs, err := filepath.Abs(arg)
		if err != nil {
			return nil, err
		}
		if _, err := os.Stat(abs); err != nil {
			return nil, fmt.Errorf("%s is not a file or folder of the project (to pick tests by name, use --run %s)", arg, arg)
		}
		scopes = append(scopes, abs)
	}

	return func(t module.TestFunc) bool {
		if len(scopes) > 0 {
			inside := false
			for _, scope := range scopes {
				if t.File == scope || strings.HasPrefix(t.File, scope+string(os.PathSeparator)) {
					inside = true
				}
			}
			if !inside {
				return false
			}
		}
		return filter == nil || filter.MatchString(testName(t)) || filter.MatchString(t.ID())
	}, nil
}

func sortTests(tests []module.TestFunc) {
	sort.SliceStable(tests, func(i, j int) bool {
		if tests[i].File != tests[j].File {
			return tests[i].File < tests[j].File
		}
		return tests[i].Line < tests[j].Line
	})
}

const skipExitCode = 77

type testOutcome struct {
	test    module.TestFunc
	passed  bool
	skipped bool
	note    string
	reason  string
	output  string
	took    time.Duration
}

func runOneTest(bin string, env []module.EnvVar, test module.TestFunc) testOutcome {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, resolveExecutable(bin), test.ID())
	cmd.Env = os.Environ()
	for _, v := range env {
		if _, set := os.LookupEnv(v.Name); !set {
			cmd.Env = append(cmd.Env, v.Name+"="+v.Value)
		}
	}
	var out bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &out

	start := time.Now()
	err := cmd.Run()
	outcome := testOutcome{test: test, output: out.String(), took: time.Since(start)}

	timedOut := ctx.Err() == context.DeadlineExceeded
	code, signal := 0, ""
	if err != nil {
		var exitErr *exec.ExitError
		switch {
		case timedOut:
		case errors.As(err, &exitErr):
			code = exitErr.ExitCode()
			if status, ok := exitErr.Sys().(syscall.WaitStatus); ok && status.Signaled() {
				signal = status.Signal().String()
			}
		default:
			outcome.reason = err.Error()
			return outcome
		}
	}

	switch {
	case timedOut:
		outcome.reason = fmt.Sprintf("timed out after %s", testTimeout)
	case test.ExpectAbort:
		aborted := err != nil && code != skipExitCode
		switch {
		case !aborted:
			outcome.reason = "was expected to stop the program, and returned"
		case test.AbortText != "" && !strings.Contains(outcome.output, test.AbortText):
			outcome.reason = fmt.Sprintf("stopped the program, but what it printed lacks %q", test.AbortText)
		default:
			outcome.passed = true
			outcome.note = "stopped the program, as expected"
		}
	case err == nil:
		outcome.passed = true
	case code == skipExitCode && signal == "":
		outcome.skipped = true
		outcome.passed = true
	case signal != "":
		outcome.reason = "crashed with " + signal
	default:
		outcome.reason = fmt.Sprintf("exit status %d", code)
	}
	return outcome
}

func indent(text, prefix string) string {
	text = strings.TrimRight(text, "\n")
	if text == "" {
		return ""
	}
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lines[i] = prefix + line
	}
	return strings.Join(lines, "\n") + "\n"
}

func runTests(cmd *cobra.Command, args []string) error {
	start := time.Now()
	module.ConfigureTests(true)
	testRun = true

	root, base, err := openProject(cmd)
	if err != nil {
		return err
	}
	cfg, tgt, err := settle(base)
	if err != nil {
		return err
	}
	cfg.Entry = ""

	if tgt.Arch != "x64" {
		return fmt.Errorf("target %q has no build pipeline", tgt.Name)
	}

	keep, err := testSelector(args)
	if err != nil {
		return err
	}
	testKeep = keep

	if testList {
		module.ConfigureTarget(tgt.Name)
		target.SetActiveProfile(cfg.ProfileFacts(tgt.Profile))
		_, tests, _, err := module.DiscoverProjectWithTests(root, keep, generateTestMain)
		if err != nil {
			return err
		}
		sortTests(tests)
		for _, t := range tests {
			fmt.Printf("%s  (%s)\n", testName(t), testPlace(t))
		}
		return nil
	}

	if verboseOpt {
		detail("Testing %s", tgt.Name)
	}
	res, err := buildX64(root, cfg, tgt, true, buildJob{name: projectName(root), tests: true})
	if errors.Is(err, errNoTests) {
		status("There are no tests: mark a function with #test")
		return nil
	}
	if errors.Is(err, errNoMatch) {
		status("No test matches")
		return nil
	}
	if err != nil {
		return err
	}

	tests := res.tests
	sortTests(tests)

	jobs := testJobs
	if jobs <= 0 {
		jobs = runtime.NumCPU()
	}
	env := cfg.RunEnv()

	fmt.Printf("Running %s\n", count(len(tests), "test"))
	slots := make(chan struct{}, jobs)
	done := make([]chan testOutcome, len(tests))
	for i, test := range tests {
		done[i] = make(chan testOutcome, 1)
		go func() {
			slots <- struct{}{}
			done[i] <- runOneTest(res.binPath, env, test)
			<-slots
		}()
	}

	report := testReport || verboseOpt
	dots := quietOpt && !report
	passed, failed, skipped := 0, 0, 0
	var failures []testOutcome
	var details []string
	marks := 0
	for i := range tests {
		o := <-done[i]
		label, mark := "ok  ", "."
		switch {
		case o.skipped:
			label, mark = "skip", "s"
			skipped++
		case !o.passed:
			label, mark = "FAIL", "F"
			failed++
			failures = append(failures, o)
		default:
			passed++
		}
		line := fmt.Sprintf("%s  %s  (%s, %s)", label, testName(o.test), testPlace(o.test), shortDuration(o.took))
		switch {
		case !o.passed:
			line += ": " + o.reason
		case o.note != "":
			line += ": " + o.note
		}
		text := line + "\n"
		if !o.passed || o.skipped || testShowOutput {
			text += indent(o.output, "      ")
		}
		if report {
			fmt.Print(text)
			continue
		}
		if dots {
			fmt.Print(mark)
			marks++
			if marks%60 == 0 {
				fmt.Println()
			}
		} else {
			fmt.Printf("%s  %s\n", label, testName(o.test))
		}
		if !o.passed || o.skipped || testShowOutput {
			details = append(details, text)
		}
	}
	if !report {
		if dots && marks%60 != 0 {
			fmt.Println()
		}
		if len(details) > 0 {
			fmt.Println()
			for i, text := range details {
				if i > 0 {
					fmt.Println()
				}
				fmt.Print(text)
			}
		}
	}

	fmt.Println()
	if failed > 0 && report {
		fmt.Println("Failed:")
		for _, o := range failures {
			fmt.Printf("  %s  (%s)\n", testName(o.test), testPlace(o.test))
		}
		fmt.Println()
	}
	summary := fmt.Sprintf("%d passed, %d failed", passed, failed)
	if skipped > 0 {
		summary += fmt.Sprintf(", %d skipped", skipped)
	}
	summary += " in " + elapsed(start)
	if failed > 0 {
		fmt.Println("Test result: FAILED. " + summary)
		os.Exit(1)
	}
	fmt.Println("Test result: ok. " + summary)
	return nil
}

func shortDuration(d time.Duration) string {
	if d < time.Millisecond {
		return "<1ms"
	}
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	return fmt.Sprintf("%.2fs", d.Seconds())
}
