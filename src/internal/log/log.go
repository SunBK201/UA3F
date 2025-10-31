package log

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/sunbk201/ua3f/internal/config"
	"gopkg.in/natefinch/lumberjack.v2"
)

const log_file = "/var/log/ua3f/ua3f.log"

type uctFormatter struct {
}

func (formatter *uctFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	var b *bytes.Buffer
	if entry.Buffer != nil {
		b = entry.Buffer
	} else {
		b = &bytes.Buffer{}
	}
	formatTime := entry.Time.Format("2006-01-02 15:04:05")
	b.WriteString(fmt.Sprintf("[%s][%s]: %s\n", formatTime, strings.ToUpper(entry.Level.String()), entry.Message))
	return b.Bytes(), nil
}

func SetLogConf(level string) {
	writer1 := &bytes.Buffer{}
	writer2 := os.Stdout
	writer3 := &lumberjack.Logger{
		Filename:   log_file,
		MaxSize:    1, // megabytes
		MaxBackups: 5,
		MaxAge:     7, //days
		LocalTime:  true,
		Compress:   true,
	}
	logrus.SetOutput(io.MultiWriter(writer1, writer2, writer3))
	formatter := &uctFormatter{}
	logrus.SetFormatter(formatter)
	switch level {
	case "trace":
		logrus.SetLevel(logrus.TraceLevel)
	case "debug":
		logrus.SetLevel(logrus.DebugLevel)
	case "info":
		logrus.SetLevel(logrus.InfoLevel)
	case "warn":
		logrus.SetLevel(logrus.WarnLevel)
	case "error":
		logrus.SetLevel(logrus.ErrorLevel)
	case "fatal":
		logrus.SetLevel(logrus.FatalLevel)
	case "panic":
		logrus.SetLevel(logrus.PanicLevel)
	default:
		logrus.SetLevel(logrus.InfoLevel)
	}
}

func LogHeader(version string, cfg *config.Config) {
	logrus.Info("UA3F v" + version)
	logrus.Info("Server Mode: " + cfg.ServerMode)
	logrus.Infof("Listen on %s", cfg.ListenAddr)
	logrus.Infof("User-Agent: %s", cfg.PayloadUA)
	logrus.Infof("User-Agent Regex Pattern: '%s'", cfg.UAPattern)
	logrus.Infof("Enable Partial Replace: %v", cfg.EnablePartialReplace)
	logrus.Infof("Log level: %s", cfg.LogLevel)
}

func LogDebugWithAddr(src string, dest string, msg string) {
	logrus.Debugf("[%s -> %s] %s", src, dest, msg)
}

func LogInfoWithAddr(src string, dest string, msg string) {
	logrus.Infof("[%s -> %s] %s", src, dest, msg)
}

func LogWarnWithAddr(src string, dest string, msg string) {
	logrus.Warnf("[%s -> %s] %s", src, dest, msg)
}

func LogErrorWithAddr(src string, dest string, msg string) {
	logrus.Errorf("[%s -> %s] %s", src, dest, msg)
}
