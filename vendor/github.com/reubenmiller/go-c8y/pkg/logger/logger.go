package logger

import (
	"io"

	"github.com/op/go-logging"
)

// Logging logger interface which enable the user to provide their own logger for individual requests if necessary
type Logger interface {
	Debugf(format string, args ...interface{})
	Debug(args ...interface{})

	Infof(format string, args ...interface{})
	Info(args ...interface{})

	Warnf(format string, args ...interface{})
	Warn(args ...interface{})

	Errorf(format string, args ...interface{})
	Error(args ...interface{})

	Fatalf(format string, args ...interface{})
	Fatal(args ...interface{})
}

type NopBackend struct {
}

func (b *NopBackend) Log(level logging.Level, calldepth int, rec *logging.Record) error {
	// noop
	return nil
}

func (b *NopBackend) GetLevel(val string) logging.Level {
	return 0
}

func (b *NopBackend) SetLevel(level logging.Level, val string) {}
func (b *NopBackend) IsEnabledFor(level logging.Level, val string) bool {
	return false
}

type Log struct {
	*logging.Logger
}

func (l Log) Printf(format string, args ...interface{}) {
	l.Infof(format, args...)
}

func (l Log) Println(args ...interface{}) {
	l.Info(args...)
}

func (l Log) Warnf(format string, args ...interface{}) {
	l.Logger.Warningf(format, args...)
}

func (l Log) Warn(args ...interface{}) {
	l.Logger.Warning(args...)
}

func NewDummyLogger(name string) *Log {
	gologger := logging.MustGetLogger(name)
	gologger.SetBackend(&NopBackend{})
	return &Log{
		Logger: gologger,
	}
}

func NewLogger(name string) Logger {
	gologger := logging.MustGetLogger(name)
	return &Log{
		Logger: gologger,
	}
}

// NewCustomLogger returns a new custom logger writing it to a given writer
func NewCustomLogger(name string, w io.Writer, level int, customFormatter ...logging.Formatter) *Log {
	gologger := logging.MustGetLogger(name)
	backend := logging.NewLogBackend(w, "", 0)

	if level < 0 {
		level = int(logging.INFO)
	}

	logFormat := logging.DefaultFormatter
	if len(customFormatter) > 0 {
		logFormat = customFormatter[0]
	}

	backend1Leveled := logging.AddModuleLevel(logging.NewBackendFormatter(backend, logFormat))

	backend1Leveled.SetLevel(logging.Level(level), "")
	gologger.SetBackend(backend1Leveled)

	return &Log{
		Logger: gologger,
	}
}
