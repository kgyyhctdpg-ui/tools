package oss

import (
	"fmt"
	"os"
)

func deleteLocalFile(filePath string) error {
	if err := os.Remove(filePath); err != nil {
		return fmt.Errorf("failed to delete local file %q: %w", filePath, err)
	}
	return nil
}
