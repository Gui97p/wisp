package update

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}
}

func read(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func makeTar(t *testing.T, file string, files map[string]string) {
	t.Helper()
	f, err := os.Create(file)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	gz := gzip.NewWriter(f)
	tw := tar.NewWriter(gz)
	for name, body := range files {
		if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0o755, Size: int64(len(body)), Typeflag: tar.TypeReg}); err != nil {
			t.Fatal(err)
		}
		tw.Write([]byte(body))
	}
	tw.Close()
	gz.Close()
}

func makeZip(t *testing.T, file string, files map[string]string) {
	t.Helper()
	f, err := os.Create(file)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	zw := zip.NewWriter(f)
	for name, body := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		w.Write([]byte(body))
	}
	zw.Close()
}

func TestCompare(t *testing.T) {
	for _, tc := range []struct {
		a, b string
		want int
	}{
		{"0.39.0", "0.38.0", 1}, {"0.9.0", "0.10.0", -1}, {"1.0.0", "1.0.0", 0}, {"0.39.1", "0.39.0", 1}, {"0.39.0-rc1", "0.39.0", 0},
	} {
		if got := Compare(tc.a, tc.b); got != tc.want {
			t.Errorf("Compare(%s, %s) = %d, want %d", tc.a, tc.b, got, tc.want)
		}
	}
}

func TestAssetName(t *testing.T) {
	if got, _ := AssetName("0.39.0", "linux", "amd64"); got != "wisp-0.39.0-linux-amd64.tar.gz" {
		t.Errorf("linux: %s", got)
	}
	if got, _ := AssetName("0.39.0", "windows", "amd64"); got != "wisp-0.39.0-windows-amd64.zip" {
		t.Errorf("windows: %s", got)
	}
	if _, err := AssetName("0.39.0", "linux", "arm64"); err == nil {
		t.Error("arm64 has no release")
	}
}

func TestExtractStripsTheTopFolderAndRefusesEscapes(t *testing.T) {
	dir := t.TempDir()
	files := map[string]string{"wisp/bin/wisp": "new", "wisp/std/a.wsp": "std"}
	for _, kind := range []string{"x.tar.gz", "x.zip"} {
		archive := filepath.Join(dir, kind)
		if kind == "x.zip" {
			makeZip(t, archive, files)
		} else {
			makeTar(t, archive, files)
		}
		out := filepath.Join(dir, "out-"+kind)
		if err := Extract(archive, out); err != nil {
			t.Fatalf("%s: %v", kind, err)
		}
		if read(t, filepath.Join(out, "bin", "wisp")) != "new" || read(t, filepath.Join(out, "std", "a.wsp")) != "std" {
			t.Errorf("%s: wrong layout", kind)
		}
	}
	bad := filepath.Join(dir, "bad.tar.gz")
	makeTar(t, bad, map[string]string{"wisp/../../evil": "x"})
	if err := Extract(bad, filepath.Join(dir, "bad-out")); err == nil {
		t.Error("an escaping path must be refused")
	}
}

func TestInstallReplacesAndKeepsWhatItDoesNotKnow(t *testing.T) {
	root := t.TempDir()
	write(t, filepath.Join(root, "bin", "wisp"), "old")
	write(t, filepath.Join(root, "std", "a.wsp"), "old std")
	write(t, filepath.Join(root, "std", "gone.wsp"), "removed upstream")
	write(t, filepath.Join(root, "toolchain", "bin", "nasm"), "old nasm")

	staged, err := Stage(root)
	if err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(staged, "bin", "wisp"), "new")
	write(t, filepath.Join(staged, "std", "a.wsp"), "new std")
	write(t, filepath.Join(staged, "toolchain", "bin", "nasm"), "new nasm")

	if err := Install(root, staged); err != nil {
		t.Fatal(err)
	}
	if read(t, filepath.Join(root, "bin", "wisp")) != "new" || read(t, filepath.Join(root, "std", "a.wsp")) != "new std" || read(t, filepath.Join(root, "toolchain", "bin", "nasm")) != "new nasm" {
		t.Error("the files were not replaced")
	}
	if _, err := os.Stat(filepath.Join(root, "std", "gone.wsp")); err == nil {
		t.Error("a file the new release dropped must go away")
	}
	entries, _ := os.ReadDir(filepath.Join(root, "bin"))
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), oldSuffix) {
			t.Errorf("leftover %s", e.Name())
		}
	}
}

func TestRootRefusesADevelopmentCheckout(t *testing.T) {
	repo := t.TempDir()
	write(t, filepath.Join(repo, "go.mod"), "module x")
	write(t, filepath.Join(repo, "std", "a.wsp"), "")
	write(t, filepath.Join(repo, "bin", "wisp"), "")
	if _, err := Root(filepath.Join(repo, "bin", "wisp")); err == nil {
		t.Error("a checkout with go.mod is not an installation")
	}
	pkg := t.TempDir()
	write(t, filepath.Join(pkg, "std", "a.wsp"), "")
	write(t, filepath.Join(pkg, "bin", "wisp"), "")
	if _, err := Root(filepath.Join(pkg, "bin", "wisp")); err != nil {
		t.Errorf("a package layout is fine: %v", err)
	}
}

func TestLatestVerifyAndDownload(t *testing.T) {
	archive := filepath.Join(t.TempDir(), "wisp-0.39.0-linux-amd64.tar.gz")
	makeTar(t, archive, map[string]string{"wisp/std/a.wsp": "x"})
	body, _ := os.ReadFile(archive)
	sum := sha256.Sum256(body)
	sums := fmt.Sprintf("%s  wisp-0.39.0-linux-amd64.tar.gz\n", hex.EncodeToString(sum[:]))

	mux := http.NewServeMux()
	mux.HandleFunc("/releases/latest", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/releases/tag/v0.39.0", http.StatusFound)
	})
	mux.HandleFunc("/releases/download/v0.39.0/SHA256SUMS", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(sums)) })
	mux.HandleFunc("/releases/download/v0.39.0/wisp-0.39.0-linux-amd64.tar.gz", func(w http.ResponseWriter, r *http.Request) { w.Write(body) })
	server := httptest.NewServer(mux)
	defer server.Close()
	old := BaseURL
	BaseURL = server.URL
	defer func() { BaseURL = old }()

	r, err := Latest()
	if err != nil || r.Version != "0.39.0" || r.Tag != "v0.39.0" {
		t.Fatalf("Latest = %+v, %v", r, err)
	}

	dir := t.TempDir()
	sumsFile, got := filepath.Join(dir, "sums"), filepath.Join(dir, "a.tar.gz")
	if err := Download(AssetURL(r, SumsName), sumsFile, nil); err != nil {
		t.Fatal(err)
	}
	if err := Download(AssetURL(r, "wisp-0.39.0-linux-amd64.tar.gz"), got, nil); err != nil {
		t.Fatal(err)
	}
	if err := Verify(sumsFile, "wisp-0.39.0-linux-amd64.tar.gz", got); err != nil {
		t.Errorf("a good archive must verify: %v", err)
	}
	write(t, got, "tampered")
	if err := Verify(sumsFile, "wisp-0.39.0-linux-amd64.tar.gz", got); err == nil {
		t.Error("a tampered archive must fail")
	}
}

func TestLatestIgnoresReleasesThatAreNotVersions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/releases/tag/cc-tweaked-final", http.StatusFound)
	}))
	defer server.Close()
	old := BaseURL
	BaseURL = server.URL
	defer func() { BaseURL = old }()
	if _, err := Latest(); err == nil {
		t.Error("a tag that is not vX.Y.Z is not a release")
	}
}
