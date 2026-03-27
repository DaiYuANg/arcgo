package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// writeVersionsFile 写入 versions.yaml 文件
func writeVersionsFile(filename string, versions []Version) error {
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return fmt.Errorf("无法创建目录：%w", err)
	}

	if err := os.WriteFile(filename, []byte(renderVersionsYAML(versions)), 0o600); err != nil {
		return fmt.Errorf("无法写入文件：%w", err)
	}
	return nil
}

func renderVersionsYAML(versions []Version) string {
	var b strings.Builder
	mustWriteString(&b, `# 版本文档配置
# 此文件定义了文档的版本列表
# 版本按时间倒序排列，第一个为当前版本

versions:
`)

	for i, version := range versions {
		mustWriteString(&b, versionSection(version.Current))
		mustFprintf(&b, "  - name: \"%s\"\n", version.Name)
		mustFprintf(&b, "    release: \"%s\"\n", version.Release)
		mustFprintf(&b, "    path: \"%s\"\n", version.Path)
		mustFprintf(&b, "    current: %t\n", version.Current)

		if i < len(versions)-1 {
			mustWriteString(&b, "\n")
		}
	}

	return b.String()
}

func versionSection(current bool) string {
	if current {
		return "  # 当前版本（最新版本）\n"
	}
	return "\n  # 历史版本\n"
}
