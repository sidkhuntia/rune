package update

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
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

func TestIsNewer(t *testing.T) {
	for _, tt := range []struct {
		cur, latest string
		want        bool
	}{
		{"v2.0.1", "v2.0.2", true},
		{"2.0.1", "v2.1.0", true},
		{"v2.0.9", "v2.0.10", true}, // numeric, not lexical
		{"v2.0.1", "v2.0.1", false},
		{"v2.1.0", "v2.0.9", false},
		{"v3.0.0-rc1", "v3.0.0", false},
		{"dev", "v2.0.1", true},
		{"v2.0.1", "garbage", false},
	} {
		if got := IsNewer(tt.cur, tt.latest); got != tt.want {
			t.Errorf("IsNewer(%q, %q) = %v, want %v", tt.cur, tt.latest, got, tt.want)
		}
	}
}

func makeArchive(t *testing.T, binary string) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for name, body := range map[string]string{"LICENSE": "mit", "rune": binary} {
		if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0o755, Size: int64(len(body)), Typeflag: tar.TypeReg}); err != nil {
			t.Fatal(err)
		}
		_, _ = tw.Write([]byte(body))
	}
	_ = tw.Close()
	_ = gz.Close()
	return buf.Bytes()
}

// fakeRelease serves a release whose checksum for the archive is sumOverride
// when non-empty (to simulate tampering).
func fakeRelease(t *testing.T, archive []byte, sumOverride string) *Client {
	t.Helper()
	name := AssetName("linux", "amd64")
	sum := sha256.Sum256(archive)
	hash := hex.EncodeToString(sum[:])
	if sumOverride != "" {
		hash = sumOverride
	}

	mux := http.NewServeMux()
	var srv *httptest.Server
	mux.HandleFunc("/releases/latest", func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprintf(w, `{"tag_name":"v9.9.9","assets":[
			{"name":%q,"browser_download_url":"%s/dl/%s"},
			{"name":"checksums.txt","browser_download_url":"%s/dl/checksums.txt"}]}`, name, srv.URL, name, srv.URL)
	})
	mux.HandleFunc("/dl/"+name, func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write(archive) })
	mux.HandleFunc("/dl/checksums.txt", func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprintf(w, "%s  %s\n%s  other.tar.gz\n", hash, name, strings.Repeat("0", 64))
	})
	srv = httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return &Client{APIBase: srv.URL, HTTP: srv.Client()}
}

func TestApplyReplacesBinary(t *testing.T) {
	c := fakeRelease(t, makeArchive(t, "NEW-BINARY"), "")
	rel, err := c.Latest(context.Background())
	if err != nil || rel.Tag != "v9.9.9" {
		t.Fatalf("Latest = %+v, %v", rel, err)
	}

	exe := filepath.Join(t.TempDir(), "rune")
	if err := os.WriteFile(exe, []byte("OLD"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := c.Apply(context.Background(), rel, exe, "linux", "amd64"); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(exe)
	if string(got) != "NEW-BINARY" {
		t.Errorf("binary = %q", got)
	}
	if info, _ := os.Stat(exe); info.Mode().Perm()&0o100 == 0 {
		t.Error("updated binary is not executable")
	}
	if entries, _ := os.ReadDir(filepath.Dir(exe)); len(entries) != 1 {
		t.Errorf("temp files left behind: %v", entries)
	}
}

func TestApplyRejectsChecksumMismatch(t *testing.T) {
	c := fakeRelease(t, makeArchive(t, "EVIL"), strings.Repeat("a", 64))
	rel, _ := c.Latest(context.Background())

	exe := filepath.Join(t.TempDir(), "rune")
	_ = os.WriteFile(exe, []byte("OLD"), 0o755)
	err := c.Apply(context.Background(), rel, exe, "linux", "amd64")
	if err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("err = %v", err)
	}
	if got, _ := os.ReadFile(exe); string(got) != "OLD" {
		t.Error("binary was modified despite checksum failure")
	}
}

func TestApplyUnsupportedPlatform(t *testing.T) {
	c := fakeRelease(t, makeArchive(t, "x"), "")
	rel, _ := c.Latest(context.Background())
	if err := c.Apply(context.Background(), rel, filepath.Join(t.TempDir(), "rune"), "windows", "arm64"); err == nil {
		t.Fatal("expected error for missing platform build")
	}
}

func TestIsHomebrewPath(t *testing.T) {
	if !IsHomebrewPath("/opt/homebrew/Cellar/rune/2.0.1/bin/rune") || IsHomebrewPath("/usr/local/bin/rune") {
		t.Error("homebrew detection wrong")
	}
}
