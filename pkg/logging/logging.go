package logging

import (
	"log/slog"

	"github.com/go-logr/logr"
)

// A Logger logs messages. Messages may be supplemented by structured data.
type Logger interface {
	// Info logs a message with optional structured data. Structured data must
	// be supplied as an array that alternates between string keys and values of
	// an arbitrary type. Use Info for messages that operators are
	// very likely to be concerned with when running.
	Info(msg string, keysAndValues ...any)

	// Debug logs a message with optional structured data. Structured data must
	// be supplied as an array that alternates between string keys and values of
	// an arbitrary type. Use Debug for messages that operators or
	// developers may be concerned with when debugging.
	Debug(msg string, keysAndValues ...any)

	// Error logs an error with a message and optional structured data.
	// Structured data must be supplied as an array that alternates between
	// string keys and values of an arbitrary type. Use Error for messages
	Error(err error, msg string, keysAndValues ...any)

	// Warn logs a message with optional structured data. Structured data must
	// be supplied as an array that alternates between string keys and values of
	// an arbitrary type. Use Warn for messages that operators should be aware of.
	Warn(msg string, keysAndValues ...any)

	// WithValues returns a Logger that will include the supplied structured
	// data with any subsequent messages it logs. Structured data must
	// be supplied as an array that alternates between string keys and values of
	// an arbitrary type.
	WithValues(keysAndValues ...any) Logger

	// WithName returns a Logger that will include the supplied name with any
	// subsequent messages it logs. The name is typically a component name or
	// similar, and should be used to distinguish between different components
	// or subsystems.
	WithName(name string) Logger
}

// NewNopLogger returns a Logger that does nothing.
func NewNopLogger() Logger { return nopLogger{} }

type nopLogger struct{}

func (l nopLogger) Info(msg string, keysAndValues ...any)             {}
func (l nopLogger) Debug(msg string, keysAndValues ...any)            {}
func (l nopLogger) Error(err error, msg string, keysAndValues ...any) {}
func (l nopLogger) Warn(msg string, keysAndValues ...any)             {}
func (l nopLogger) WithName(name string) Logger                       { return nopLogger{} }
func (l nopLogger) WithValues(keysAndValues ...any) Logger            { return nopLogger{} }

// NewLogrLogger returns a Logger that is satisfied by the supplied logr.Logger,
// which may be satisfied in turn by various logging implementations (Zap, klog,
// etc). Debug messages are logged at V(1).
func NewLogrLogger(l logr.Logger) Logger {
	return logrLogger{log: l}
}

type logrLogger struct {
	log logr.Logger
}

func (l logrLogger) Info(msg string, keysAndValues ...any) {
	l.log.Info(msg, keysAndValues...) //nolint:logrlint // False positive - logrlint thinks there's an odd number of args.
}

func (l logrLogger) Debug(msg string, keysAndValues ...any) {
	l.log.V(1).Info(msg, keysAndValues...) //nolint:logrlint // False positive - logrlint thinks there's an odd number of args.
}

func (l logrLogger) Error(err error, msg string, keysAndValues ...any) {
	l.log.Error(err, msg, keysAndValues...) //nolint:logrlint // False positive - logrlint thinks there's an odd number of args.
}

func (l logrLogger) Warn(msg string, keysAndValues ...any) {
	l.log.V(0).Info(msg, keysAndValues...) //nolint:logrlint // False positive - logrlint thinks there's an odd number of args.
}

func (l logrLogger) WithName(name string) Logger {
	return logrLogger{log: l.log.WithName(name)}
}

func (l logrLogger) WithValues(keysAndValues ...any) Logger {
	return logrLogger{log: l.log.WithValues(keysAndValues...)} //nolint:logrlint // False positive - logrlint thinks there's an odd number of args.
}

type slogLogger struct {
	log *slog.Logger
}

func NewSlogLogger(l slog.Logger) Logger {
	return slogLogger{log: &l}
}

func (l slogLogger) Info(msg string, keysAndValues ...any) {
	l.log.Info(msg, keysAndValues...)
}
func (l slogLogger) Debug(msg string, keysAndValues ...any) {
	l.log.Debug(msg, keysAndValues...)
}
func (l slogLogger) Warn(msg string, keysAndValues ...any) {
	l.log.Warn(msg, keysAndValues...)
}
func (l slogLogger) Error(err error, msg string, keysAndValues ...any) {
	l.log.Error(msg, append(keysAndValues, slog.Any("err", err))...)
}
func (l slogLogger) WithName(name string) Logger {
	return slogLogger{log: l.log.With(slog.String("logger", name))}
}
func (l slogLogger) WithValues(keysAndValues ...any) Logger {
	return slogLogger{log: l.log.With(keysAndValues...)}
}
