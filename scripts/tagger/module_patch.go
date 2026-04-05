package main

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	semver "github.com/Masterminds/semver/v3"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/goyek/goyek/v3"
	"github.com/samber/lo"
)

type releaseTarget struct {
	name   string
	latest semver.Version
	next   semver.Version
}

type moduleBumpConfig struct {
	remote      string
	taggerName  string
	taggerEmail string
}

func defineModuleBumpTasks() {
	lo.ForEach(bumpTaskSpecs, func(spec bumpTaskSpec, _ int) {
		mode := spec.mode
		name := spec.name

		goyek.Define(goyek.Task{
			Name:  "modules-" + name,
			Usage: fmt.Sprintf("Create next %s tags for all modules (default scope: libs)", name),
			Action: func(a *goyek.A) {
				runModuleBumpTagger(a, mode, false, false)
			},
		})

		goyek.Define(goyek.Task{
			Name:  "modules-" + name + "-push",
			Usage: fmt.Sprintf("Create and push next %s tags for all modules (default scope: libs)", name),
			Action: func(a *goyek.A) {
				runModuleBumpTagger(a, mode, true, false)
			},
		})

		goyek.Define(goyek.Task{
			Name:  "modules-" + name + "-dry-run",
			Usage: fmt.Sprintf("Show module %s tags that would be created (default scope: libs)", name),
			Action: func(a *goyek.A) {
				runModuleBumpTagger(a, mode, false, true)
			},
		})
	})
}

func runModuleBumpTagger(a *goyek.A, mode bumpMode, push, dryRun bool) {
	repo, err := git.PlainOpen(".")
	if err != nil {
		a.Fatal(err)
	}

	targets, err := moduleBumpTargets(repo, mode)
	if err != nil {
		a.Fatal(err)
	}
	if len(targets) == 0 {
		a.Log("No module tags found")
		return
	}

	commit, cfg, err := moduleBumpContext(repo)
	if err != nil {
		a.Fatal(err)
	}

	lo.ForEach(targets, func(item releaseTarget, index int) {
		runModuleBumpTarget(a, repo, commit, cfg, &targets[index], push, dryRun)
	})

	logModuleBumpSummary(a, len(targets), cfg.remote, push, dryRun)
}

func moduleBumpContext(repo *git.Repository) (*object.Commit, moduleBumpConfig, error) {
	head, err := repo.Head()
	if err != nil {
		return nil, moduleBumpConfig{}, fmt.Errorf("resolve repository head: %w", err)
	}
	commit, err := repo.CommitObject(head.Hash())
	if err != nil {
		return nil, moduleBumpConfig{}, fmt.Errorf("resolve head commit: %w", err)
	}
	return commit, moduleBumpConfig{
		remote:      getenvDefault("TAGGER_REMOTE", "origin"),
		taggerName:  getenvDefault("TAGGER_NAME", "auto-tagger"),
		taggerEmail: getenvDefault("TAGGER_EMAIL", "ci@local"),
	}, nil
}

func runModuleBumpTarget(
	a *goyek.A,
	repo *git.Repository,
	commit *object.Commit,
	cfg moduleBumpConfig,
	target *releaseTarget,
	push, dryRun bool,
) {
	newTag := fmt.Sprintf("%s/v%s", target.name, target.next.String())
	a.Logf("%s -> %s", target.name, newTag)
	if dryRun {
		return
	}

	_, err := repo.CreateTag(newTag, commit.Hash, &git.CreateTagOptions{
		Tagger: &object.Signature{
			Name:  cfg.taggerName,
			Email: cfg.taggerEmail,
			When:  time.Now(),
		},
		Message: newTag,
	})
	if err != nil {
		a.Fatalf("create tag %s: %v", newTag, err)
	}

	if !push {
		return
	}

	err = repo.Push(&git.PushOptions{
		RemoteName: cfg.remote,
		RefSpecs: []config.RefSpec{
			config.RefSpec("refs/tags/" + newTag + ":refs/tags/" + newTag),
		},
	})
	if err != nil && !errors.Is(err, git.NoErrAlreadyUpToDate) {
		a.Fatalf("push tag %s: %v", newTag, err)
	}
}

func logModuleBumpSummary(a *goyek.A, count int, remote string, push, dryRun bool) {
	if dryRun {
		a.Logf("Dry-run complete (%d tags)", count)
		return
	}

	if push {
		a.Logf("Created and pushed %d module tags", count)
		return
	}

	a.Logf("Created %d module tags locally", count)
	a.Logf("Push manually with: git push %s --tags", remote)
}

func moduleBumpTargets(repo *git.Repository, mode bumpMode) ([]releaseTarget, error) {
	iter, err := repo.Tags()
	if err != nil {
		return nil, fmt.Errorf("iterate repository tags: %w", err)
	}

	latestByModule := map[string]semver.Version{}
	err = iter.ForEach(func(ref *plumbing.Reference) error {
		tag := ref.Name().Short()
		moduleName, version, ok := parseModuleSemverTag(tag)
		if !ok {
			return nil
		}
		if !includeModule(moduleName) {
			return nil
		}
		if current, exists := latestByModule[moduleName]; !exists || version.GreaterThan(&current) {
			latestByModule[moduleName] = *version
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("scan module tags: %w", err)
	}

	modules := lo.Keys(latestByModule)
	sort.Strings(modules)

	targets := make([]releaseTarget, 0, len(modules))
	for _, moduleName := range modules {
		latest := latestByModule[moduleName]
		next := bump(latest, mode)
		targets = append(targets, releaseTarget{
			name:   moduleName,
			latest: latest,
			next:   next,
		})
	}
	return targets, nil
}

func parseModuleSemverTag(tag string) (string, *semver.Version, bool) {
	i := strings.LastIndex(tag, "/")
	if i <= 0 || i == len(tag)-1 {
		return "", nil, false
	}

	moduleName := tag[:i]
	versionPart := tag[i+1:]
	version, ok := parseSemverTag(versionPart)
	if !ok {
		return "", nil, false
	}
	return moduleName, version, true
}

func includeModule(moduleName string) bool {
	scope := strings.ToLower(strings.TrimSpace(getenvDefault("TAGGER_MODULE_SCOPE", "libs")))
	if scope == "all" {
		return true
	}
	if moduleName == "docs" {
		return false
	}
	return !strings.HasPrefix(moduleName, "examples/") &&
		!strings.HasPrefix(moduleName, "docs/") &&
		!strings.HasPrefix(moduleName, "pkg/")
}
