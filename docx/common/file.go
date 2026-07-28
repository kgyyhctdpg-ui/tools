package common

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// PrepareFilepath returns a local file path, downloading and caching HTTP(S) URLs when needed.
func PrepareFilepath(filepathOrURL string) (string, bool, error) {
	u, err := url.Parse(filepathOrURL)
	if err != nil {
		return "", false, fmt.Errorf("无效的URL: %w", err)
	}

	if !strings.HasPrefix(u.Scheme, "http") {
		if _, err := os.Stat(filepathOrURL); err != nil {
			return "", false, fmt.Errorf("本地文件不存在: %w", err)
		}
		return filepathOrURL, false, nil
	}

	hash := md5.Sum([]byte(filepathOrURL))
	hashStr := hex.EncodeToString(hash[:])

	filename := filepath.Base(u.Path)
	if filename == "" || filename == "/" || filename == "." {
		filename = "downloaded.xlsx"
	}

	finalFilename := fmt.Sprintf("%s_%s", hashStr, filename)
	finalPath := filepath.Join(os.TempDir(), finalFilename)

	if _, err := os.Stat(finalPath); err == nil {
		log.Printf("使用缓存文件: %s", finalPath)
		return finalPath, false, nil
	}

	tempFile, err := DownloadFile(filepathOrURL)
	if err != nil {
		return "", false, fmt.Errorf("下载文件失败: %w", err)
	}

	if err := os.Rename(tempFile, finalPath); err != nil {
		if _, statErr := os.Stat(finalPath); statErr == nil {
			_ = os.Remove(tempFile)
			log.Printf("并发下载，使用已存在的文件: %s", finalPath)
			return finalPath, true, nil
		}
		_ = os.Remove(tempFile)
		return "", false, fmt.Errorf("重命名临时文件失败: %w", err)
	}

	return finalPath, true, nil
}

// DownloadFile downloads a URL into a temporary xlsx file and returns its path.
func DownloadFile(urlStr string) (string, error) {
	client := http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequest(http.MethodGet, urlStr, nil)
	if err != nil {
		return "", fmt.Errorf("创建请求失败: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("下载文件失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("下载文件失败，状态码: %d", resp.StatusCode)
	}

	tempFile, err := os.CreateTemp("", "excel-download-*.xlsx")
	if err != nil {
		return "", fmt.Errorf("创建临时文件失败: %w", err)
	}

	_, err = io.Copy(tempFile, resp.Body)
	if err != nil {
		_ = tempFile.Close()
		_ = os.Remove(tempFile.Name())
		return "", fmt.Errorf("写入临时文件失败: %w", err)
	}

	if err := tempFile.Close(); err != nil {
		_ = os.Remove(tempFile.Name())
		return "", fmt.Errorf("关闭临时文件失败: %w", err)
	}

	return tempFile.Name(), nil
}
