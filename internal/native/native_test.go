package native

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func touch(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, nil, 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestNothingDeclaredAddsNothing(t *testing.T) {
	res, err := Resolve("elf64", Spec{}, t.TempDir(), false)
	if err != nil || len(res.Args) != 0 || res.Interpreter != "" {
		t.Errorf("got %+v, %v", res, err)
	}
}

func TestFindsALibraryInASearchFolderAndAsksForTheLoader(t *testing.T) {
	root := t.TempDir()
	touch(t, filepath.Join(root, "libs", "libgreet.so"))
	loader := filepath.Join(root, "ld.so")
	touch(t, loader)

	res, err := Resolve("elf64", Spec{Libs: []string{"greet"}, Search: []string{"libs"}, Interpreter: loader}, root, false)
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(res.Args, " ")
	for _, want := range []string{"-L" + filepath.Join(root, "libs"), "-lgreet", "--dynamic-linker " + loader} {
		if !strings.Contains(joined, want) {
			t.Errorf("the arguments lack %q: %s", want, joined)
		}
	}
}

func TestAVersionedLibraryWithoutTheDevelopmentLink(t *testing.T) {
	root := t.TempDir()
	touch(t, filepath.Join(root, "libs", "libgreet.so.0"))
	touch(t, filepath.Join(root, "libs", "libgreet.so.1"))

	res, err := Resolve("elf64", Spec{Libs: []string{"greet"}, Search: []string{"libs"}, Interpreter: filepath.Join(root, "libs", "libgreet.so.0")}, root, true)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(res.Args, " "), "-l:libgreet.so.1") {
		t.Errorf("the newest version should be used: %v", res.Args)
	}
	if strings.Contains(strings.Join(res.Args, " "), "--dynamic-linker") {
		t.Errorf("with libc the C driver chooses the loader: %v", res.Args)
	}
}

func TestMissingLibraryAndFolderExplainThemselves(t *testing.T) {
	root := t.TempDir()
	if _, err := Resolve("elf64", Spec{Libs: []string{"nothing"}}, root, false); err == nil || !strings.Contains(err.Error(), "install it, or add its folder to search") {
		t.Errorf("the missing library: %v", err)
	}
	if _, err := Resolve("elf64", Spec{Libs: []string{"x"}, Search: []string{"nowhere"}}, root, false); err == nil || !strings.Contains(err.Error(), "does not exist") {
		t.Errorf("the missing folder: %v", err)
	}
}

func TestWindowsLooksForImportLibrariesAndNeedsNoLoader(t *testing.T) {
	root := t.TempDir()
	touch(t, filepath.Join(root, "win", "libSDL3.dll.a"))
	res, err := Resolve("win64", Spec{Libs: []string{"SDL3"}, Search: []string{"win"}}, root, false)
	if err != nil {
		t.Fatal(err)
	}
	if res.Interpreter != "" || strings.Contains(strings.Join(res.Args, " "), "dynamic-linker") {
		t.Errorf("windows has no loader to name: %+v", res)
	}
}
