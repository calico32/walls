package main

import (
	"fmt"
	"os"

	"github.com/fatih/color"
)

// logLevel is the level of logging to do. The higher the number, the more
// verbose the logging will be.
type logLevel int

const (
	LogLevelFatal logLevel = iota
	LogLevelError
	LogLevelWarn
	LogLevelInfo
	LogLevelDebug
)

type Logger struct {
	Level logLevel
}

var logger Logger

func (l *Logger) Logf(level logLevel, format string, args ...any) {
	if level > l.Level {
		return
	}
	fmt.Fprintf(os.Stderr, format+"\n", args...)
}

func (l *Logger) Fatal(args ...any) {
	red := color.New(color.FgRed, color.Bold)
	l.Logf(LogLevelFatal, red.Sprint("fatal")+": %s", fmt.Sprint(args...))
	os.Exit(1)
}

func (l *Logger) Fatalf(format string, args ...any) {
	red := color.New(color.FgRed, color.Bold)
	l.Logf(LogLevelFatal, red.Sprint("fatal")+": %s", fmt.Errorf(format, args...))
	os.Exit(1)
}

func (l *Logger) Errorf(format string, args ...any) {
	red := color.New(color.FgRed)
	l.Logf(LogLevelError, red.Sprint("error")+": %s", fmt.Errorf(format, args...))
}

func (l *Logger) Warnf(format string, args ...any) {
	yellow := color.New(color.FgYellow)
	l.Logf(LogLevelWarn, yellow.Sprint(" warn")+": "+format, args...)
}

func (l *Logger) Infof(format string, args ...any) {
	blue := color.New(color.FgBlue)
	l.Logf(LogLevelInfo, blue.Sprint(" info")+": "+format, args...)
}

func (l *Logger) Debugf(format string, args ...any) {
	cyan := color.New(color.FgCyan)
	l.Logf(LogLevelDebug, cyan.Sprint("debug")+": "+format, args...)
}
