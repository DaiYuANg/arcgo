package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func safeJoinPath(base, name string) (string, error) {
	base = filepath.Clean(base)
	path := filepath.Clean(filepath.Join(base, name))
	rel, err := filepath.Rel(base, path)
	if err != nil {
		return "", fmt.Errorf("resolve relative path for %s: %w", name, err)
	}
	if strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("path traversal not allowed: %s", name)
	}
	return path, nil
}

// createVersionedDirs 创建版本文档目录
func createVersionedDirs(contentDir string, versions []Version) error {
	versionedDir := filepath.Join(contentDir, "versioned")
	sourceDocsDir := filepath.Join(contentDir, "docs")
	sourceRootFiles := rootFiles(contentDir)

	for _, version := range versions {
		if version.Current {
			continue
		}
		if err := createVersionDir(versionedDir, sourceDocsDir, sourceRootFiles, version); err != nil {
			return err
		}
	}

	return nil
}

func rootFiles(contentDir string) []string {
	return []string{
		filepath.Join(contentDir, "_index.md"),
		filepath.Join(contentDir, "_index.en.md"),
		filepath.Join(contentDir, "_index.zh.md"),
	}
}

func createVersionDir(versionedDir, sourceDocsDir string, sourceRootFiles []string, version Version) error {
	versionDir := filepath.Join(versionedDir, version.Name)
	versionDocsDir := filepath.Join(versionDir, "docs")

	if _, err := os.Stat(versionDir); err == nil {
		stdoutf("   ⏭️  跳过已存在的版本目录：%s\n", version.Name)
		return nil
	}

	stdoutf("   📁 创建版本文档目录：%s\n", version.Name)
	if err := os.MkdirAll(versionDocsDir, 0o750); err != nil {
		return fmt.Errorf("无法创建目录 %s：%w", versionDir, err)
	}

	if err := copyDir(sourceDocsDir, versionDocsDir); err != nil {
		stdoutf("      ⚠️  复制 docs 目录失败：%v\n", err)
	}

	copyRootFiles(versionDir, sourceRootFiles)
	return nil
}

func copyRootFiles(versionDir string, sourceRootFiles []string) {
	for _, srcFile := range sourceRootFiles {
		if _, err := os.Stat(srcFile); os.IsNotExist(err) {
			continue
		}
		dstFile, err := safeJoinPath(versionDir, filepath.Base(srcFile))
		if err != nil {
			stdoutf("      ⚠️  解析目标文件 %s 失败：%v\n", filepath.Base(srcFile), err)
			continue
		}
		if err := copyFile(srcFile, dstFile); err != nil {
			stdoutf("      ⚠️  复制文件 %s 失败：%v\n", filepath.Base(srcFile), err)
		}
	}
}

// copyDir 递归复制目录
func copyDir(srcDir, dstDir string) error {
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return fmt.Errorf("读取目录 %s 失败：%w", srcDir, err)
	}

	for _, entry := range entries {
		if isSkippedVersionEntry(entry) {
			continue
		}

		if err := copyDirEntry(srcDir, dstDir, entry); err != nil {
			return err
		}
	}

	return nil
}

func isSkippedVersionEntry(entry os.DirEntry) bool {
	return entry.Name() == "versioned"
}

func copyDirEntry(srcDir, dstDir string, entry os.DirEntry) error {
	srcPath, err := safeJoinPath(srcDir, entry.Name())
	if err != nil {
		return fmt.Errorf("invalid source path %s: %w", entry.Name(), err)
	}
	dstPath, err := safeJoinPath(dstDir, entry.Name())
	if err != nil {
		return fmt.Errorf("invalid destination path %s: %w", entry.Name(), err)
	}

	if entry.IsDir() {
		if err := os.MkdirAll(dstPath, 0o750); err != nil {
			return fmt.Errorf("创建目录 %s 失败：%w", dstPath, err)
		}
		return copyDir(srcPath, dstPath)
	}

	return copyFile(srcPath, dstPath)
}

// copyFile 复制文件
func copyFile(srcPath, dstPath string) (retErr error) {
	//nolint:gosec // srcPath is validated by safeJoinPath before copyFile is called.
	src, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("读取文件 %s 失败：%w", srcPath, err)
	}
	defer func() {
		retErr = errors.Join(retErr, closeFile(src, "源文件", srcPath))
	}()

	//nolint:gosec // dstPath is validated by safeJoinPath before copyFile is called.
	dst, err := os.OpenFile(dstPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("写入文件 %s 失败：%w", dstPath, err)
	}
	defer func() {
		retErr = errors.Join(retErr, closeFile(dst, "目标文件", dstPath))
	}()

	if _, err := io.Copy(dst, src); err != nil {
		return fmt.Errorf("复制文件 %s 到 %s 失败：%w", srcPath, dstPath, err)
	}

	return nil
}

func closeFile(file io.Closer, kind, path string) error {
	if err := file.Close(); err != nil {
		return fmt.Errorf("关闭%s %s 失败：%w", kind, path, err)
	}

	return nil
}
