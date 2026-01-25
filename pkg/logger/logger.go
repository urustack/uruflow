package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

type Level int

const (
	DEBUG Level = iota
	INFO
	WARN
	ERROR
)

var levelNames = map[Level]string{
	DEBUG: "DEBUG",
	INFO:  "INFO",
	WARN:  "WARN",
	ERROR: "ERROR",
}

type Logger struct {
	level  Level
	output io.Writer
	prefix string
}

var std *Logger

func Init(logPath string, level string) error {
	logLevel := INFO
	switch level {
	case "debug":
		logLevel = DEBUG
	case "warn":
		logLevel = WARN
	case "error":
		logLevel = ERROR
	}

	if logPath == "" {
		std = &Logger{
			level:  logLevel,
			output: os.Stdout,
			prefix: "[URUFLOW] ",
		}
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(logPath), 0755); err != nil {
		return fmt.Errorf("create log directory: %w", err)
	}

	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("open log file: %w", err)
	}

	std = &Logger{
		level:  logLevel,
		output: io.MultiWriter(os.Stdout, file),
		prefix: "[URUFLOW] ",
	}

	return nil
}

func (l *Logger) log(level Level, format string, args ...interface{}) {
	if level < l.level {
		return
	}

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	levelStr := levelNames[level]
	message := fmt.Sprintf(format, args...)

	fmt.Fprintf(l.output, "%s %s %s%s\n", timestamp, levelStr, l.prefix, message)
}

func Debug(format string, args ...interface{}) {
	if std != nil {
		std.log(DEBUG, format, args...)
	}
}

func Info(format string, args ...interface{}) {
	if std != nil {
		std.log(INFO, format, args...)
	}
}

func Warn(format string, args ...interface{}) {
	if std != nil {
		std.log(WARN, format, args...)
	}
}

func Error(format string, args ...interface{}) {
	if std != nil {
		std.log(ERROR, format, args...)
	}
}

func With(prefix string) *Logger {
	if std == nil {
		return nil
	}
	return &Logger{
		level:  std.level,
		output: std.output,
		prefix: fmt.Sprintf("%s[%s] ", std.prefix, prefix),
	}
}
