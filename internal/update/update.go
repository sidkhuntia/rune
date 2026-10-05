// Package update implements rune's self-update: find the latest GitHub
// release, verify it against the published checksums, and swap the binary.
package update

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	// APIBase is the GitHub API root for the rune repository.
	APIBase = "https://api.github.com/repos/sidkhuntia/rune"

	checksumsAsset = "checksums.txt"
	binaryName     = "rune"
	maxDownload    = 100 << 20
)

// Release is the subset of a GitHub release the updater needs.
type Release struct {
	Tag    string
	assets map[string]string // asset name -> download URL
}

// Client talks to the GitHub releases API.
type Client struct {
	APIBase string
	HTTP    *http.Client
}

// NewClient returns a Client for the real GitHub API.
func NewClient() *Client {
	return &Client{APIBase: APIBase, HTTP: &http.Client{Timeout: 2 * time.Minute}}
}

// Latest fetches the most recent published release.
func (c *Client) Latest(ctx context.Context) (*Release, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.APIBase+"/releases/latest", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to query latest release: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("latest release lookup failed with status %d", resp.StatusCode)
	}

	var payload struct {
		Tag    string `json:"tag_name"`
		Assets []struct {
			Name string `json:"name"`
			URL  string `json:"browser_download_url"`
		} `json:"assets"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(&payload); err != nil {
		return nil, fmt.Errorf("failed to parse release: %w", err)
	}

	rel := &Release{Tag: payload.Tag, assets: map[string]string{}}
	for _, a := range payload.Assets {
		rel.assets[a.Name] = a.URL
	}
	return rel, nil
}

// AssetName is the release archive name for a platform (matches .goreleaser.yml).
func AssetName(goos, goarch string) string {
	return fmt.Sprintf("%s-%s-%s.tar.gz", binaryName, goos, goarch)
}

// IsNewer reports whether latest is a higher semantic version than current.
// Unparseable versions (e.g. "dev") are never considered up to date.
func IsNewer(current, latest string) bool {
	c, okC := parseVersion(current)
	l, okL := parseVersion(latest)
	if !okL {
		return false
	}
	if !okC {
		return true
	}
	for i := range c {
		if l[i] != c[i] {
			return l[i] > c[i]
		}
	}
	return false
}

func parseVersion(v string) ([3]int, bool) {
	var out [3]int
	v = strings.TrimPrefix(strings.TrimSpace(v), "v")
	if i := strings.IndexAny(v, "-+"); i >= 0 {
		v = v[:i]
	}
	parts := strings.Split(v, ".")
	if len(parts) != 3 {
		return out, false
	}
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return out, false
		}
		out[i] = n
	}
	return out, true
}

// Apply downloads the release archive for the platform, verifies its SHA-256
// against checksums.txt, and atomically replaces the executable at exePath.
func (c *Client) Apply(ctx context.Context, rel *Release, exePath, goos, goarch string) error {
	name := AssetName(goos, goarch)
	archiveURL, ok := rel.assets[name]
	if !ok {
		return fmt.Errorf("release %s has no build for %s/%s", rel.Tag, goos, goarch)
	}
	sumsURL, ok := rel.assets[checksumsAsset]
	if !ok {
		return fmt.Errorf("release %s has no %s; refusing to install unverified binary", rel.Tag, checksumsAsset)
	}

	sums, err := c.download(ctx, sumsURL)
	if err != nil {
		return fmt.Errorf("failed to download checksums: %w", err)
	}
	want, err := checksumFor(string(sums), name)
	if err != nil {
		return err
	}

	archive, err := c.download(ctx, archiveURL)
	if err != nil {
		return fmt.Errorf("failed to download %s: %w", name, err)
	}
	if got := sha256.Sum256(archive); hex.EncodeToString(got[:]) != want {
		return fmt.Errorf("checksum mismatch for %s; aborting", name)
	}

	bin, err := extractBinary(archive)
	if err != nil {
		return err
	}
	return replaceFile(exePath, bin)
}

func (c *Client) download(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}
	return io.ReadAll(io.LimitReader(resp.Body, maxDownload))
}

// checksumFor finds a file's hash in `sha256sum`-style output.
func checksumFor(sums, name string) (string, error) {
	for _, line := range strings.Split(sums, "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[1] == name {
			return strings.ToLower(fields[0]), nil
		}
	}
	return "", fmt.Errorf("no checksum listed for %s", name)
}

// extractBinary returns the rune executable from a .tar.gz archive.
func extractBinary(archive []byte) ([]byte, error) {
	gz, err := gzip.NewReader(bytes.NewReader(archive))
	if err != nil {
		return nil, fmt.Errorf("failed to open archive: %w", err)
	}
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return nil, fmt.Errorf("archive does not contain %q", binaryName)
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read archive: %w", err)
		}
		if hdr.Typeflag == tar.TypeReg && filepath.Base(hdr.Name) == binaryName {
			return io.ReadAll(io.LimitReader(tr, maxDownload))
		}
	}
}

// replaceFile swaps exePath for data via a temp file in the same directory,
// so the rename is atomic and a running binary is never half-written.
func replaceFile(exePath string, data []byte) error {
	dir := filepath.Dir(exePath)
	tmp, err := os.CreateTemp(dir, ".rune-update-*")
	if err != nil {
		return fmt.Errorf("cannot write to %s (try sudo, or reinstall via your package manager): %w", dir, err)
	}
	tmpPath := tmp.Name()
	defer func() { _ = os.Remove(tmpPath) }() // no-op after a successful rename

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("failed to write update: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("failed to write update: %w", err)
	}
	if err := os.Chmod(tmpPath, 0o755); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, exePath); err != nil {
		return fmt.Errorf("failed to replace %s: %w", exePath, err)
	}
	return nil
}

// IsHomebrewPath reports whether the executable lives in a Homebrew Cellar,
// where in-place replacement would desync brew's records.
func IsHomebrewPath(exePath string) bool {
	return strings.Contains(filepath.ToSlash(exePath), "/Cellar/")
}
