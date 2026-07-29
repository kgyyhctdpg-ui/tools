package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunRendersMarkdownFile(t *testing.T) {
	tempDir := t.TempDir()
	inputPath := filepath.Join(tempDir, "report.md")
	outputPath := filepath.Join(tempDir, "report.docx")

	markdown := []byte(`# 第一章

正文内容。

1. 一级列表
   1. 二级列表

行内公式 $a^2+b^2=c^2$
`)
	if err := os.WriteFile(inputPath, markdown, 0o600); err != nil {
		t.Fatalf("write markdown fixture: %v", err)
	}

	if err := run([]string{
		"-input", inputPath,
		"-output", outputPath,
		"-heading-style", "default",
	}); err != nil {
		t.Fatalf("run cli: %v", err)
	}

	info, err := os.Stat(outputPath)
	if err != nil {
		t.Fatalf("stat output docx: %v", err)
	}
	if info.Size() == 0 {
		t.Fatal("expected output docx to be non-empty")
	}
}

func TestRunRequiresOutputForStdin(t *testing.T) {
	if err := run([]string{"-input", "-"}); err == nil {
		t.Fatal("expected stdin input without output path to fail")
	}
}

func TestRunWithoutArgsPrintsHelp(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	if err := runWithIO(nil, strings.NewReader(""), &stdout, &stderr); err != nil {
		t.Fatalf("run cli without args: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "Usage: docx") || !strings.Contains(output, "-input") {
		t.Fatalf("expected help output, got: %s", output)
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected empty stderr, got: %s", stderr.String())
	}
}
