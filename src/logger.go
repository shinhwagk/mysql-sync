package main

import (
	"fmt"
	"log"
)

const (
	LevelInfo int = iota
	LevelDebug
)

type Logger struct {
	Level  int
	Module string
}

func NewLogger(level int, module string) *Logger {
	return &Logger{Level: level, Module: module}
}

func (l Logger) Info(args string) {
	if l.Level >= LevelInfo {
		log.Println(fmt.Sprintf(" -- INFO  -- %s -- %s", l.Module, args))
	}
}

func (l Logger) Debug(args string) {
	if l.Level >= LevelDebug {
		log.Println(fmt.Sprintf(" -- DEBUG -- %s -- %s", l.Module, args))
	}
}

func (l Logger) Error(args string) {
	log.Println(fmt.Sprintf(" -- ERROR -- %s -- %s", l.Module, args))
}

func (l Logger) Waring(args string) {
	log.Println(fmt.Sprintf(" -- WARNING -- %s -- %s", l.Module, args))
}
