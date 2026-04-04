package main

import (
	"errors"
	"fmt"
	"io"
	"os"
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

var bumpTaskSpecs = []bumpTaskSpec{
	{name: "patch", mode: bumpPatch},
	{name: "minor", mode: bumpMinor},
	{name: "major", mode: bumpMajor},
}

func main() {
	patch := defineBumpTasks()
	defineModuleBumpTasks()

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
	mustWriteString(w, "Tasks: patch, patch-push, minor, minor-push, major, major-push, modules-patch, modules-patch-push, modules-patch-dry-run, modules-minor, modules-minor-push, modules-minor-dry-run, modules-major, modules-major-push, modules-major-dry-run, help\n")
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
	case bumpMinor:
		return v.IncMinor()
	case bumpMajor:
		return v.IncMajor()
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
