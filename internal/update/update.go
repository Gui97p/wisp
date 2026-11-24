package update

import (
	"archive/tar"
	"archive/zip"
	"bufio"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const Repo = "Gui97p/wisp"

const SumsName = "SHA256SUMS"

var BaseURL = "https://github.com/" + Repo

type Release struct {
	Tag     string
	Version string
}

func client() *http.Client {
	return &http.Client{Timeout: 10 * time.Minute}
}

func Latest() (Release, error) {
	noRedirect := &http.Client{
		Timeout:       30 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
	resp, err := noRedirect.Get(BaseURL + "/releases/latest")
	if err != nil {
		return Release{}, fmt.Errorf("could not reach the release page (%w)", err)
	}
	defer resp.Body.Close()
	location := resp.Header.Get("Location")
	if resp.StatusCode < 300 || resp.StatusCode > 399 || location == "" {
		return Release{}, errors.New("no release has been published yet")
	}
	u, err := url.Parse(location)
	if err != nil {
		return Release{}, err
	}
	tag := path.Base(u.Path)
	if !tagPattern.MatchString(tag) {
		return Release{}, errors.New("no release has been published yet")
	}
	return ForTag(tag), nil
}

var tagPattern = regexp.MustCompile(`^v\d+\.\d+\.\d+$`)

func ForTag(tag string) Release {
	if !strings.HasPrefix(tag, "v") {
		tag = "v" + tag
	}
	return Release{Tag: tag, Version: strings.TrimPrefix(tag, "v")}
}

func parse(v string) [3]int {
	v, _, _ = strings.Cut(v, "-")
	var out [3]int
	for i, part := range strings.SplitN(v, ".", 3) {
		out[i], _ = strconv.Atoi(part)
	}
	return out
}

func Compare(a, b string) int {
	pa, pb := parse(a), parse(b)
	for i := range pa {
		if pa[i] != pb[i] {
			if pa[i] < pb[i] {
				return -1
			}
			return 1
		}
	}
	return 0
}

func AssetName(version, goos, goarch string) (string, error) {
	if goarch != "amd64" {
		return "", fmt.Errorf("there is no release for %s/%s yet", goos, goarch)
	}
	switch goos {
	case "linux":
		return fmt.Sprintf("wisp-%s-linux-amd64.tar.gz", version), nil
	case "windows":
		return fmt.Sprintf("wisp-%s-windows-amd64.zip", version), nil
	}
	return "", fmt.Errorf("there is no release for %s/%s yet", goos, goarch)
}

func AssetURL(r Release, name string) string {
	return fmt.Sprintf("%s/releases/download/%s/%s", BaseURL, r.Tag, name)
}

func Download(from, to string, progress func(done, total int64)) error {
	resp, err := client().Get(from)
	if err != nil {
		return fmt.Errorf("could not download %s (%w)", from, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("could not download %s (%s)", from, resp.Status)
	}
	out, err := os.Create(to)
	if err != nil {
		return err
	}
	defer out.Close()
	var done int64
	buf := make([]byte, 64*1024)
	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			if _, err := out.Write(buf[:n]); err != nil {
				return err
			}
			done += int64(n)
			if progress != nil {
				progress(done, resp.ContentLength)
			}
		}
		if readErr == io.EOF {
			return nil
		}
		if readErr != nil {
			return readErr
		}
	}
}

func Verify(sumsFile, name, archive string) error {
	f, err := os.Open(sumsFile)
	if err != nil {
		return err
	}
	defer f.Close()
	want := ""
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) == 2 && strings.TrimPrefix(fields[1], "*") == name {
			want = strings.ToLower(fields[0])
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	if want == "" {
		return fmt.Errorf("%s has no checksum for %s", SumsName, name)
	}
	a, err := os.Open(archive)
	if err != nil {
		return err
	}
	defer a.Close()
	h := sha256.New()
	if _, err := io.Copy(h, a); err != nil {
		return err
	}
	if got := hex.EncodeToString(h.Sum(nil)); got != want {
		return fmt.Errorf("the checksum of %s does not match (got %s, want %s)", name, got, want)
	}
	return nil
}

func inside(dest, name string) (string, error) {
	clean := path.Clean(strings.ReplaceAll(name, "\\", "/"))
	if clean == "." {
		return "", nil
	}
	if path.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, "../") {
		return "", fmt.Errorf("the archive has an unsafe path %q", name)
	}
	_, rest, found := strings.Cut(clean, "/")
	if !found {
		return "", nil
	}
	return filepath.Join(dest, filepath.FromSlash(rest)), nil
}

func Extract(archive, dest string) error {
	if strings.HasSuffix(archive, ".zip") {
		return extractZip(archive, dest)
	}
	return extractTar(archive, dest)
}

func extractTar(archive, dest string) error {
	f, err := os.Open(archive)
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		target, err := inside(dest, header.Name)
		if err != nil {
			return err
		}
		if target == "" {
			continue
		}
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeSymlink:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			if err := os.Symlink(header.Linkname, target); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := writeFile(target, tr, os.FileMode(header.Mode)&0o777); err != nil {
				return err
			}
		}
	}
}

func extractZip(archive, dest string) error {
	r, err := zip.OpenReader(archive)
	if err != nil {
		return err
	}
	defer r.Close()
	for _, file := range r.File {
		target, err := inside(dest, file.Name)
		if err != nil {
			return err
		}
		if target == "" {
			continue
		}
		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
			continue
		}
		src, err := file.Open()
		if err != nil {
			return err
		}
		err = writeFile(target, src, file.Mode()&0o777)
		src.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func writeFile(target string, src io.Reader, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode|0o600)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, src); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

const oldSuffix = ".old"

func Root(exe string) (string, error) {
	exe, err := filepath.EvalSymlinks(exe)
	if err != nil {
		return "", err
	}
	root := filepath.Dir(filepath.Dir(exe))
	for _, dev := range []string{"go.mod", ".git"} {
		if _, err := os.Stat(filepath.Join(root, dev)); err == nil {
			return "", fmt.Errorf("%s is a development checkout, not an installation: update it with git and make", root)
		}
	}
	if info, err := os.Stat(filepath.Join(root, "std")); err != nil || !info.IsDir() {
		return "", fmt.Errorf("%s does not look like a wisp installation (no std folder)", root)
	}
	return root, nil
}

func Cleanup(root string) {
	for _, dir := range []string{root, filepath.Join(root, "bin")} {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if strings.HasSuffix(e.Name(), oldSuffix) || strings.HasPrefix(e.Name(), ".update-") {
				os.RemoveAll(filepath.Join(dir, e.Name()))
			}
		}
	}
}

func Stage(root string) (string, error) {
	dir, err := os.MkdirTemp(root, ".update-")
	if err != nil {
		if errors.Is(err, os.ErrPermission) {
			return "", fmt.Errorf("no permission to write in %s: run the update with sudo", root)
		}
		return "", err
	}
	return dir, nil
}

func Install(root, staged string) error {
	entries, err := os.ReadDir(staged)
	if err != nil {
		return err
	}
	var undo []func()
	rollback := func() {
		for i := len(undo) - 1; i >= 0; i-- {
			undo[i]()
		}
	}
	replace := func(target, fresh string) error {
		old := target + oldSuffix
		os.RemoveAll(old)
		moved := false
		if _, err := os.Lstat(target); err == nil {
			if err := os.Rename(target, old); err != nil {
				return err
			}
			moved = true
		}
		if err := os.Rename(fresh, target); err != nil {
			if moved {
				os.Rename(old, target)
			}
			return err
		}
		undo = append(undo, func() {
			os.Rename(target, fresh)
			if moved {
				os.Rename(old, target)
			}
		})
		return nil
	}

	for _, e := range entries {
		target := filepath.Join(root, e.Name())
		fresh := filepath.Join(staged, e.Name())
		if e.Name() == "bin" && e.IsDir() {
			files, err := os.ReadDir(fresh)
			if err != nil {
				rollback()
				return err
			}
			os.MkdirAll(target, 0o755)
			for _, f := range files {
				if err := replace(filepath.Join(target, f.Name()), filepath.Join(fresh, f.Name())); err != nil {
					rollback()
					return err
				}
			}
			continue
		}
		if err := replace(target, fresh); err != nil {
			rollback()
			return err
		}
	}
	Cleanup(root)
	return nil
}
