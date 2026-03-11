package sloglambda

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"testing"
	"time"

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

func Test_normalizeValue(t *testing.T) {
	t.Run("slog.KindBool", func(t *testing.T) {
		v := slog.BoolValue(true)
		assert.Equal(t, true, normalizeValue(v))
	})

	t.Run("slog.KindDuration", func(t *testing.T) {
		v := slog.DurationValue(1 * time.Second)
		assert.Equal(t, "1s", normalizeValue(v))
	})

	t.Run("slog.KindFloat64", func(t *testing.T) {
		v := slog.Float64Value(1.23)
		assert.Equal(t, float64(1.23), normalizeValue(v))
	})

	t.Run("slog.KindUint64", func(t *testing.T) {
		v := slog.Uint64Value(123)
		assert.Equal(t, uint64(123), normalizeValue(v))
	})

	t.Run("slog.KindAny", func(t *testing.T) {
		t.Run("error", func(t *testing.T) {
			v := slog.AnyValue(errors.New("testing-error"))
			assert.Equal(t, "testing-error", normalizeValue(v))
		})

		t.Run("nil", func(t *testing.T) {
			v := slog.AnyValue(nil)
			assert.Equal(t, nil, normalizeValue(v))
		})

		t.Run("json.Marshaler", func(t *testing.T) {
			t.Run("success", func(t *testing.T) {
				v := slog.AnyValue(jsonMarshalerSuccess{})

				assert.Equal(t, `"JSON Marshal"`, normalizeValue(v))
			})

			t.Run("failure", func(t *testing.T) {
				v := slog.AnyValue(jsonMarshalerFail{})

				assert.Equal(t, assert.AnError.Error(), normalizeValue(v))
			})
		})
	})

	t.Run("slog.KindGroup", func(t *testing.T) {
		t.Run("non-empty group", func(t *testing.T) {
			v := slog.GroupValue(slog.String("key", "value"), slog.Int("num", 42))
			result := normalizeValue(v)
			rec, ok := result.(logRecord)
			require.True(t, ok, "expected logRecord")
			assert.Equal(t, "value", rec["key"])
			assert.Equal(t, int64(42), rec["num"])
		})

		t.Run("empty group", func(t *testing.T) {
			v := slog.GroupValue()
			assert.Nil(t, normalizeValue(v))
		})
	})

	t.Run("LogValuer resolving to group", func(t *testing.T) {
		v := slog.AnyValue(groupLogValuer{})
		result := normalizeValue(v)
		rec, ok := result.(logRecord)
		require.True(t, ok, "expected logRecord")
		assert.Equal(t, "bar", rec["foo"])
	})

	t.Run("unknown kind returns string", func(t *testing.T) {
		// Verify default case doesn't panic
		assert.NotPanics(t, func() {
			normalizeValue(slog.Value{})
		})
	})
}

type groupLogValuer struct{}

func (groupLogValuer) LogValue() slog.Value {
	return slog.GroupValue(slog.String("foo", "bar"))
}

type jsonMarshalerSuccess struct{}

func (jsonMarshalerSuccess) MarshalJSON() ([]byte, error) {
	return []byte(`"JSON Marshal"`), nil
}

var _ json.Marshaler = (*jsonMarshalerSuccess)(nil)

type jsonMarshalerFail struct{}

func (jsonMarshalerFail) MarshalJSON() ([]byte, error) {
	return nil, assert.AnError
}

var _ json.Marshaler = (*jsonMarshalerFail)(nil)
