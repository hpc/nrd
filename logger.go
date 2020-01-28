package main

import (
	"log"
	"os"
)

// Logging
const logPrefix = ""

//const logFlags = log.Ldate | log.Ltime
const logFlags = 0

type logLevel uint8

// logging levels
const (
	ERROR logLevel = iota
	WARN
	INFO
	DEBUG
)

var logLevelStr = map[logLevel]string{
	ERROR: "ERROR",
	WARN:  "WARN",
	INFO:  "INFO",
	DEBUG: "DEBUG",
}

type logger struct {
	lvl logLevel
	f   *os.File
	l   *log.Logger
}

// NewLogger returns an initialized logger
func NewLogger(f *os.File, lvl logLevel) *logger {
	return &logger{
		lvl: lvl,
		f:   f,
		l:   log.New(f, logPrefix, logFlags),
	}
}

func (l *logger) Printf(lvl logLevel, fmt string, args ...interface{}) {
	if l.lvl >= lvl {
		l.l.Printf(logLevelStr[lvl]+":"+fmt+"\n", args...)
	}
}

func (l *logger) ERROR(fmt string, args ...interface{}) {
	l.Printf(ERROR, fmt, args...)
}

func (l *logger) WARN(fmt string, args ...interface{}) {
	l.Printf(WARN, fmt, args...)
}

func (l *logger) INFO(fmt string, args ...interface{}) {
	l.Printf(INFO, fmt, args...)
}

func (l *logger) DEBUG(fmt string, args ...interface{}) {
	l.Printf(DEBUG, fmt, args...)
}

func (l *logger) FATAL(fmt string, args ...interface{}) {
	l.ERROR(fmt, args...)
	os.Exit(1)
}
