package log

import (
	"os"
	"path/filepath"
	"runtime"
)

var (
	logDir     string
	logDirOnce bool
)

// GetLogDir returns the platform-specific log directory for UA3F.
// - Linux: /var/log/ua3f/
// - Windows: ~/.ua3f/
// - Other (fallback): temp directory
// The directory is created automatically with proper permissions if it doesn't exist.
func GetLogDir() string {
	if logDirOnce {
		return logDir
	}

	logDir = determineLogDir()
	logDirOnce = true

	// Ensure the directory exists
	if err := os.MkdirAll(logDir, 0755); err != nil {
		// If creation fails, fall back to temp directory
		logDir = filepath.Join(os.TempDir(), "ua3f")
		_ = os.MkdirAll(logDir, 0755)
	}

	return logDir
}

func determineLogDir() string {
	switch runtime.GOOS {
	case "linux":
		// Try /var/log/ua3f/ first for Linux
		varLogDir := "/var/log/ua3f"
		if err := os.MkdirAll(varLogDir, 0755); err == nil {
			// Test write permission
			testFile := filepath.Join(varLogDir, ".write_test")
			if f, err := os.Create(testFile); err == nil {
				_ = f.Close()
				_ = os.Remove(testFile)
				return varLogDir
			}
		}
		// Fall through to user home directory
		return getUserLogDir()
	default:
		// macOS, Windows, FreeBSD, etc. - try user home first
		return getUserLogDir()
	}
}

func getUserLogDir() string {
	// Try user home directory
	homeDir, err := os.UserHomeDir()
	if err == nil {
		userLogDir := filepath.Join(homeDir, ".ua3f")
		if err := os.MkdirAll(userLogDir, 0755); err == nil {
			return userLogDir
		}
	}

	// Fallback to temp directory
	return filepath.Join(os.TempDir(), "ua3f")
}

// GetLogFilePath returns the full path to the main log file.
func GetLogFilePath() string {
	return filepath.Join(GetLogDir(), "ua3f.log")
}

// GetStatsFilePath returns the full path to a stats file.
func GetStatsFilePath(name string) string {
	return filepath.Join(GetLogDir(), name)
}
