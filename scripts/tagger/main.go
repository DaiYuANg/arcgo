package main

import (
	"fmt"
	"log"
	"regexp"
	"sort"
	"strconv"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
)

func main() {

	// 打开当前仓库
	repo, err := git.PlainOpen(".")
	if err != nil {
		log.Fatal(err)
	}

	// 获取所有 tag
	tagIter, err := repo.Tags()
	if err != nil {
		log.Fatal(err)
	}

	var tags []string

	err = tagIter.ForEach(func(ref *plumbing.Reference) error {
		tags = append(tags, ref.Name().Short())
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}

	re := regexp.MustCompile(`v(\d+)\.(\d+)\.(\d+)`)

	// 过滤符合 semver 的 tag
	var versions []string
	for _, t := range tags {
		if re.MatchString(t) {
			versions = append(versions, t)
		}
	}

	if len(versions) == 0 {
		versions = append(versions, "v0.0.0")
	}

	// 排序找到最大版本
	sort.Slice(versions, func(i, j int) bool {
		return versions[i] < versions[j]
	})

	latest := versions[len(versions)-1]

	match := re.FindStringSubmatch(latest)

	major, _ := strconv.Atoi(match[1])
	minor, _ := strconv.Atoi(match[2])
	patch, _ := strconv.Atoi(match[3])

	patch++

	newTag := fmt.Sprintf("v%d.%d.%d", major, minor, patch)

	fmt.Println("New tag:", newTag)

	// 获取 HEAD commit
	head, err := repo.Head()
	if err != nil {
		log.Fatal(err)
	}

	commit, err := repo.CommitObject(head.Hash())
	if err != nil {
		log.Fatal(err)
	}

	// 创建 annotated tag
	_, err = repo.CreateTag(newTag, commit.Hash, &git.CreateTagOptions{
		Tagger: &object.Signature{
			Name:  "auto-tagger",
			Email: "ci@local",
		},
		Message: newTag,
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Tag created")

	// push tag
	err = repo.Push(&git.PushOptions{
		RefSpecs: []config.RefSpec{
			config.RefSpec("refs/tags/" + newTag + ":refs/tags/" + newTag),
		},
	})

	if err != nil && err != git.NoErrAlreadyUpToDate {
		log.Fatal(err)
	}

	fmt.Println("Tag pushed 🚀")
}
