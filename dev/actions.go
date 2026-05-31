package main

// actions.go defines every menu entry and implements the underlying Go logic
// that was previously spread across multiple shell scripts. Each action is
// represented by the action struct, and the allActions slice is the single
// source of truth for what appears in the menu.

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ─── Action Definition ────────────────────────────────────────────────────────

// action is a single menu entry. Exactly one execution strategy should be
// populated per action, determining how the model handles it:
//
//   - makeCmd:      runs a single command interactively via tea.ExecProcess.
//     The TUI suspends and the user sees live output until the process exits.
//   - run:          runs a multi-step Go function in a background goroutine.
//     The TUI shows a spinner, then the captured output in a scrollable modal.
//   - inputPrompt + makeInputCmd: prompts the user for text, then runs the
//     resulting command interactively via tea.ExecProcess.
//   - inputPrompt + runWithInput: prompts the user for text, then runs a
//     Go function in a background goroutine.
//   - releaseScript: opens a release form (version + channel), then runs the
//     named script as a captured action and shows its output in a modal.
type action struct {
	title       string
	description string

	// Direct interactive command — TUI suspends while it runs.
	makeCmd func(repoRoot string) *exec.Cmd

	// Background Go function — shows spinner, then the captured output modal.
	// Returns the combined command output AND an error.
	run func(repoRoot string) (string, error)

	// Text input is collected first, then one of the following executes.
	inputPrompt  string
	makeInputCmd func(repoRoot, input string) *exec.Cmd // → interactive
	runWithInput func(repoRoot, input string) error     // → background goroutine

	// Release script path, relative to the repo root. When set, selecting the
	// action opens stateReleaseForm; on submit the script is run as a captured
	// action: bash <repoRoot>/<releaseScript> <version> [--prerelease].
	releaseScript string
}

// ─── Go Command Helper ────────────────────────────────────────────────────────

// goCmd creates a `go` exec.Cmd rooted at dir with GOWORK=off so the command
// resolves dependencies from the module's own go.mod/go.sum, independent of the
// repo-root go.work workspace. This matches how release and Docker builds
// resolve modules, keeping the dev hub's artifacts reproducible and unaffected
// by the other modules in the workspace.
func goCmd(dir string, args ...string) *exec.Cmd {
	cmd := exec.Command("go", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOWORK=off")
	return cmd
}

// ─── Action Registry ──────────────────────────────────────────────────────────

// allActions is the ordered list of menu entries shown in the dev hub.
var allActions = []*action{

	// ── CLI Tool ──────────────────────────────────────────────────────────────

	{
		title:       "CLI › Dev Install",
		description: "Build bs3 into cli-tool/.testing/ and copy it to ~/.local/bin",
		run:         cliDevInstall,
	},
	{
		title:       "CLI › Cross-Compile Build",
		description: "Build linux/amd64 + linux/arm64 zips into cli-tool/.builds/",
		run:         cliCrossCompileBuild,
	},
	{
		title:       "CLI › Run TUI",
		description: "Launch the bs3 interactive TUI (go run . tui from cli-tool/)",
		makeCmd: func(repoRoot string) *exec.Cmd {
			return goCmd(filepath.Join(repoRoot, "cli-tool"), "run", ".", "tui")
		},
	},
	{
		title:         "CLI › Release",
		description:   "Run dev/scripts/release-cli.sh — tag the version and create a GitHub release for the CLI",
		releaseScript: "dev/scripts/release-cli.sh",
	},

	// ── Server: local testing ─────────────────────────────────────────────────
	// Mirrors the README "Server Deployment" section.

	{
		title:       "Server › Local Test (Docker)",
		description: "Build the image from local source via server/compose/compose.dev.yml, run it on :8080",
		makeCmd: func(repoRoot string) *exec.Cmd {
			// compose.dev.yml has the build step; ctrl+c stops it.
			cmd := exec.Command("docker", "compose",
				"-f", "server/compose/compose.dev.yml", "up", "--build")
			cmd.Dir = repoRoot
			return cmd
		},
	},
	{
		title:       "Server › Stop Local Test",
		description: "docker compose down — stop and remove the local test container",
		run:         serverComposeDown,
	},
	{
		title:       "Server › Clean Dev Environment",
		description: "Destroy the local test container, image, volume, and network (deletes local vault data)",
		run:         serverCleanDev,
	},
	{
		title:       "Server › Run Dev Server",
		description: "Run the API server from source (go run ./cmd/ from server/)",
		makeCmd: func(repoRoot string) *exec.Cmd {
			return goCmd(filepath.Join(repoRoot, "server"), "run", "./cmd/")
		},
	},
	{
		title:       "Server › Build From Source",
		description: "Compile the server binary to server/bs3-server (GOWORK=off go build ./cmd)",
		run:         serverBuildBinary,
	},

	// ── Server: release & deploy ──────────────────────────────────────────────

	{
		title:       "Docker › Login",
		description: "docker login ghcr.io — authenticate to GHCR before releasing (PAT with write:packages)",
		makeCmd: func(repoRoot string) *exec.Cmd {
			return exec.Command("docker", "login", "ghcr.io")
		},
	},
	{
		title:         "Server › Release",
		description:   "Run dev/scripts/release.sh — build & push the multi-platform image and create a GitHub release",
		releaseScript: "dev/scripts/release.sh",
	},
	{
		title:       "Server › Deploy (Production)",
		description: "docker compose pull && up -d via server/compose/compose.prod.yml (the published image)",
		run:         serverDeployProd,
	},

	// ── Logger ────────────────────────────────────────────────────────────────

	{
		title:       "Logger › Run Tests",
		description: "Run the logger visual test program (go run ./cmd/ from logger/)",
		run:         loggerRunTests,
	},
}

// ─── CLI Tool Implementations ─────────────────────────────────────────────────

// cliDevInstall builds the bs3 binary into cli-tool/.testing/ and copies it
// to ~/.local/bin/bs3. It accumulates a short progress log and returns it
// alongside any error.
func cliDevInstall(repoRoot string) (string, error) {
	var log strings.Builder
	cliDir := filepath.Join(repoRoot, "cli-tool")
	testDir := filepath.Join(cliDir, ".testing")
	binPath := filepath.Join(testDir, "bs3")

	if err := os.MkdirAll(testDir, 0o755); err != nil {
		return log.String(), fmt.Errorf("create .testing dir: %w", err)
	}
	fmt.Fprintf(&log, "Created %s\n", testDir)

	// Build the binary.
	cmd := goCmd(cliDir, "build", "-o", binPath)
	if out, err := cmd.CombinedOutput(); err != nil {
		log.WriteString(string(out))
		return log.String(), fmt.Errorf("go build: %s", strings.TrimSpace(string(out)))
	}
	fmt.Fprintf(&log, "Built binary at %s\n", binPath)

	// Resolve destination in ~/.local/bin.
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return log.String(), fmt.Errorf("get home directory: %w", err)
	}
	dst := filepath.Join(homeDir, ".local", "bin", "bs3")

	if err := copyFile(binPath, dst); err != nil {
		return log.String(), fmt.Errorf("copy to %s: %w", dst, err)
	}
	fmt.Fprintf(&log, "Copied to %s\n", dst)

	return log.String(), nil
}

// cliCrossCompileBuild builds bs3 for linux/amd64 and linux/arm64 and zips
// each binary into cli-tool/.builds/. It accumulates a short progress log and
// returns it alongside any error.
func cliCrossCompileBuild(repoRoot string) (string, error) {
	var log strings.Builder
	platforms := [][2]string{
		{"linux", "amd64"},
		{"linux", "arm64"},
	}

	cliDir := filepath.Join(repoRoot, "cli-tool")
	outDir := filepath.Join(cliDir, ".builds")

	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return log.String(), fmt.Errorf("create .builds dir: %w", err)
	}
	fmt.Fprintf(&log, "Created %s\n", outDir)

	for _, p := range platforms {
		goos, goarch := p[0], p[1]
		if err := buildAndZip(cliDir, outDir, goos, goarch); err != nil {
			return log.String(), fmt.Errorf("%s/%s: %w", goos, goarch, err)
		}
		fmt.Fprintf(&log, "Built and zipped %s/%s → %s_%s.zip\n", goos, goarch, goos, goarch)
	}

	return log.String(), nil
}

// buildAndZip compiles the CLI binary for the given GOOS/GOARCH and zips it
// into outDir. The binary is removed after a successful zip.
func buildAndZip(cliDir, outDir, goos, goarch string) error {
	binName := "bs3"
	binPath := filepath.Join(cliDir, binName)
	zipPath := filepath.Join(outDir, fmt.Sprintf("%s_%s.zip", goos, goarch))

	// Cross-compile: inject GOOS/GOARCH into the environment.
	// GOWORK=off is already set by goCmd; append platform vars after.
	cmd := goCmd(cliDir, "build", "-o", binName)
	cmd.Env = append(cmd.Env, "GOOS="+goos, "GOARCH="+goarch)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("go build: %s", strings.TrimSpace(string(out)))
	}

	// Always remove the raw binary after this function returns.
	defer os.Remove(binPath)

	return zipFile(zipPath, binPath, binName)
}

// ─── Server Implementations ───────────────────────────────────────────────────

// serverComposeDown stops and removes the local test container started by
// "Server › Local Test", using server/compose/compose.dev.yml.
func serverComposeDown(repoRoot string) (string, error) {
	cmd := exec.Command("docker", "compose",
		"-f", "server/compose/compose.dev.yml", "down")
	cmd.Dir = repoRoot
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("docker compose down: %s", strings.TrimSpace(string(out)))
	}
	return string(out), nil
}

// serverCleanDev tears down the local test stack from
// server/compose/compose.dev.yml completely: the container, the default
// network, the bs3-data volume, and the image built from local source. This
// deletes any vault data in that volume.
func serverCleanDev(repoRoot string) (string, error) {
	cmd := exec.Command("docker", "compose",
		"-f", "server/compose/compose.dev.yml", "down",
		"--volumes", "--rmi", "local", "--remove-orphans")
	cmd.Dir = repoRoot
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("docker compose down: %s", strings.TrimSpace(string(out)))
	}
	return string(out), nil
}

// serverBuildBinary compiles the server into server/bs3-server. GOWORK=off is
// set by goCmd so the build uses server/go.mod, not the repo-root workspace.
func serverBuildBinary(repoRoot string) (string, error) {
	cmd := goCmd(filepath.Join(repoRoot, "server"), "build", "-o", "bs3-server", "./cmd")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("go build: %s", strings.TrimSpace(string(out)))
	}
	return string(out), nil
}

// serverDeployProd pulls the published image and starts it detached, using
// server/compose/compose.prod.yml — the production deployment step. Run it on
// the host that should serve the vault.
func serverDeployProd(repoRoot string) (string, error) {
	var log strings.Builder
	composeFile := filepath.Join("server", "compose", "compose.prod.yml")
	for _, args := range [][]string{
		{"compose", "-f", composeFile, "pull"},
		{"compose", "-f", composeFile, "up", "-d"},
	} {
		cmd := exec.Command("docker", args...)
		cmd.Dir = repoRoot
		out, err := cmd.CombinedOutput()
		log.WriteString(string(out))
		if err != nil {
			return log.String(), fmt.Errorf("docker %s: %s", strings.Join(args, " "), strings.TrimSpace(string(out)))
		}
	}
	return log.String(), nil
}

// ─── Logger Implementation ────────────────────────────────────────────────────

// loggerRunTests runs the logger's visual test program (go run ./cmd/) and
// captures its combined output so it can be shown in the dev hub's output
// modal. CLICOLOR_FORCE=1 is set so lipgloss still emits ANSI colors even
// though the command's stdout is a pipe rather than a real terminal.
func loggerRunTests(repoRoot string) (string, error) {
	cmd := goCmd(filepath.Join(repoRoot, "logger"), "run", "./cmd/")
	cmd.Env = append(cmd.Env, "CLICOLOR_FORCE=1")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("go run: %s", strings.TrimSpace(string(out)))
	}
	return string(out), nil
}

// ─── Release Implementation ───────────────────────────────────────────────────

// runReleaseScript executes a release script with the frozen contract
// `bash <repoRoot>/<script> <version> [--prerelease]`. --prerelease is passed
// when stable is false. It returns the combined output and any error.
func runReleaseScript(repoRoot, script, version string, stable bool) (string, error) {
	scriptPath := filepath.Join(repoRoot, filepath.FromSlash(script))
	args := []string{scriptPath, version}
	if !stable {
		args = append(args, "--prerelease")
	}
	cmd := exec.Command("bash", args...)
	cmd.Dir = repoRoot
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("bash %s: %s", script, strings.TrimSpace(string(out)))
	}
	return string(out), nil
}

// ─── Shared Utilities ─────────────────────────────────────────────────────────

// zipFile creates a zip archive at zipPath containing srcPath stored under
// nameInArchive. File mode bits from the source are preserved in the header.
func zipFile(zipPath, srcPath, nameInArchive string) error {
	zf, err := os.Create(zipPath)
	if err != nil {
		return fmt.Errorf("create zip file: %w", err)
	}
	defer zf.Close()

	w := zip.NewWriter(zf)
	defer w.Close()

	src, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("open source file: %w", err)
	}
	defer src.Close()

	// Use FileInfoHeader so the archive entry preserves executable permissions.
	info, err := src.Stat()
	if err != nil {
		return fmt.Errorf("stat source file: %w", err)
	}
	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return fmt.Errorf("create zip header: %w", err)
	}
	header.Name = nameInArchive
	header.Method = zip.Deflate

	fw, err := w.CreateHeader(header)
	if err != nil {
		return fmt.Errorf("create zip entry: %w", err)
	}

	if _, err = io.Copy(fw, src); err != nil {
		return fmt.Errorf("write zip entry: %w", err)
	}

	return nil
}

// copyFile copies src to dst, preserving the source file's permission bits.
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	info, err := in.Stat()
	if err != nil {
		return err
	}

	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, info.Mode())
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}
