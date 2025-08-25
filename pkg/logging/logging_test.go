package logging

import (
	"bytes"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/go-logr/logr"
	"github.com/go-logr/logr/testr"
	prettylog "github.com/krateoplatformops/plumbing/slogs/pretty"
)

func TestNewNopLogger(t *testing.T) {
	logger := NewNopLogger()
	if logger == nil {
		t.Error("NewNopLogger returned nil")
	}

	// Test that all methods can be called without panicking
	logger.Info("test message", "key", "value")
	logger.Debug("test message", "key", "value")
	logger.Error(errors.New("test error"), "test message", "key", "value")
	logger.Warn("test message", "key", "value")

	withName := logger.WithName("test")
	if withName == nil {
		t.Error("WithName returned nil")
	}

	withValues := logger.WithValues("key", "value")
	if withValues == nil {
		t.Error("WithValues returned nil")
	}
}

func TestNewLogrLogger(t *testing.T) {
	testLogger := testr.New(t)
	logger := NewLogrLogger(testLogger)

	if logger == nil {
		t.Error("NewLogrLogger returned nil")
	}

	// Test logging methods
	logger.Info("info message", "key", "value")
	logger.Debug("debug message", "key", "value")
	logger.Error(errors.New("test error"), "error message", "key", "value")
	logger.Warn("warn message", "key", "value")

	// Test WithName
	namedLogger := logger.WithName("component")
	if namedLogger == nil {
		t.Error("WithName returned nil")
	}
	namedLogger.Info("named logger message")

	// Test WithValues
	valuedLogger := logger.WithValues("component", "test")
	if valuedLogger == nil {
		t.Error("WithValues returned nil")
	}
	valuedLogger.Info("valued logger message")
}

func TestNewSlogLogger(t *testing.T) {
	var buf bytes.Buffer
	slogger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	logger := NewSlogLogger(*slogger)
	if logger == nil {
		t.Error("NewSlogLogger returned nil")
	}

	// Test Info
	logger.Info("info message", "key", "value")
	if !bytes.Contains(buf.Bytes(), []byte("info message")) {
		t.Error("Info message not logged")
	}
	buf.Reset()

	// Test Debug
	logger.Debug("debug message", "key", "value")
	if !bytes.Contains(buf.Bytes(), []byte("debug message")) {
		t.Error("Debug message not logged")
	}
	buf.Reset()

	// Test Warn
	logger.Warn("warn message", "key", "value")
	if !bytes.Contains(buf.Bytes(), []byte("warn message")) {
		t.Error("Warn message not logged")
	}
	buf.Reset()

	// Test Error
	testErr := errors.New("test error")
	logger.Error(testErr, "error message", "key", "value")
	logOutput := buf.String()
	if !bytes.Contains(buf.Bytes(), []byte("error message")) {
		t.Error("Error message not logged")
	}
	if !bytes.Contains([]byte(logOutput), []byte("test error")) {
		t.Error("Error not included in log output")
	}
	buf.Reset()

	// Test WithName
	namedLogger := logger.WithName("component")
	if namedLogger == nil {
		t.Error("WithName returned nil")
	}
	namedLogger.Info("named message")
	if !bytes.Contains(buf.Bytes(), []byte("logger=component")) {
		t.Error("Logger name not included")
	}
	buf.Reset()

	// Test WithValues
	valuedLogger := logger.WithValues("component", "test")
	if valuedLogger == nil {
		t.Error("WithValues returned nil")
	}
	valuedLogger.Info("valued message")
	if !bytes.Contains(buf.Bytes(), []byte("component=test")) {
		t.Error("Values not included")
	}
}

func TestLoggerInterface(t *testing.T) {
	// Test that all implementations satisfy the Logger interface
	var _ Logger = NewNopLogger()
	var _ Logger = NewLogrLogger(logr.Discard())
	var _ Logger = NewSlogLogger(*slog.Default())
}

func parseLogTokens(s string) map[string]string {
	result := make(map[string]string)
	// match key=value or key="value with spaces"
	re := regexp.MustCompile(`(\w+)=(".*?"|\S+)`)
	for _, m := range re.FindAllStringSubmatch(s, -1) {
		k := m[1]
		v := m[2]
		v = strings.Trim(v, `"`)
		result[k] = v
	}
	// If there's a plain message without msg=, try to capture last quoted/unquoted text as msg
	if _, ok := result["msg"]; !ok {
		// heuristics: take substring after last "level=... "
		if idx := strings.LastIndex(s, "msg="); idx != -1 {
			rest := s[idx+4:]
			rest = strings.TrimSpace(strings.Trim(rest, `"`))
			result["msg"] = rest
		}
	}
	return result
}

func TestSlogAndLogrOutputsMatch(t *testing.T) {
	// prepare slog -> buf1
	var buf1 bytes.Buffer

	lh1 := prettylog.New(&slog.HandlerOptions{
		Level:     slog.LevelDebug,
		AddSource: false,
	},
		prettylog.WithColor(),
		prettylog.WithOutputEmptyAttrs(),
		prettylog.WithDestinationWriter(&buf1),
	)

	slogger := slog.New(lh1)
	sl := NewSlogLogger(*slogger)

	// prepare logr (stdr) -> buf2
	var buf2 bytes.Buffer
	lh2 := prettylog.New(&slog.HandlerOptions{
		Level:     slog.LevelDebug,
		AddSource: false,
	},
		prettylog.WithDestinationWriter(os.Stderr),
		prettylog.WithColor(),
		prettylog.WithOutputEmptyAttrs(),
		prettylog.WithDestinationWriter(&buf2),
	)
	ll := NewLogrLogger(logr.FromSlogHandler(lh2))

	cases := []struct {
		name string
		call func(l Logger)
	}{
		{"info", func(l Logger) { l.Info("hello", "k1", "v1", "k2", "v2") }},
		{"debug", func(l Logger) { l.Debug("debugging", "k", "val") }},
		{"error", func(l Logger) { l.Error(errors.New("boom"), "failed", "reason", "unit-test") }},
	}

	for _, tc := range cases {
		buf1.Reset()
		buf2.Reset()

		tc.call(sl)
		tc.call(ll)

		out1 := buf1.String()
		fmt.Println("SL:", out1)
		out2 := buf2.String()

		tokens1 := parseLogTokens(out1)
		tokens2 := parseLogTokens(out2)

		// compare token sets
		if len(tokens1) != len(tokens2) {
			t.Fatalf("%s: token count differ\nslog: %q\nlogr: %q", tc.name, out1, out2)
		}
		for k, v1 := range tokens1 {
			if v2, ok := tokens2[k]; !ok {
				t.Fatalf("%s: key %q missing in logr output\nslog: %q\nlogr: %q", tc.name, k, out1, out2)
			} else if v1 != v2 {
				t.Fatalf("%s: value mismatch for %q: slog=%q logr=%q\nslog: %q\nlogr: %q", tc.name, k, v1, v2, out1, out2)
			}
		}
	}
}
