/*
Copyright (C) 2018 Daniel Morandini

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

// Package log implements a simple leveled logging interface.
package log

import (
	"io"
	"log"
	"os"
	"sync"
)

// Level represents the level of logging.
type Level int

// Different levels of logging.
const (
	DebugLevel Level = iota
	InfoLevel
	ErrorLevel
	DisabledLevel
)

// The set of default loggers for each log level.
var (
	Debug = &logger{DebugLevel}
	Info  = &logger{InfoLevel}
	Error = &logger{ErrorLevel}
)

type globalState struct {
	currentLevel  Level
	defaultLogger *log.Logger
}

type logger struct {
	level Level
}

var (
	mu    sync.RWMutex
	state = globalState{
		currentLevel:  InfoLevel,
		defaultLogger: newDefaultLogger(os.Stdout),
	}
)

func newDefaultLogger(w io.Writer) *log.Logger {
	return log.New(w, "", log.Ldate|log.Ltime|log.LUTC|log.Lmicroseconds)
}

func globals() globalState {
	mu.RLock()
	defer mu.RUnlock()
	return state
}

func SetLevel(l Level) {
	mu.RLock()
	state.currentLevel = l
	mu.RUnlock()
}

// Printf writes a formatted message to the log.
func Printf(format string, v ...interface{}) {
	Info.Printf(format, v...)
}

// Print writes a message to the log.
func Print(v ...interface{}) {
	Info.Print(v...)
}

// Println writes a line to the log.
func Println(v ...interface{}) {
	Info.Println(v...)
}

// Printf writes a formatted message to the log.
func (l *logger) Printf(format string, v ...interface{}) {
	g := globals()

	if l.level < g.currentLevel {
		return // Don't log at lower levels.
	}
	if g.defaultLogger != nil {
		g.defaultLogger.Printf(format, v...)
	}
}

// Print writes a message to the log.
func (l *logger) Print(v ...interface{}) {
	g := globals()

	if l.level < g.currentLevel {
		return // Don't log at lower levels.
	}
	if g.defaultLogger != nil {
		g.defaultLogger.Print(v...)
	}
}

// Println writes a line to the log.
func (l *logger) Println(v ...interface{}) {
	g := globals()

	if l.level < g.currentLevel {
		return // Don't log at lower levels.
	}
	if g.defaultLogger != nil {
		g.defaultLogger.Println(v...)
	}
}

// Fatalf writes a formatted message to the log and aborts, regardless of the
// current log level.
func (l *logger) Fatalf(format string, v ...interface{}) {
	g := globals()

	if g.defaultLogger != nil {
		g.defaultLogger.Fatalf(format, v...)
	} else {
		log.Fatalf(format, v...)
	}
}

// Fatalf writes a formatted message to the log and aborts.
func Fatalf(format string, v ...interface{}) {
	Info.Fatalf(format, v...)
}
