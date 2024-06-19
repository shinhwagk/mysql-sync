package main

import (
	"log"
	"os"
)

const (
	LevelInfo int = iota
	LevelDebug
)

type Logger struct {
	Level      int
	Module     string
	LogError   *log.Logger
	LogInfo    *log.Logger
	LogDebug   *log.Logger
	LogWarning *log.Logger
}

func NewLogger(level int, module string) *Logger {
	return &Logger{
		Level:      level,
		Module:     module,
		LogError:   log.New(os.Stderr, "ERROR", log.Ldate|log.Ltime|log.Llongfile),
		LogInfo:    log.New(os.Stdout, "INFO", log.Ldate|log.Ltime),
		LogDebug:   log.New(os.Stdout, "DEBUG", log.Ldate|log.Ltime|log.Llongfile),
		LogWarning: log.New(os.Stdout, "WARNING", log.Ldate|log.Ltime),
	}
}

func (l Logger) Info(format string, a ...any) {
	if l.Level >= LevelInfo {
		l.LogInfo.Printf("-- %s -- "+format+"\n", append([]any{l.Module}, a...)...)
	}
}

func (l Logger) Debug(format string, a ...any) {
	if l.Level >= LevelDebug {
		l.LogDebug.Printf("-- %s -- "+format+"\n", append([]any{l.Module}, a...)...)
	}
}

func (l Logger) Error(format string, a ...any) {
	l.LogError.Printf("-- %s -- "+format+"\n", append([]any{l.Module}, a...)...)
}

func (l Logger) Waring(format string, a ...any) {
	l.LogWarning.Printf("-- %s -- "+format+"\n", append([]any{l.Module}, a...)...)
}
