package sloglambda

import (
	"fmt"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_loggerLevelFromString(t *testing.T) {
	cases := map[string]slog.Level{
		"TRACE":  slog.LevelDebug - 4,
		"DEBUG":  slog.LevelDebug,
		"INFO":   slog.LevelInfo,
		"WARN":   slog.LevelWarn,
		"ERROR":  slog.LevelError,
		"FATAL":  slog.LevelError + 4,
		"trace":  slog.LevelDebug - 4,
		"debug":  slog.LevelDebug,
		"info":   slog.LevelInfo,
		"Warn":   slog.LevelWarn,
		" error": slog.LevelError,
		" info ": slog.LevelInfo,
		"":       slog.LevelInfo,
	}

	for str, level := range cases {
		t.Run(fmt.Sprintf("%s=%s", str, &level), func(t *testing.T) {
			assert.Equal(t, level, loggerLevelFromString(str))
		})
	}
}

func Test_lambdaLoggerLevelString(t *testing.T) {
	cases := map[slog.Level]string{
		slog.LevelDebug - 8: "TRACE-4",
		slog.LevelDebug - 4: "TRACE",
		slog.LevelDebug:     "DEBUG",
		slog.LevelInfo:      "INFO",
		slog.LevelWarn:      "WARN",
		slog.LevelError:     "ERROR",
		slog.LevelError + 4: "FATAL",
		slog.LevelError + 8: "FATAL+4",
	}

	for level, str := range cases {
		t.Run(fmt.Sprintf("%s=%s", level, str), func(t *testing.T) {
			assert.Equal(t, str, lambdaLoggerLevelString(level))
		})
	}
}
