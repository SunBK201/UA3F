package daemon

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"os/user"
	"strings"

	"github.com/sunbk201/ua3f/internal/config"
)

func DaemonSetup(cfg *config.Config) error {
	if IsOpenWrt() {
		if err := SetOOMScoreAdj(-900); err != nil {
			slog.Warn("SetOOMScoreAdj", slog.Any("error", err))
		}
	}
	if err := SetUserGroup(cfg); err != nil {
		return fmt.Errorf("SetUserGroup: %w", err)
	}
	return nil
}

func IsOpenWrt() bool {
	checkFiles := []string{
		"/etc/openwrt_release",
	}
	for _, f := range checkFiles {
		if _, err := os.Stat(f); err == nil {
			return true
		}
	}

	data, err := os.ReadFile("/etc/os-release")
	if err == nil && strings.Contains(string(data), "OpenWrt") {
		return true
	}

	if _, err := user.Lookup("uci"); err == nil {
		return true
	}

	if _, err := exec.LookPath("opkg"); err == nil {
		return true
	}

	if _, err := exec.LookPath("apk"); err == nil {
		return true
	}

	return false
}

func IsCommandAvailable(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

func IsCommandRunning(c string) bool {
	cmd := exec.Command("pgrep", "-f", c)
	err := cmd.Run()
	return err == nil
}

func IsPackageInstalled(pkg string) bool {
	if _, err := exec.LookPath("opkg"); err == nil {
		cmd := exec.Command("opkg", "list-installed", pkg)
		output, err := cmd.Output()
		if err != nil {
			return false
		}
		return strings.Contains(string(output), pkg)
	}

	if _, err := exec.LookPath("apk"); err == nil {
		cmd := exec.Command("apk", "info", "-e", pkg)
		output, err := cmd.Output()
		if err != nil {
			return false
		}
		return strings.Contains(string(output), pkg)
	}

	return false
}
