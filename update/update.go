package update

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const (
	githubRepo = "ReinforceZwei/qb-auto"
	assetName  = "qb-auto-linux-amd64"
)

// ErrUpToDate is returned by Do when the running version matches the latest release.
var ErrUpToDate = errors.New("already up to date")

type Release struct {
	TagName string  `json:"tag_name"`
	Assets  []Asset `json:"assets"`
}

type Asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

// FetchLatestRelease queries the GitHub releases API for the latest release.
func FetchLatestRelease() (*Release, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", githubRepo)
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("fetch release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github API returned HTTP %d", resp.StatusCode)
	}

	var release Release
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("decode release: %w", err)
	}
	return &release, nil
}

// NeedsSudo reports whether writing to the directory containing binaryPath
// requires elevated privileges. It probes by attempting to create a temp file
// in that directory.
func NeedsSudo(binaryPath string) bool {
	dir := filepath.Dir(binaryPath)
	f, err := os.CreateTemp(dir, ".qb-auto-write-check-*")
	if err != nil {
		return true
	}
	f.Close()
	os.Remove(f.Name())
	return false
}

// Do downloads and installs the latest release binary over execPath.
// It returns ErrUpToDate when currentVersion already matches the latest tag.
func Do(execPath, currentVersion string) error {
	release, err := FetchLatestRelease()
	if err != nil {
		return err
	}

	if release.TagName == currentVersion {
		return ErrUpToDate
	}

	binaryAsset, checksumAsset, err := findAssets(release)
	if err != nil {
		return err
	}

	dir := filepath.Dir(execPath)

	// Download binary to a temp file in the same directory so that os.Rename
	// is guaranteed to be an atomic same-filesystem move.
	tmp, err := os.CreateTemp(dir, ".qb-auto-update-*")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpName := tmp.Name()
	// Remove the temp file on any failure path.
	success := false
	defer func() {
		if !success {
			os.Remove(tmpName)
		}
	}()

	if err := downloadTo(tmp, binaryAsset.BrowserDownloadURL); err != nil {
		tmp.Close()
		return fmt.Errorf("download binary: %w", err)
	}
	tmp.Close()

	// Fetch and verify the SHA256 checksum.
	expectedHash, err := fetchChecksum(checksumAsset.BrowserDownloadURL)
	if err != nil {
		return fmt.Errorf("fetch checksum: %w", err)
	}
	if err := verifySHA256(tmpName, expectedHash); err != nil {
		return fmt.Errorf("checksum verification failed: %w", err)
	}

	if err := os.Chmod(tmpName, 0o755); err != nil {
		return fmt.Errorf("chmod: %w", err)
	}

	// Atomic replace. On Linux, rename unlinks the old inode so the running
	// process is unaffected; the next launch will use the new binary.
	if err := os.Rename(tmpName, execPath); err != nil {
		return fmt.Errorf("replace binary: %w", err)
	}

	success = true
	return nil
}

func findAssets(release *Release) (binary, checksum *Asset, err error) {
	checksumName := assetName + ".sha256"
	for i := range release.Assets {
		switch release.Assets[i].Name {
		case assetName:
			binary = &release.Assets[i]
		case checksumName:
			checksum = &release.Assets[i]
		}
	}
	if binary == nil {
		return nil, nil, fmt.Errorf("release %s has no asset named %q", release.TagName, assetName)
	}
	if checksum == nil {
		return nil, nil, fmt.Errorf("release %s has no asset named %q", release.TagName, checksumName)
	}
	return binary, checksum, nil
}

func downloadTo(dst io.Writer, url string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	_, err = io.Copy(dst, resp.Body)
	return err
}

func fetchChecksum(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(body)), nil
}

func verifySHA256(path, expected string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}

	got := hex.EncodeToString(h.Sum(nil))
	if got != strings.ToLower(expected) {
		return fmt.Errorf("got %s, want %s", got, expected)
	}
	return nil
}
