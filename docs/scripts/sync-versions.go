// sync-versions.go
// 版本文档同步工具
// 用途：从 git tags 自动创建版本文档目录和配置文件

package main

import (
	"os"
	"path/filepath"
)

const separatorLine = "========================================"

type syncPaths struct {
	docsDir      string
	contentDir   string
	versionsFile string
}

func main() {
	printBanner()

	paths := resolvePaths()
	tags := loadTags()
	latestTag := tags[0]

	printCurrentVersion(latestTag)

	versions := createVersionsConfig(tags)
	writeVersions(paths.versionsFile, versions)
	createVersionDirs(paths.contentDir, versions)

	printSummary(latestTag, len(tags))
}

func printBanner() {
	stdoutln(separatorLine)
	stdoutln("   ArcGo 版本文档同步工具")
	stdoutln(separatorLine)
	stdoutln()
}

func resolvePaths() syncPaths {
	projectRoot, err := getProjectRoot()
	if err != nil {
		exitWithError(err)
	}

	docsDir := filepath.Join(projectRoot, "docs")
	return syncPaths{
		docsDir:      docsDir,
		contentDir:   filepath.Join(docsDir, "content"),
		versionsFile: filepath.Join(docsDir, "data", "versions.yaml"),
	}
}

func loadTags() []string {
	stdoutln("[1/4] 获取 git tags...")
	tags, err := getGitTags()
	if err != nil {
		exitWithError(err)
	}
	if len(tags) == 0 {
		stdoutln("⚠️  未找到 git tags，使用 short commit 作为版本标识")
		shortCommit, err := getShortCommit()
		if err != nil {
			exitWithError(err)
		}
		stdoutf("✅ 使用版本：%s (short commit)\n\n", shortCommit)
		return []string{shortCommit}
	}

	stdoutln("✅ 找到以下 tags:")
	for _, tag := range tags {
		stdoutf("   - %s\n", tag)
	}
	stdoutln()
	return tags
}

func printCurrentVersion(latestTag string) {
	stdoutf("[2/4] 当前版本：%s\n", latestTag)
	stdoutln()
}

func writeVersions(versionsFile string, versions []Version) {
	stdoutln("[3/4] 创建版本配置文件...")
	if err := writeVersionsFile(versionsFile, versions); err != nil {
		exitWithError(err)
	}
	stdoutf("✅ 版本文档配置已更新到：%s\n", versionsFile)
	stdoutln()
}

func createVersionDirs(contentDir string, versions []Version) {
	stdoutln("[4/4] 创建版本文档目录...")
	if err := createVersionedDirs(contentDir, versions); err != nil {
		exitWithError(err)
	}
	stdoutln()
}

func printSummary(latestTag string, total int) {
	stdoutln(separatorLine)
	stdoutln("   版本统计")
	stdoutln(separatorLine)
	stdoutf("   当前版本：%s\n", latestTag)
	stdoutf("   历史版本数：%d 个\n", total)
	stdoutln(separatorLine)
	stdoutln()
	stdoutln("💡 提示：运行 'go tool hugo server -D' 预览版本文档")
	stdoutln()
}

func exitWithError(err error) {
	stderrf("❌ 错误：%v\n", err)
	os.Exit(1)
}
