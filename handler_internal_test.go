package sloglambda

import (
	"bytes"
	"fmt"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func Test_logRecord(t *testing.T) {
	t.Run("clean", func(t *testing.T) {
		t.Run("when the log record has an empty sub-record", func(t *testing.T) {
			r := logRecord{
				"foo": logRecord{},
			}
			r.clean()

			_, ok := r["foo"]
			assert.False(t, ok, "the sub-record should have been removed")
		})

		t.Run("when the log record has a non-empty sub-record", func(t *testing.T) {
			r := logRecord{
				"foo": logRecord{"bar": "baz", "qux": logRecord{}},
			}
			r.clean()

			foo, ok := r["foo"]
			require.True(t, ok, "the sub-record should not have been removed")

			_, ok = foo.(logRecord)["qux"]
			assert.False(t, ok, "the sub-record should have been removed")
		})
	})

	t.Run("append", func(t *testing.T) {
		t.Run("when given an empty group", func(t *testing.T) {
			r := logRecord{}
			r.append(slog.Group("foo"))

			assert.Equal(t, logRecord{}, r)
		})

		t.Run("when given a non-empty group without a name", func(t *testing.T) {
			r := logRecord{}
			r.append(slog.Group("", slog.String("foo", "bar")))

			assert.Equal(t, logRecord{"foo": "bar"}, r)
		})
	})
}

func Test_writeTextRecord(t *testing.T) {
	t.Run("when the record is empty", func(t *testing.T) {
		buffer := new(bytes.Buffer)
		err := writeTextRecord(buffer, logRecord{}, "")

		assert.NoError(t, err)
		assert.Equal(t, "", buffer.String())
	})

	t.Run("when the record is nil", func(t *testing.T) {
		buffer := new(bytes.Buffer)
		err := writeTextRecord(buffer, nil, "")

		assert.NoError(t, err)
		assert.Equal(t, "", buffer.String())
	})

	t.Run("when the record contains a stringer", func(t *testing.T) {
		buffer := new(bytes.Buffer)
		err := writeTextRecord(buffer, logRecord{"foo": stringerValue{}}, "")

		assert.NoError(t, err)
		assert.Equal(t, "foo=stringerValue ", buffer.String())
	})

	t.Run("when the record contains an int", func(t *testing.T) {
		buffer := new(bytes.Buffer)
		err := writeTextRecord(buffer, logRecord{"foo": 1}, "")

		assert.NoError(t, err)
		assert.Equal(t, "foo=1 ", buffer.String())
	})

	t.Run("when the record contains a string", func(t *testing.T) {
		buffer := new(bytes.Buffer)
		err := writeTextRecord(buffer, logRecord{"bar": "baz"}, "foo")

		assert.NoError(t, err)
		assert.Equal(t, `foo.bar="baz" `, buffer.String())
	})

	t.Run("when the record contains a sub-record", func(t *testing.T) {
		buffer := new(bytes.Buffer)
		err := writeTextRecord(buffer, logRecord{"bar": logRecord{"baz": 1}}, "foo")

		assert.NoError(t, err)
		assert.Equal(t, `foo.bar.baz=1 `, buffer.String())
	})
}

type stringerValue struct{}

func (s stringerValue) String() string {
	return "stringerValue"
}
