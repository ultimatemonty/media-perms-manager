package logger

import (
    "io"
    "log"
    "os"
    "strings"
)

type level int

const (
    levelDebug level = iota
    levelInfo
    levelWarn
    levelError
)

type Logger struct {
    l     *log.Logger
    level level
}

func New(levelStr string) *Logger {
    l := log.New(os.Stdout, "", log.LstdFlags)
    var lvl level
    switch strings.ToLower(levelStr) {
    case "debug":
        lvl = levelDebug
    case "warn":
        lvl = levelWarn
    case "error":
        lvl = levelError
    default:
        lvl = levelInfo
    }
    return &Logger{l: l, level: lvl}
}

func (lg *Logger) SetOutput(w io.Writer) {
    lg.l.SetOutput(w)
}

func (lg *Logger) Debug(v ...interface{}) { if lg.level <= levelDebug { lg.l.SetPrefix("DEBUG: "); lg.l.Println(v...) } }
func (lg *Logger) Info(v ...interface{})  { if lg.level <= levelInfo  { lg.l.SetPrefix("INFO: "); lg.l.Println(v...) } }
func (lg *Logger) Warn(v ...interface{})  { if lg.level <= levelWarn  { lg.l.SetPrefix("WARN: "); lg.l.Println(v...) } }
func (lg *Logger) Error(v ...interface{}) { if lg.level <= levelError { lg.l.SetPrefix("ERROR: "); lg.l.Println(v...) } }

// Printf helpers
func (lg *Logger) Debugf(format string, v ...interface{}) { if lg.level <= levelDebug { lg.l.SetPrefix("DEBUG: "); lg.l.Printf(format, v...) } }
func (lg *Logger) Infof(format string, v ...interface{})  { if lg.level <= levelInfo  { lg.l.SetPrefix("INFO: "); lg.l.Printf(format, v...) } }
func (lg *Logger) Warnf(format string, v ...interface{})  { if lg.level <= levelWarn  { lg.l.SetPrefix("WARN: "); lg.l.Printf(format, v...) } }
func (lg *Logger) Errorf(format string, v ...interface{}) { if lg.level <= levelError { lg.l.SetPrefix("ERROR: "); lg.l.Printf(format, v...) } }
