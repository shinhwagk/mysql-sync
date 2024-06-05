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
	Level int
}

func (l Logger) Info(module string, args string) {
	if l.Level >= LevelInfo {
		log.Println(fmt.Sprintf(" -- INFO -- %s --", module), args)
	}
}

func (l Logger) Debug(module string, args string) {
	if l.Level >= LevelDebug {
		log.Println(fmt.Sprintf(" -- DEBUG -- %s --", module), args)
	}
}

func (l Logger) Error(module string, args string) {
	log.Println(fmt.Sprintf(" -- ERROR -- %s --", module), args)
}
