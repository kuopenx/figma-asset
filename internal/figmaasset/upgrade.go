package figmaasset

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	GitHubOwner = "kuopenx"
	GitHubRepo  = "figma-asset"
)

// --- version check ---

type githubRelease struct {
	TagName string `json:"tag_name"`
}

// FetchLatestVersion calls GitHub API and returns the latest release tag.
func FetchLatestVersion() (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", GitHubOwner, GitHubRepo)

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API returned %s", resp.Status)
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", err
	}
	return release.TagName, nil
}

// IsUpdateAvailable returns true if latest differs from current.
func IsUpdateAvailable(current, latest string) bool {
	return current != latest
}

// --- upgrade ---

// RunUpgrade downloads, verifies, and installs the latest release.
func RunUpgrade(currentVersion string) error {
	latest, err := FetchLatestVersion()
	if err != nil {
		return err
	}
	if !IsUpdateAvailable(currentVersion, latest) {
		fmt.Println("Already up to date.")
		return nil
	}

	fmt.Printf("Upgrading %s -> %s\n", currentVersion, latest)

	zipName := platformZipName()
	zipURL := fmt.Sprintf("https://github.com/%s/%s/releases/latest/download/%s", GitHubOwner, GitHubRepo, zipName)

	// 1. Download zip.
	fmt.Print("Downloading... ")
	zipPath, err := downloadFile(zipURL)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	defer os.Remove(zipPath)
	fmt.Println("done.")

	// 2. Verify checksum.
	fmt.Print("Verifying checksum... ")
	if err := verifyChecksum(zipPath, zipName); err != nil {
		return err
	}
	fmt.Println("done.")

	// 3. Stop daemon before replacing binary.
	_, _ = shutdownDaemonIfRunning()

	// 4. Extract and replace.
	fmt.Print("Installing... ")
	if err := extractAndReplace(zipPath); err != nil {
		return err
	}
	fmt.Println("done.")

	// 5. Start new daemon.
	if err := ensureDaemon(); err != nil {
		fmt.Printf("Binary updated. Start daemon manually: figma-asset start\n")
		return nil
	}

	fmt.Printf("Upgraded to %s.\n", latest)
	return nil
}

// --- helpers ---

func platformZipName() string {
	return fmt.Sprintf("figma-asset-%s-%s.zip", runtime.GOOS, runtime.GOARCH)
}

func downloadFile(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %s", resp.Status)
	}

	tmp, err := os.CreateTemp("", "figma-asset-*.zip")
	if err != nil {
		return "", err
	}

	if _, err := io.Copy(tmp, resp.Body); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return "", err
	}
	tmp.Close()
	return tmp.Name(), nil
}

func verifyChecksum(zipPath, zipName string) error {
	checksumURL := fmt.Sprintf("https://github.com/%s/%s/releases/latest/download/checksums.txt", GitHubOwner, GitHubRepo)
	resp, err := http.Get(checksumURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("checksums.txt not found (HTTP %s)", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	expectedHash := ""
	for _, line := range strings.Split(string(body), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[1] == zipName {
			expectedHash = fields[0]
			break
		}
	}
	if expectedHash == "" {
		return fmt.Errorf("no checksum found for %s", zipName)
	}

	data, err := os.ReadFile(zipPath)
	if err != nil {
		return err
	}
	sum := sha256.Sum256(data)
	actualHex := hex.EncodeToString(sum[:])

	if actualHex != expectedHash {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expectedHash, actualHex)
	}
	return nil
}

func extractAndReplace(zipPath string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()

	// Resolve current binary path (follow symlinks).
	exePath, err := os.Executable()
	if err != nil {
		return err
	}
	exePath, err = filepath.EvalSymlinks(exePath)
	if err != nil {
		return err
	}
	binDir := filepath.Dir(exePath) // ~/.local/bin/
	pDir := pluginDir()             // ~/figma-asset-plugin/

	// Extract to temp dir.
	tmpDir, err := os.MkdirTemp("", "figma-asset-extract-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	for _, f := range r.File {
		if f.FileInfo().IsDir() {
			continue
		}
		destPath := filepath.Join(tmpDir, f.Name)
		if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		dest, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			rc.Close()
			return err
		}
		if _, err := io.Copy(dest, rc); err != nil {
			rc.Close()
			dest.Close()
			return err
		}
		rc.Close()
		dest.Close()
	}

	// Replace binary atomically (write temp in same dir, then rename).
	binaryName := "figma-asset"
	if runtime.GOOS == "windows" {
		binaryName = "figma-asset.exe"
	}
	newBinary := filepath.Join(tmpDir, binaryName)
	if _, err := os.Stat(newBinary); err != nil {
		return fmt.Errorf("binary not found in archive")
	}

	tmpBinary := filepath.Join(binDir, "."+binaryName+".new")
	if err := copyFile(newBinary, tmpBinary); err != nil {
		return err
	}
	if err := os.Chmod(tmpBinary, 0o755); err != nil {
		os.Remove(tmpBinary)
		return err
	}
	if err := os.Rename(tmpBinary, exePath); err != nil {
		os.Remove(tmpBinary)
		return err
	}

	// Replace manifest.json if present in archive.
	manifestSrc := filepath.Join(tmpDir, "manifest.json")
	if _, err := os.Stat(manifestSrc); err == nil {
		if err := os.MkdirAll(pDir, 0o755); err != nil {
			return err
		}
		if err := copyFile(manifestSrc, filepath.Join(pDir, "manifest.json")); err != nil {
			return err
		}
	}

	// Replace plugin files.
	pluginSrc := filepath.Join(tmpDir, "plugin")
	pluginDst := filepath.Join(pDir, "plugin")
	if _, err := os.Stat(pluginSrc); err == nil {
		if err := os.MkdirAll(pluginDst, 0o755); err != nil {
			return err
		}
		entries, err := os.ReadDir(pluginSrc)
		if err != nil {
			return err
		}
		for _, entry := range entries {
			src := filepath.Join(pluginSrc, entry.Name())
			dst := filepath.Join(pluginDst, entry.Name())
			if err := copyFile(src, dst); err != nil {
				return err
			}
		}
	}

	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}
