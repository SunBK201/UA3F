//go:build unix

package log

import (
	"log/slog"
	"os"
	"runtime"
	"strings"

	"golang.org/x/sys/unix"
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
	case "linux", "darwin", "freebsd", "openbsd", "netbsd":
		var uname unix.Utsname
		if err := unix.Uname(&uname); err == nil {
			toStr := func(b []byte) string {
				n := 0
				for ; n < len(b); n++ {
					if b[n] == 0 {
						break
					}
				}
				return strings.TrimSpace(string(b[:n]))
			}
			attrs = append(attrs,
				slog.String("sysname", toStr(uname.Sysname[:])),
				slog.String("nodename", toStr(uname.Nodename[:])),
				slog.String("release", toStr(uname.Release[:])),
				slog.String("version", toStr(uname.Version[:])),
				slog.String("machine", toStr(uname.Machine[:])),
			)
		}
	default:
		attrs = append(attrs, slog.String("info", "unknown OS details"))
	}
	return attrs
}
