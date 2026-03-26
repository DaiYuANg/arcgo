package main

import (
	"errors"
	"fmt"
	"io"
	"os"
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

type bumpMode int

const (
	bumpPatch bumpMode = iota
	bumpMinor
	bumpMajor
)

type bumpTaskSpec struct {
	name string
	mode bumpMode
}

type releaseTarget struct {
	name   string
	latest semver.Version
	next   semver.Version
}

type modulePatchConfig struct {
	remote      string
	taggerName  string
	taggerEmail string
}

var bumpTaskSpecs = []bumpTaskSpec{
	{name: "patch", mode: bumpPatch},
	{name: "minor", mode: bumpMinor},
	{name: "major", mode: bumpMajor},
}

func main() {
	patch := defineBumpTasks()
	defineModulePatchTasks()

	goyek.Define(goyek.Task{
		Name:  "help",
		Usage: "Show script usage",
		Action: func(a *goyek.A) {
			mustWriteString(a.Output(), "Usage:\n")
			mustWriteString(a.Output(), "  go run ./scripts/tagger [task]\n")
			printUsage(a.Output())
		},
	})

	goyek.SetDefault(patch)
	goyek.SetUsage(func() {
		printUsage(os.Stderr)
	})

	goyek.Main(os.Args[1:])
}

func defineBumpTasks() *goyek.DefinedTask {
	var patch *goyek.DefinedTask
	lo.ForEach(bumpTaskSpecs, func(spec bumpTaskSpec, _ int) {
		mode := spec.mode
		name := spec.name

		localTask := goyek.Define(goyek.Task{
			Name:  name,
			Usage: fmt.Sprintf("Auto bump %s version and create local tag", name),
			Action: func(a *goyek.A) {
				runTagger(a, mode, false)
			},
		})
		if mode == bumpPatch {
			patch = localTask
		}

		goyek.Define(goyek.Task{
			Name:  name + "-push",
			Usage: fmt.Sprintf("Auto bump %s version and push tag", name),
			Action: func(a *goyek.A) {
				runTagger(a, mode, true)
			},
		})
	})
	return patch
}

func printUsage(w io.Writer) {
	mustWriteString(w, "Usage: go run ./scripts/tagger [task]\n")
	mustWriteString(w, "Tasks: patch, patch-push, minor, minor-push, major, major-push, modules-patch, modules-patch-push, modules-patch-dry-run, help\n")
	mustWriteString(w, "Env: TAGGER_REMOTE=origin TAGGER_NAME=auto-tagger TAGGER_EMAIL=ci@local TAGGER_MODULE_SCOPE=libs|all\n")
}

func mustWriteString(w io.Writer, text string) {
	if _, err := io.WriteString(w, text); err != nil {
		panic(err)
	}
}

func runTagger(a *goyek.A, mode bumpMode, push bool) {
	repo, err := git.PlainOpen(".")
	if err != nil {
		a.Fatal(err)
	}

	latest, err := latestSemverTag(repo)
	if err != nil {
		a.Fatal(err)
	}

	next := bump(latest, mode)
	newTag := "v" + next.String()
	a.Logf("New tag: %s", newTag)

	head, err := repo.Head()
	if err != nil {
		a.Fatal(err)
	}

	commit, err := repo.CommitObject(head.Hash())
	if err != nil {
		a.Fatal(err)
	}

	taggerName := getenvDefault("TAGGER_NAME", "auto-tagger")
	taggerEmail := getenvDefault("TAGGER_EMAIL", "ci@local")
	_, err = repo.CreateTag(newTag, commit.Hash, &git.CreateTagOptions{
		Tagger: &object.Signature{
			Name:  taggerName,
			Email: taggerEmail,
			When:  time.Now(),
		},
		Message: newTag,
	})
	if err != nil {
		a.Fatal(err)
	}
	a.Log("Tag created locally")

	remote := getenvDefault("TAGGER_REMOTE", "origin")
	if !push {
		a.Logf("Push manually: git push %s %s", remote, newTag)
		return
	}

	err = repo.Push(&git.PushOptions{
		RemoteName: remote,
		RefSpecs: []config.RefSpec{
			config.RefSpec("refs/tags/" + newTag + ":refs/tags/" + newTag),
		},
	})
	if err != nil && !errors.Is(err, git.NoErrAlreadyUpToDate) {
		a.Fatal(err)
	}

	a.Log("Tag pushed")
}

func defineModulePatchTasks() {
	goyek.Define(goyek.Task{
		Name:  "modules-patch",
		Usage: "Create next patch tags for all modules (default scope: libs)",
		Action: func(a *goyek.A) {
			runModulePatchTagger(a, false, false)
		},
	})

	goyek.Define(goyek.Task{
		Name:  "modules-patch-push",
		Usage: "Create and push next patch tags for all modules (default scope: libs)",
		Action: func(a *goyek.A) {
			runModulePatchTagger(a, true, false)
		},
	})

	goyek.Define(goyek.Task{
		Name:  "modules-patch-dry-run",
		Usage: "Show module patch tags that would be created (default scope: libs)",
		Action: func(a *goyek.A) {
			runModulePatchTagger(a, false, true)
		},
	})
}

func runModulePatchTagger(a *goyek.A, push, dryRun bool) {
	repo, err := git.PlainOpen(".")
	if err != nil {
		a.Fatal(err)
	}

	targets, err := modulePatchTargets(repo)
	if err != nil {
		a.Fatal(err)
	}
	if len(targets) == 0 {
		a.Log("No module tags found")
		return
	}

	commit, cfg, err := modulePatchContext(repo)
	if err != nil {
		a.Fatal(err)
	}

	for i := range targets {
		runModulePatchTarget(a, repo, commit, cfg, &targets[i], push, dryRun)
	}

	logModulePatchSummary(a, len(targets), cfg.remote, push, dryRun)
}

func modulePatchContext(repo *git.Repository) (*object.Commit, modulePatchConfig, error) {
	head, err := repo.Head()
	if err != nil {
		return nil, modulePatchConfig{}, fmt.Errorf("resolve repository head: %w", err)
	}
	commit, err := repo.CommitObject(head.Hash())
	if err != nil {
		return nil, modulePatchConfig{}, fmt.Errorf("resolve head commit: %w", err)
	}
	return commit, modulePatchConfig{
		remote:      getenvDefault("TAGGER_REMOTE", "origin"),
		taggerName:  getenvDefault("TAGGER_NAME", "auto-tagger"),
		taggerEmail: getenvDefault("TAGGER_EMAIL", "ci@local"),
	}, nil
}

func runModulePatchTarget(
	a *goyek.A,
	repo *git.Repository,
	commit *object.Commit,
	cfg modulePatchConfig,
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

func logModulePatchSummary(a *goyek.A, count int, remote string, push, dryRun bool) {
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

func modulePatchTargets(repo *git.Repository) ([]releaseTarget, error) {
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
		next := latest.IncPatch()
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

func latestSemverTag(repo *git.Repository) (semver.Version, error) {
	iter, err := repo.Tags()
	if err != nil {
		return semver.Version{}, fmt.Errorf("iterate repository tags: %w", err)
	}

	latest := semver.New(0, 0, 0, "", "")
	found := false

	err = iter.ForEach(func(ref *plumbing.Reference) error {
		tag := ref.Name().Short()
		v, ok := parseSemverTag(tag)
		if !ok {
			return nil
		}
		if !found || v.GreaterThan(latest) {
			latest = v
			found = true
		}
		return nil
	})
	if err != nil {
		return semver.Version{}, fmt.Errorf("scan semver tags: %w", err)
	}

	if !found {
		return *semver.New(0, 0, 0, "", ""), nil
	}
	return *latest, nil
}

func parseSemverTag(tag string) (*semver.Version, bool) {
	if !strings.HasPrefix(tag, "v") {
		return nil, false
	}

	v, err := semver.StrictNewVersion(strings.TrimPrefix(tag, "v"))
	if err != nil {
		return nil, false
	}
	// Keep compatibility with previous behavior: only stable vX.Y.Z tags.
	if v.Prerelease() != "" || v.Metadata() != "" {
		return nil, false
	}
	return v, true
}

func bump(v semver.Version, mode bumpMode) semver.Version {
	switch mode {
	case bumpPatch:
		return v.IncPatch()
	case bumpMajor:
		return v.IncMajor()
	case bumpMinor:
		return v.IncMinor()
	}
	return v.IncPatch()
}

func getenvDefault(key, fallback string) string {
	val := strings.TrimSpace(os.Getenv(key))
	if val == "" {
		return fallback
	}
	return val
}
