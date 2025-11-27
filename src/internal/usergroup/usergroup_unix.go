//go:build linux

package usergroup

import (
	"bufio"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"

	"github.com/sunbk201/ua3f/internal/config"
)

func SetUserGroup(cfg *config.Config) error {
	groupName := determineGroup(cfg.ServerMode)
	if groupName == "" {
		return nil
	}

	gid, err := getGroupID(groupName)
	if err != nil {
		slog.Warn("getGroupID", slog.String("group", groupName), slog.Any("error", err))
		return nil
	}

	if err := syscall.Setgid(gid); err != nil {
		if err == syscall.EPERM {
			slog.Warn("syscall.Setgid", slog.String("group", groupName), slog.Int("gid", gid), slog.Any("error", err))
			return nil
		}
		return fmt.Errorf("syscall.Setgid: %w", err)
	}

	slog.Info("Setup user group", slog.String("group", groupName), slog.Int("gid", gid))
	return nil
}

func determineGroup(serverMode config.ServerMode) string {
	if serverMode == config.ServerModeRedirect || serverMode == config.ServerModeNFQueue {
		return "root"
	}

	if processRunning("openclash") {
		return "nogroup"
	}

	if processRunning("ShellCrash") {
		return "shellcrash"
	}

	if openClashExists() {
		return "nogroup"
	}

	if shellClashExists() {
		return "shellcrash"
	}

	return "nogroup"
}

func openClashExists() bool {
	if !opkgAvailable() {
		return false
	}

	cmd := exec.Command("opkg", "list-installed", "luci-app-openclash")
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	return strings.Contains(string(output), "luci-app-openclash")
}

func shellClashExists() bool {
	if userExists("shellclash") {
		return true
	}
	if userExists("shellcrash") {
		return true
	}
	return false
}

func opkgAvailable() bool {
	_, err := exec.LookPath("opkg")
	return err == nil
}

func processRunning(name string) bool {
	cmd := exec.Command("pgrep", "-f", name)
	err := cmd.Run()
	return err == nil
}

func userExists(username string) bool {
	cmd := exec.Command("id", "-u", username)
	err := cmd.Run()
	return err == nil
}

func getGroupID(groupName string) (int, error) {
	if groupName == "root" {
		return 0, nil
	}

	file, err := os.Open("/etc/group")
	if err != nil {
		return 0, fmt.Errorf("failed to open /etc/group: %w", err)
	}
	defer func() {
		_ = file.Close()
	}()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Split(line, ":")
		if len(parts) >= 3 && parts[0] == groupName {
			gid, err := strconv.Atoi(parts[2])
			if err != nil {
				return 0, fmt.Errorf("failed to parse GID for group %s: %w", groupName, err)
			}
			return gid, nil
		}
	}

	if err := scanner.Err(); err != nil {
		return 0, fmt.Errorf("error reading /etc/group: %w", err)
	}

	return 0, fmt.Errorf("group %s not found", groupName)
}
