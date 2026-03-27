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

func copyDirContents(srcDir, dstDir string) error {
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return fmt.Errorf("read directory %s: %w", srcDir, err)
	}
	for _, entry := range entries {
		if err := copyDirEntry(srcDir, dstDir, entry); err != nil {
			return err
		}
	}
	return nil
}

func copyDirEntry(srcDir, dstDir string, entry os.DirEntry) error {
	srcPath, err := safeJoinPath(srcDir, entry.Name())
	if err != nil {
		return err
	}
	dstPath, err := safeJoinPath(dstDir, entry.Name())
	if err != nil {
		return err
	}
	if entry.IsDir() {
		if err := copyDir(srcPath, dstPath); err != nil {
			return fmt.Errorf("copy directory %s: %w", srcPath, err)
		}
		return nil
	}
	if err := copyFile(srcPath, dstPath); err != nil {
		return fmt.Errorf("copy file %s: %w", srcPath, err)
	}
	return nil
}

func copyDir(srcDir, dstDir string) error {
	if err := os.MkdirAll(dstDir, 0o750); err != nil {
		return fmt.Errorf("create directory %s: %w", dstDir, err)
	}
	return copyDirContents(srcDir, dstDir)
}

func copyFile(srcPath, dstPath string) (retErr error) {
	//nolint:gosec // srcPath is validated through safeJoinPath before use.
	src, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("open source file %s: %w", srcPath, err)
	}
	defer func() {
		retErr = errors.Join(retErr, closeFile(src, "source", srcPath))
	}()

	info, err := src.Stat()
	if err != nil {
		return fmt.Errorf("stat source file %s: %w", srcPath, err)
	}

	//nolint:gosec // dstPath is validated through safeJoinPath before use.
	dst, err := os.OpenFile(dstPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, info.Mode().Perm())
	if err != nil {
		return fmt.Errorf("open destination file %s: %w", dstPath, err)
	}
	defer func() {
		retErr = errors.Join(retErr, closeFile(dst, "destination", dstPath))
	}()

	if _, err := io.Copy(dst, src); err != nil {
		return fmt.Errorf("copy %s to %s: %w", srcPath, dstPath, err)
	}
	return nil
}

func closeFile(file io.Closer, kind, path string) error {
	if err := file.Close(); err != nil {
		return fmt.Errorf("close %s file %s: %w", kind, path, err)
	}
	return nil
}
