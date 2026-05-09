//go:build !unix

package log

import (
	"log/slog"
	"os"
	"runtime"
)

func GetOSInfo() (attrs []any) {

	attrs = append(attrs,
		slog.String("GOOS", runtime.GOOS),
		slog.String("GOARCH", runtime.GOARCH),
		slog.String("Go Version", runtime.Version()),
	)

	if hostname, err := os.Hostname(); err == nil {
		attrs = append(attrs, slog.String("hostname", hostname))
	}

	switch runtime.GOOS {
	case "windows":
		osver := "unknown"
		if v, ok := os.LookupEnv("OS"); ok {
			osver = v
		}
		attrs = append(attrs, slog.String("os_version", osver))
	default:
		attrs = append(attrs, slog.String("info", "unknown OS details"))
	}
	return attrs
}
