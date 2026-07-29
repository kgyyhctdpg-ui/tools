package common

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/scoming-dev/tools/filex"
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

	tempFile, err := filex.Download(context.Background(), filepathOrURL)
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
