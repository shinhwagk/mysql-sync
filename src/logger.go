package main

import (
	"fmt"
	"log"
	"os"
	"time"
)

const (
	LevelInfo int = iota
	LevelDebug
	LevelTrace
)

type Logger struct {
	Level      int
	Module     string
	LogError   *log.Logger
	LogInfo    *log.Logger
	LogDebug   *log.Logger
	LogWarning *log.Logger
	LogTrace   *log.Logger
	LogAudit   *log.Logger
}

func NewLogger(level int, module string) *Logger {
	return &Logger{
		Level:      level,
		Module:     module,
		LogError:   log.New(os.Stderr, "", 0),
		LogInfo:    log.New(os.Stdout, "", 0),
		LogDebug:   log.New(os.Stdout, "", 0),
		LogWarning: log.New(os.Stdout, "", 0),
		LogAudit:   log.New(os.Stdout, "", 0),
	}
}

func (l *Logger) Info(format string, a ...interface{}) {
	if l.Level >= LevelInfo {
		l.output(l.LogInfo, "INFO", format, a...)
	}
}

func (l *Logger) Debug(format string, a ...interface{}) {
	if l.Level >= LevelDebug {
		l.output(l.LogDebug, "DEBUG", format, a...)
	}
}

func (l *Logger) Error(format string, a ...interface{}) {
	l.output(l.LogError, "ERROR", format, a...)
}

func (l *Logger) Warning(format string, a ...interface{}) {
	l.output(l.LogWarning, "WARNING", format, a...)
}

func (l *Logger) Trace(format string, a ...interface{}) {
	if l.Level >= LevelTrace {
		l.output(l.LogWarning, "TRACE", format, a...)
	}
}

func (l *Logger) Audit(format string, a ...interface{}) {
	if l.Level >= LevelTrace {
		l.output(l.LogWarning, "AUDIT", format, a...)
	}
}

func (l *Logger) output(logger *log.Logger, level string, format string, a ...interface{}) {
	fullFormat := "%s -- %s -- %s -- " + format
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	args := append([]interface{}{timestamp, level, l.Module}, a...)
	msg := fmt.Sprintf(fullFormat, args...)
	logger.Output(2, msg)
}
