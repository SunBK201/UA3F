package log

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
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
		MaxSize:    2, // megabytes
		MaxBackups: 3,
		MaxAge:     28, //days
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
