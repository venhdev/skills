package updater

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// ReleaseAsset holds download information for a GitHub Release artifact.
type ReleaseAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

// Release holds release metadata from the GitHub Releases API.
type Release struct {
	TagName string         `json:"tag_name"`
	Assets  []ReleaseAsset `json:"assets"`
}

var httpClient = &http.Client{
	Timeout: 60 * time.Second,
}

// GetLatestRelease fetches metadata for the newest release from GitHub.
func GetLatestRelease(repo string) (*Release, error) {
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", repo)
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", "zenf-updater")
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to query GitHub API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		var rel Release
		if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
			return nil, fmt.Errorf("failed to decode release metadata: %w", err)
		}
		return &rel, nil
	}

	// Fallback when GitHub API rate-limited: resolve web redirect to find tag
	webURL := fmt.Sprintf("https://github.com/%s/releases/latest", repo)
	noFollowClient := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Timeout: 15 * time.Second,
	}
	redirectResp, err := noFollowClient.Get(webURL)
	if err != nil {
		return nil, fmt.Errorf("failed to probe latest release via web redirect: %w", err)
	}
	defer redirectResp.Body.Close()

	loc := redirectResp.Header.Get("Location")
	if loc == "" {
		return nil, fmt.Errorf("github api returned status %d and redirect location is empty", resp.StatusCode)
	}

	// Location is like https://github.com/{repo}/releases/tag/vX.Y.Z
	parts := strings.Split(loc, "/")
	tag := parts[len(parts)-1]
	if tag == "" {
		return nil, fmt.Errorf("unable to parse tag from redirect URL: %s", loc)
	}

	return ConstructReleaseFromTag(repo, tag), nil
}

// ConstructReleaseFromTag synthesizes asset URLs for a known release tag based on GoReleaser naming.
func ConstructReleaseFromTag(repo, tag string) *Release {
	cleanVer := strings.TrimPrefix(tag, "v")
	osName := runtime.GOOS
	archName := runtime.GOARCH

	ext := "tar.gz"
	if osName == "windows" {
		ext = "zip"
	}

	assetName := fmt.Sprintf("zenf_%s_%s_%s.%s", cleanVer, osName, archName, ext)
	baseURL := fmt.Sprintf("https://github.com/%s/releases/download/%s", repo, tag)

	return &Release{
		TagName: tag,
		Assets: []ReleaseAsset{
			{
				Name:               assetName,
				BrowserDownloadURL: fmt.Sprintf("%s/%s", baseURL, assetName),
			},
			{
				Name:               "checksums.txt",
				BrowserDownloadURL: fmt.Sprintf("%s/checksums.txt", baseURL),
			},
		},
	}
}

// TargetAssetName determines the expected release asset name for current OS and architecture.
func TargetAssetName(tag string) string {
	cleanVer := strings.TrimPrefix(tag, "v")
	ext := "tar.gz"
	if runtime.GOOS == "windows" {
		ext = "zip"
	}
	return fmt.Sprintf("zenf_%s_%s_%s.%s", cleanVer, runtime.GOOS, runtime.GOARCH, ext)
}

// DownloadBytes downloads the content of a URL into memory.
func DownloadBytes(url string) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "zenf-updater")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download from %s returned HTTP %d", url, resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

// VerifyChecksum ensures the SHA-256 hash of data matches the entry in checksums.txt.
func VerifyChecksum(data []byte, checksumsTxt, expectedFilename string) error {
	sum := sha256.Sum256(data)
	actualHash := hex.EncodeToString(sum[:])

	lines := strings.Split(checksumsTxt, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			hash := strings.ToLower(fields[0])
			filename := fields[1]
			if filename == expectedFilename || strings.HasSuffix(filename, expectedFilename) {
				if hash == actualHash {
					return nil // Hash verified!
				}
				return fmt.Errorf("checksum mismatch for %s: expected %s, got %s", expectedFilename, hash, actualHash)
			}
		}
	}
	return fmt.Errorf("file %s not found in checksums.txt", expectedFilename)
}

// ExtractBinary extracts the zenf executable bytes from a tar.gz or zip archive.
func ExtractBinary(archiveData []byte, assetName string) ([]byte, error) {
	binaryName := "zenf"
	if runtime.GOOS == "windows" {
		binaryName = "zenf.exe"
	}

	if strings.HasSuffix(assetName, ".tar.gz") || strings.HasSuffix(assetName, ".tgz") {
		return extractFromTarGz(archiveData, binaryName)
	} else if strings.HasSuffix(assetName, ".zip") {
		return extractFromZip(archiveData, binaryName)
	}
	return nil, fmt.Errorf("unsupported archive format: %s", assetName)
}

func extractFromTarGz(data []byte, targetName string) ([]byte, error) {
	gzr, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed reading tar stream: %w", err)
		}

		if header.Typeflag == tar.TypeReg && (header.Name == targetName || strings.HasSuffix(header.Name, "/"+targetName)) {
			return io.ReadAll(tr)
		}
	}
	return nil, fmt.Errorf("binary %s not found in archive", targetName)
}

func extractFromZip(data []byte, targetName string) ([]byte, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("failed to open zip reader: %w", err)
	}

	for _, file := range zr.File {
		if file.Name == targetName || strings.HasSuffix(file.Name, "/"+targetName) {
			rc, err := file.Open()
			if err != nil {
				return nil, fmt.Errorf("failed to open file in zip: %w", err)
			}
			defer rc.Close()
			return io.ReadAll(rc)
		}
	}
	return nil, fmt.Errorf("binary %s not found in zip archive", targetName)
}

// IsNewer compares semantic versions (e.g. "v0.3.2" > "0.3.1").
func IsNewer(latestTag, currentVersion string) bool {
	vLatest := parseSemver(latestTag)
	vCurrent := parseSemver(currentVersion)

	for i := 0; i < 3; i++ {
		if vLatest[i] > vCurrent[i] {
			return true
		}
		if vLatest[i] < vCurrent[i] {
			return false
		}
	}
	return false
}

func parseSemver(v string) [3]int {
	clean := strings.TrimPrefix(strings.TrimSpace(v), "v")
	parts := strings.Split(clean, ".")
	var res [3]int
	for i := 0; i < len(parts) && i < 3; i++ {
		num, _ := strconv.Atoi(parts[i])
		res[i] = num
	}
	return res
}
