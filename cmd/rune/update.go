package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"time"

	"github.com/siddhartha/rune/internal/ui"
	"github.com/siddhartha/rune/internal/update"
)

// version is stamped at release time via -ldflags "-X main.version=...".
var version = "dev"

const brewFormula = "sidkhuntia/tap/rune"

// currentVersion prefers the stamped version, then the module version
// recorded by `go install`, then "dev".
func currentVersion() string {
	if version != "dev" {
		return version
	}
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return version
}

// runUpdate brings the installed binary up to the latest release.
func runUpdate() error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	current := currentVersion()
	client := update.NewClient()

	rel, err := client.Latest(ctx)
	if err != nil {
		return err
	}
	if !update.IsNewer(current, rel.Tag) {
		ui.Success(fmt.Sprintf("rune %s is already up to date", current))
		return nil
	}
	ui.Info(fmt.Sprintf("Updating rune %s → %s", current, rel.Tag))

	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("cannot locate running binary: %w", err)
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}

	// Let Homebrew own binaries it installed so its records stay accurate.
	if update.IsHomebrewPath(exe) {
		ui.Info("Installed via Homebrew; running brew upgrade")
		cmd := exec.CommandContext(ctx, "brew", "upgrade", brewFormula)
		cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("brew upgrade failed: %w", err)
		}
		return nil
	}

	if err := client.Apply(ctx, rel, exe, runtime.GOOS, runtime.GOARCH); err != nil {
		return err
	}
	ui.Success(fmt.Sprintf("Updated to %s", rel.Tag))
	return nil
}
