package main

import (
	"fmt"
	"path/filepath"
	"sort"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

// openRepo 打开当前目录所在的 git 仓库
func openRepo() (*git.Repository, error) {
	repo, err := git.PlainOpenWithOptions(".", &git.PlainOpenOptions{
		DetectDotGit: true,
	})
	if err != nil {
		return nil, fmt.Errorf("无法打开 git 仓库：%w", err)
	}
	return repo, nil
}

// getProjectRoot 获取项目根目录
func getProjectRoot() (string, error) {
	repo, err := openRepo()
	if err != nil {
		return "", err
	}
	wt, err := repo.Worktree()
	if err != nil {
		return "", fmt.Errorf("无法获取 worktree：%w", err)
	}
	root, err := filepath.Abs(wt.Filesystem.Root())
	if err != nil {
		return "", fmt.Errorf("无法解析项目根目录：%w", err)
	}
	return root, nil
}

// getGitTags 获取所有 git tags，按版本号降序排序
func getGitTags() ([]string, error) {
	repo, err := openRepo()
	if err != nil {
		return nil, err
	}
	iter, err := repo.Tags()
	if err != nil {
		return nil, fmt.Errorf("无法获取 git tags：%w", err)
	}
	defer iter.Close()

	var tags []string
	err = iter.ForEach(func(ref *plumbing.Reference) error {
		name := ref.Name().Short()
		if name != "" {
			tags = append(tags, name)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("遍历 git tags 失败：%w", err)
	}
	sort.Slice(tags, func(i, j int) bool {
		return compareVersions(tags[i], tags[j]) > 0
	})
	return tags, nil
}

// getShortCommit 获取当前 HEAD 的 short commit hash
func getShortCommit() (string, error) {
	repo, err := openRepo()
	if err != nil {
		return "", err
	}
	head, err := repo.Head()
	if err != nil {
		return "", fmt.Errorf("无法获取 HEAD：%w", err)
	}
	hash := head.Hash().String()
	if len(hash) > 7 {
		hash = hash[:7]
	}
	return hash, nil
}
