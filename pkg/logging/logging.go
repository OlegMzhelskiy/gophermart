package logging

import (
	"github.com/sirupsen/logrus"
	"os"
)

type Level uint32

const (
	// FatalLevel level. Logs and then calls `logger.Exit(1)`. It will exit even if the
	// logging level is set to Panic.
	FatalLevel Level = iota
	// ErrorLevel level. Logs. Used for errors that should definitely be noted.
	// Commonly used for hooks to send errors to an error tracking service.
	ErrorLevel
	// InfoLevel level. General operational entries about what's going on inside the
	// application.
	InfoLevel
	// DebugLevel level. Usually only enabled when debugging. Very verbose logging.
	DebugLevel
)

type Fielders interface{}

type Fields map[string]interface{}

type Loggerer interface {
	Info(message string, args ...interface{})
	Debug(message interface{}, args ...interface{})
	Error(message interface{}, args ...interface{})
	Fatal(message interface{}, args ...interface{})
	LogWithFields(level Level, message string, fields Fields)
	//LogWithFields(level Level, message string, fields Fielders)
}

type Logger struct {
	logger *logrus.Logger
}

func NewLogger(production bool) Loggerer {
	log := logrus.New()
	log.Formatter = new(logrus.TextFormatter) //default
	if production {
		log.Level = logrus.ErrorLevel
	} else {
		log.Level = logrus.DebugLevel
		log.Formatter.(*logrus.TextFormatter).DisableTimestamp = true // remove timestamp from test output
	}
	log.Formatter.(*logrus.TextFormatter).DisableColors = false // remove colors
	log.Out = os.Stdout
	return &Logger{logger: log}
}

func (l *Logger) LogWithFields(level Level, message string, fields Fields) {
	fld := logrus.Fields(fields)
	ent := l.logger.WithFields(fld)
	if level == FatalLevel {
		ent.Fatal(message)
	} else if level == ErrorLevel {
		ent.Error(message)
	} else if level == DebugLevel {
		ent.Debug(message)
	} else if level == InfoLevel {
		ent.Info(message)
	}
}

func (l *Logger) Info(message string, args ...interface{}) {
	a := []interface{}{message}
	a = append(a, args...)
	l.logger.Info(a...)
}

func (l *Logger) Debug(message interface{}, args ...interface{}) {
	a := []interface{}{message}
	a = append(a, args...)
	l.logger.Debug(a...)
}

func (l *Logger) Error(message interface{}, args ...interface{}) {
	a := ConErrorArgs(message, args)
	l.logger.Error(a...)
}

func (l *Logger) Fatal(message interface{}, args ...interface{}) {
	a := ConErrorArgs(message, args)
	l.logger.Fatal(a...)
}

func ConErrorArgs(message interface{}, args ...interface{}) []interface{} {
	var mes interface{}
	a := make([]interface{}, 0, len(args)+1)
	err, ok := message.(error)
	if ok {
		mes = err.Error()
	} else {
		mes = message
	}
	a = append(a, mes)
	a = append(a, args...)
	return a
}
