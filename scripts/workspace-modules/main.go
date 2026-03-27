// workspace-modules runs go test, golangci-lint, staticcheck, or govulncheck in each go.work module.
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var errUsage = errors.New("invalid usage")

type moduleCommand struct {
	name string
	argv func(dir string) []string
}

func main() {
	os.Exit(run(context.Background(), os.Args[1:], os.Stdout, os.Stderr))
}

func run(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	if len(args) < 1 {
		usage(stderr)
		return 2
	}

	root, err := findRepoRoot()
	if err != nil {
		fatal(stderr, err)
		return 1
	}
	if chdirErr := os.Chdir(root); chdirErr != nil {
		fatal(stderr, fmt.Errorf("chdir repo root: %w", chdirErr))
		return 1
	}

	command, err := commandForTask(ctx, root, args[0])
	if err != nil {
		if errors.Is(err, errUsage) {
			usage(stderr)
			return 2
		}
		fatal(stderr, err)
		return 1
	}
	if err := runPerModule(ctx, root, command, stdout); err != nil {
		fatal(stderr, err)
		return 1
	}

	return 0
}

func usage(w io.Writer) {
	mustWriteString(w, "usage: go run ./scripts/workspace-modules/main.go <test|lint|staticcheck|govulncheck>\n")
}

func fatal(w io.Writer, err error) {
	mustWriteString(w, fmt.Sprintf("workspace-modules: %v\n", err))
}

func mustWriteString(w io.Writer, text string) {
	if _, err := io.WriteString(w, text); err != nil {
		panic(err)
	}
}

func commandForTask(ctx context.Context, root, task string) (moduleCommand, error) {
	switch task {
	case "test":
		return moduleCommand{
			name: "test",
			argv: func(_ string) []string { return []string{"go", "test", "./..."} },
		}, nil
	case "lint":
		bin, err := toolPath(ctx, root, "golangci-lint")
		if err != nil {
			return moduleCommand{}, fmt.Errorf("golangci-lint: %w", err)
		}
		return moduleCommand{
			name: "lint",
			argv: func(_ string) []string { return []string{bin, "run", "./..."} },
		}, nil
	case "staticcheck":
		bin, err := toolPath(ctx, root, "staticcheck")
		if err != nil {
			return moduleCommand{}, fmt.Errorf("staticcheck: %w", err)
		}
		checks := strings.TrimSpace(os.Getenv("STATICCHECK_CHECKS"))
		return moduleCommand{
			name: "staticcheck",
			argv: func(_ string) []string {
				if checks != "" {
					return []string{bin, "-checks", checks, "./..."}
				}
				return []string{bin, "./..."}
			},
		}, nil
	case "govulncheck":
		bin, err := toolPath(ctx, root, "govulncheck")
		if err != nil {
			return moduleCommand{}, fmt.Errorf("govulncheck: %w", err)
		}
		return moduleCommand{
			name: "govulncheck",
			argv: func(_ string) []string { return []string{bin, "./..."} },
		}, nil
	default:
		return moduleCommand{}, errUsage
	}
}

func toolPath(ctx context.Context, root, tool string) (string, error) {
	switch tool {
	case "golangci-lint":
		cmd := exec.CommandContext(ctx, "go", "tool", "-n", "golangci-lint")
		cmd.Dir = root
		return commandPathOutput(cmd, tool)
	case "staticcheck":
		cmd := exec.CommandContext(ctx, "go", "tool", "-n", "staticcheck")
		cmd.Dir = root
		return commandPathOutput(cmd, tool)
	case "govulncheck":
		cmd := exec.CommandContext(ctx, "go", "tool", "-n", "govulncheck")
		cmd.Dir = root
		return commandPathOutput(cmd, tool)
	default:
		return "", fmt.Errorf("unsupported tool: %s", tool)
	}
}

func commandPathOutput(cmd *exec.Cmd, tool string) (string, error) {
	out, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && len(exitErr.Stderr) > 0 {
			return "", fmt.Errorf("resolve tool %s: %w: %s", tool, err, strings.TrimSpace(string(exitErr.Stderr)))
		}
		return "", fmt.Errorf("resolve tool %s: %w", tool, err)
	}
	return strings.TrimSpace(string(out)), nil
}

func runPerModule(ctx context.Context, root string, command moduleCommand, stdout io.Writer) error {
	dirs, err := moduleDirs(ctx, root)
	if err != nil {
		return err
	}

	failed := make([]string, 0)
	for _, dir := range dirs {
		rel := moduleRelativePath(root, dir)
		mustWriteString(stdout, fmt.Sprintf("=== %s (%s) ===\n", rel, command.name))
		if err := runModuleCommand(ctx, dir, command.argv(dir)); err != nil {
			failed = append(failed, rel)
		}
	}
	if len(failed) > 0 {
		return fmt.Errorf("workspace-modules %s: failed modules: %s", command.name, strings.Join(failed, ", "))
	}
	return nil
}

func moduleRelativePath(root, dir string) string {
	rel, err := filepath.Rel(root, dir)
	if err != nil || rel == "" || rel == "." {
		return "."
	}
	return rel
}

func runModuleCommand(ctx context.Context, dir string, args []string) error {
	if len(args) == 0 {
		return nil
	}

	//nolint:gosec // Commands come from internal task definitions and resolved tool paths.
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("run %q in %s: %w", strings.Join(args, " "), dir, err)
	}
	return nil
}

func moduleDirs(ctx context.Context, root string) ([]string, error) {
	cmd := exec.CommandContext(ctx, "go", "list", "-m", "-f", "{{.Dir}}")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("go list modules: %w", err)
	}

	dirs := make([]string, 0)
	for line := range strings.SplitSeq(strings.TrimSpace(string(out)), "\n") {
		dir := strings.TrimSpace(line)
		if dir != "" {
			dirs = append(dirs, dir)
		}
	}
	return dirs, nil
}

func findRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("get working directory: %w", err)
	}

	for {
		_, statErr := os.Stat(filepath.Join(dir, "go.work"))
		if statErr == nil {
			return dir, nil
		}
		if !errors.Is(statErr, os.ErrNotExist) {
			return "", fmt.Errorf("stat go.work in %s: %w", dir, statErr)
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("go.work not found from %s", dir)
		}
		dir = parent
	}
}
