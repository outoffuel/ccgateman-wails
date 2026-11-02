package updater

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	defaultOwner = "outoffuel"
	defaultRepo  = "ccgateman-wails"
)

// ReleaseInfo 最新リリース情報
type ReleaseInfo struct {
	Version      string `json:"version"`
	CurrentVer   string `json:"currentVersion"`
	HasUpdate    bool   `json:"hasUpdate"`
	ReleaseURL   string `json:"releaseUrl"`
	ReleaseName  string `json:"releaseName"`
	PublishedAt  string `json:"publishedAt"`
	AssetURL     string `json:"assetUrl,omitempty"`
	AssetName    string `json:"assetName,omitempty"`
}

type githubRelease struct {
	TagName     string `json:"tag_name"`
	Name        string `json:"name"`
	HTMLURL     string `json:"html_url"`
	PublishedAt string `json:"published_at"`
	Assets      []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
		Size               int64  `json:"size"`
	} `json:"assets"`
}

// Updater 自動更新管理
type Updater struct {
	CurrentVersion string
	Owner          string
	Repo           string
	httpClient     *http.Client
}

// NewUpdater インスタンス作成
func NewUpdater(currentVersion string) *Updater {
	return &Updater{
		CurrentVersion: currentVersion,
		Owner:          defaultOwner,
		Repo:           defaultRepo,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// CleanupOldExecutable 起動時に前回の更新残骸 (.old) を削除
func CleanupOldExecutable() {
	execPath, err := os.Executable()
	if err != nil {
		return
	}
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return
	}

	oldPath := execPath + ".old"
	if _, err := os.Stat(oldPath); err == nil {
		if removeErr := os.Remove(oldPath); removeErr != nil {
			log.Printf("[Updater] Warning: Failed to remove old executable %s: %v", oldPath, removeErr)
		} else {
			log.Printf("[Updater] Cleaned up previous executable: %s", oldPath)
		}
	}
}

// CheckForUpdate GitHub Releases から最新版を確認
func (u *Updater) CheckForUpdate(ctx context.Context) (*ReleaseInfo, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/tags/latest", u.Owner, u.Repo)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create update request: %w", err)
	}
	req.Header.Set("User-Agent", "ccgateman-updater")
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := u.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to query GitHub releases: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub release API returned status %d", resp.StatusCode)
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("failed to parse release json: %w", err)
	}

	// バージョン判定 (release.Name に vYYYY.MM.DD.x が設定される)
	targetVersion := strings.TrimSpace(release.Name)
	if targetVersion == "" || strings.HasPrefix(targetVersion, "Latest ") {
		targetVersion = strings.TrimSpace(release.TagName)
	}

	expectedAssetName := u.getExpectedAssetName()
	var matchedAssetURL string
	for _, asset := range release.Assets {
		if asset.Name == expectedAssetName {
			matchedAssetURL = asset.BrowserDownloadURL
			break
		}
	}

	current := strings.TrimSpace(u.CurrentVersion)
	hasUpdate := false
	if targetVersion != "" && targetVersion != "latest" {
		if current == "dev" {
			// 開発環境でも手動確認時は最新リリースがあれば検知できるようにする
			hasUpdate = true
		} else if current != targetVersion {
			hasUpdate = true
		}
	}

	info := &ReleaseInfo{
		Version:      targetVersion,
		CurrentVer:   current,
		HasUpdate:    hasUpdate,
		ReleaseURL:   release.HTMLURL,
		ReleaseName:  release.Name,
		PublishedAt:  release.PublishedAt,
		AssetURL:     matchedAssetURL,
		AssetName:    expectedAssetName,
	}

	return info, nil
}

// ApplyUpdate 指定されたリリースの更新ファイルをダウンロードして適用、再起動する
func (u *Updater) ApplyUpdate(ctx context.Context, info *ReleaseInfo) error {
	if info == nil || info.AssetURL == "" {
		return fmt.Errorf("valid asset URL not found for current OS (%s)", runtime.GOOS)
	}

	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to determine executable path: %w", err)
	}
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return fmt.Errorf("failed to resolve symlinks for executable: %w", err)
	}

	execDir := filepath.Dir(execPath)

	// 1. 一時作業ディレクトリを作成
	tempDir, err := os.MkdirTemp(execDir, "update-tmp-*")
	if err != nil {
		tempDir, err = os.MkdirTemp("", "osuccgate-update-*")
		if err != nil {
			return fmt.Errorf("failed to create temporary directory: %w", err)
		}
	}
	defer os.RemoveAll(tempDir)

	downloadPath := filepath.Join(tempDir, info.AssetName)
	log.Printf("[Updater] Downloading update from %s to %s...", info.AssetURL, downloadPath)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, info.AssetURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create download request: %w", err)
	}
	req.Header.Set("User-Agent", "ccgateman-updater")

	resp, err := u.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download returned HTTP status %d", resp.StatusCode)
	}

	dlFile, err := os.OpenFile(downloadPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return fmt.Errorf("failed to create download destination file: %w", err)
	}

	_, err = io.Copy(dlFile, resp.Body)
	dlFile.Close()
	if err != nil {
		return fmt.Errorf("failed to save downloaded file: %w", err)
	}

	// 2. 新しい実行ファイルの抽出
	var newExecPath string
	if strings.HasSuffix(info.AssetName, ".tar.gz") {
		// Linux: tar.gz の展開
		extractedPath := filepath.Join(tempDir, "extracted_bin")
		if err := extractTarGzBinary(downloadPath, "osuccgate", extractedPath); err != nil {
			return fmt.Errorf("failed to extract binary from tar.gz: %w", err)
		}
		newExecPath = extractedPath
	} else {
		newExecPath = downloadPath
	}

	// 実行権限を付与
	if err := os.Chmod(newExecPath, 0755); err != nil {
		log.Printf("[Updater] Warning: Failed to chmod 0755 on %s: %v", newExecPath, err)
	}

	// 3. 現在の実行ファイルを .old にリネーム
	oldPath := execPath + ".old"
	_ = os.Remove(oldPath) // 既存の .old があれば除去
	if err := os.Rename(execPath, oldPath); err != nil {
		return fmt.Errorf("failed to rename current executable to .old: %w", err)
	}

	// 4. 新しい実行ファイルを本来のパスへ移動
	if err := moveOrCopyFile(newExecPath, execPath); err != nil {
		// 失敗した場合は旧ファイルを復元
		_ = os.Rename(oldPath, execPath)
		return fmt.Errorf("failed to place new executable into %s: %w", execPath, err)
	}
	_ = os.Chmod(execPath, 0755)

	log.Printf("[Updater] Update applied successfully. Restarting process...")

	// 5. 新プロセスの起動
	cmd := exec.Command(execPath, os.Args[1:]...)
	cmd.Dir = execDir
	cmd.Env = os.Environ()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Start(); err != nil {
		// 起動失敗時は復元を試みる
		_ = os.Remove(execPath)
		_ = os.Rename(oldPath, execPath)
		return fmt.Errorf("failed to start new version: %w", err)
	}

	// 6. 親プロセスを安全に終了
	go func() {
		time.Sleep(500 * time.Millisecond)
		os.Exit(0)
	}()

	return nil
}

func (u *Updater) getExpectedAssetName() string {
	if runtime.GOOS == "windows" {
		return "osuccgate-windows-amd64.exe"
	}
	return "osuccgate-linux-amd64.tar.gz"
}

// extractTarGzBinary tar.gz から指定名バイナリを取り出す
func extractTarGzBinary(tarGzPath, targetBinaryName, destPath string) error {
	f, err := os.Open(tarGzPath)
	if err != nil {
		return err
	}
	defer f.Close()

	gzReader, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		baseName := filepath.Base(header.Name)
		if baseName == targetBinaryName {
			outFile, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
			if err != nil {
				return err
			}
			if _, err := io.Copy(outFile, tarReader); err != nil {
				outFile.Close()
				return err
			}
			outFile.Close()
			return nil
		}
	}

	return fmt.Errorf("binary %q not found in archive", targetBinaryName)
}

// moveOrCopyFile ファイルを移動（異なるパーティション等で Rename が失敗する場合はコピーして削除）
func moveOrCopyFile(src, dst string) error {
	if err := os.Rename(src, dst); err == nil {
		return nil
	}

	// コピーで代替
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return err
	}

	_ = os.Remove(src)
	return nil
}

