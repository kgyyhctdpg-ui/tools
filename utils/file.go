package utils

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"
)

// 是否目录
func IsDir(path string) bool {
	f, e := os.Stat(path)
	if e != nil {
		return false
	}
	return f.IsDir()
}

// 是否文件
func IsFile(path string) bool {
	f, e := os.Stat(path)
	if e != nil {
		return false
	}
	if f.IsDir() {
		return false
	}
	return true
}

// 文件大小
func FileSize(size int) string {
	s := float32(size)
	if s > 1024*1024 {
		return fmt.Sprintf("%.1f M", s/(1024*1024))
	}
	if s > 1024 {
		return fmt.Sprintf("%.1f K", s/1024)
	}
	return fmt.Sprintf("%f B", s)
}

// 图片转base64
func ImgToBase64(img string, quality int) (b string, err error) {
	bg, err := os.Open(img)
	if err != nil {
		return "", err
	}
	defer bg.Close()
	imgstr, _, err := image.Decode(bg)
	if err != nil {
		return "", err
	}

	var byteImg bytes.Buffer
	imgw := bufio.NewWriter(&byteImg)
	jpeg.Encode(imgw, imgstr, &jpeg.Options{Quality: quality})

	base_64 := base64.StdEncoding.EncodeToString(byteImg.Bytes())

	return string(base_64), nil
}

// prepareFilepath 准备文件路径，如果是远程URL则下载到临时文件
func PrepareFilepath(filepathOrURL string) (string, bool, error) {
	// 解析URL
	u, err := url.Parse(filepathOrURL)
	if err != nil {
		return "", false, fmt.Errorf("无效的URL: %w", err)
	}

	// 下载文件到临时文件
	tempFile, err := DownloadFile(filepathOrURL)
	if err != nil {
		return "", false, fmt.Errorf("下载文件失败: %w", err)
	}

	// 从URL中提取文件名，如果没有则使用默认名称
	filename := filepath.Base(u.Path)

	// 创建带扩展名的临时文件
	tempDir := filepath.Dir(tempFile)
	finalTempFile := filepath.Join(tempDir, filename)

	// 如果临时文件名和最终文件名不同，重命名
	if tempFile != finalTempFile {
		if err := os.Rename(tempFile, finalTempFile); err != nil {
			os.Remove(tempFile) // 清理原临时文件
			return "", false, fmt.Errorf("重命名临时文件失败: %w", err)
		}
		tempFile = finalTempFile
	}

	return tempFile, true, nil
}

// downloadFile 从URL下载文件到临时文件
func DownloadFile(url string) (string, error) {
	client := http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
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

	ext := filepath.Ext(url)
	if ext == "" {
		return "", fmt.Errorf("URL 文件格式有误")
	}

	// 创建临时文件
	tempFile, err := os.CreateTemp("", fmt.Sprintf("download-temp-%s", ext))
	if err != nil {
		return "", fmt.Errorf("创建临时文件失败: %w", err)
	}

	// 写入文件
	_, err = io.Copy(tempFile, resp.Body)
	if err != nil {
		tempFile.Close()
		os.Remove(tempFile.Name())
		return "", fmt.Errorf("写入临时文件失败: %w", err)
	}

	if err := tempFile.Close(); err != nil {
		os.Remove(tempFile.Name())
		return "", fmt.Errorf("关闭临时文件失败: %w", err)
	}

	return tempFile.Name(), nil
}
