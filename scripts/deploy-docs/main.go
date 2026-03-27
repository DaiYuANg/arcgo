package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/goyek/goyek/v3"
	goyekcmd "github.com/goyek/x/cmd"
)

type docsContext struct {
	rootDir      string
	docsDir      string
	hugoCacheDir string
}

func main() {
	ctx, err := newDocsContext()
	if err != nil {
		fatal(err)
	}

	defineDocsTasks(ctx)
	goyek.SetUsage(func() {
		printUsage(os.Stderr)
	})
	goyek.Main(os.Args[1:])
}

func fatal(err error) {
	mustWriteString(os.Stderr, fmt.Sprintf("error: %v\n", err))
	os.Exit(1)
}

func defineDocsTasks(ctx *docsContext) {
	build := defineBuildTask(ctx)
	defineSyncTask(ctx)
	defineServeTask(ctx)
	defineDeployTask(ctx, build)
	defineHelpTask()
}

func defineBuildTask(ctx *docsContext) *goyek.DefinedTask {
	return goyek.Define(goyek.Task{
		Name:  "build",
		Usage: "Build docs with Hugo",
		Action: func(a *goyek.A) {
			if !execHugo(a, ctx, "--gc --minify") {
				a.FailNow()
			}
			a.Logf("build complete: %s", filepath.Join(ctx.docsDir, "public"))
		},
	})
}

func defineSyncTask(ctx *docsContext) {
	goyek.Define(goyek.Task{
		Name:  "sync",
		Usage: "Sync version metadata from git tags",
		Action: func(a *goyek.A) {
			if !goyekcmd.Exec(a, "go run ./scripts", goyekcmd.Dir(ctx.docsDir)) {
				a.FailNow()
			}
		},
	})
}

func defineServeTask(ctx *docsContext) {
	goyek.Define(goyek.Task{
		Name:  "serve",
		Usage: "Run local Hugo server",
		Action: func(a *goyek.A) {
			a.Log("visit: http://127.0.0.1:1313")
			if !execHugo(a, ctx, "server -D --buildDrafts --disableFastRender") {
				a.FailNow()
			}
		},
	})
}

func defineDeployTask(ctx *docsContext, build *goyek.DefinedTask) {
	goyek.Define(goyek.Task{
		Name:  "deploy",
		Usage: "Build and force-push docs/public to gh-pages (set DOCS_REMOTE / DOCS_BRANCH to override defaults)",
		Deps:  goyek.Deps{build},
		Action: func(a *goyek.A) {
			if err := deployDocs(a, ctx); err != nil {
				a.Fatal(err)
			}
		},
	})
}

func defineHelpTask() {
	goyek.Define(goyek.Task{
		Name:  "help",
		Usage: "Show script usage",
		Action: func(a *goyek.A) {
			printUsage(a.Output())
		},
	})
}

func deployDocs(a *goyek.A, ctx *docsContext) error {
	remote := getenvDefault("DOCS_REMOTE", "origin")
	branch := getenvDefault("DOCS_BRANCH", "gh-pages")

	repoURL, err := gitOutput(ctx.rootDir, "remote", "get-url", remote)
	if err != nil {
		return fmt.Errorf("cannot resolve remote URL for %q: %w", remote, err)
	}

	tempDir := filepath.Join(ctx.docsDir, ".tmp-public")
	if err := os.RemoveAll(tempDir); err != nil {
		return fmt.Errorf("remove temp docs dir: %w", err)
	}
	if err := os.MkdirAll(tempDir, 0o750); err != nil {
		return fmt.Errorf("create temp docs dir: %w", err)
	}
	if err := copyDirContents(filepath.Join(ctx.docsDir, "public"), tempDir); err != nil {
		return fmt.Errorf("copy built docs: %w", err)
	}
	if err := os.WriteFile(filepath.Join(tempDir, ".nojekyll"), []byte{}, 0o600); err != nil {
		return fmt.Errorf("write .nojekyll marker: %w", err)
	}

	execGitOrFail(a, tempDir, "init")
	execGitOrFail(a, tempDir, "checkout", "-b", branch)
	execGitOrFail(a, tempDir, "add", "-A")

	if isCleanGitIndex(tempDir) {
		a.Log("no changes to deploy")
		return nil
	}

	commitMsg := "docs: deploy " + time.Now().UTC().Format(time.RFC3339)
	execGitOrFail(a, tempDir, "commit", "-m", commitMsg)
	execGitOrFail(a, tempDir, "remote", "add", "origin", repoURL)
	execGitOrFail(a, tempDir, "push", "-f", "origin", branch)
	a.Logf("deployed to %s/%s", remote, branch)
	return nil
}

func execHugo(a *goyek.A, ctx *docsContext, args string) bool {
	if err := os.MkdirAll(ctx.hugoCacheDir, 0o750); err != nil {
		a.Fatal(err)
	}
	return goyekcmd.Exec(
		a,
		"go tool hugo "+strings.TrimSpace(args),
		goyekcmd.Dir(ctx.docsDir),
	)
}

func printUsage(w io.Writer) {
	mustWriteString(w, "Usage: go run ./scripts/deploy-docs [task]\n")
	mustWriteString(w, "Tasks: sync, build, serve, deploy, help\n")
	mustWriteString(w, "Deploy env: DOCS_REMOTE=origin DOCS_BRANCH=gh-pages\n")
}

func mustWriteString(w io.Writer, text string) {
	if _, err := io.WriteString(w, text); err != nil {
		panic(err)
	}
}

func execGitOrFail(a *goyek.A, dir string, args ...string) {
	cmdLine := "git " + strings.Join(args, " ")
	if !goyekcmd.Exec(a, cmdLine, goyekcmd.Dir(dir)) {
		a.FailNow()
	}
}

func newDocsContext() (*docsContext, error) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		return nil, errors.New("cannot resolve script path")
	}
	scriptDir := filepath.Dir(thisFile)
	rootDir := filepath.Clean(filepath.Join(scriptDir, "..", ".."))
	docsDir := filepath.Join(rootDir, "docs")

	info, err := os.Stat(docsDir)
	if err != nil {
		return nil, fmt.Errorf("stat docs directory: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("invalid docs directory: %s", docsDir)
	}

	return &docsContext{
		rootDir:      rootDir,
		docsDir:      docsDir,
		hugoCacheDir: filepath.Join(docsDir, ".cache", "hugo"),
	}, nil
}

func gitOutput(dir string, args ...string) (string, error) {
	//nolint:gosec // Git arguments come from fixed internal call sites in this script.
	cmd := exec.CommandContext(context.Background(), "git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return strings.TrimSpace(string(out)), nil
}

func isCleanGitIndex(dir string) bool {
	cmd := exec.CommandContext(context.Background(), "git", "diff", "--cached", "--quiet")
	cmd.Dir = dir
	return cmd.Run() == nil
}

func getenvDefault(key, fallback string) string {
	val := strings.TrimSpace(os.Getenv(key))
	if val == "" {
		return fallback
	}
	return val
}
