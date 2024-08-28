package sloglambda_test

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"testing"
	"testing/slogtest"

	sloglambda "github.com/maddiesch/slog-lambda"
	"github.com/stretchr/testify/assert"
)

func TestHandler(t *testing.T) {
	t.Run("slogtest", func(t *testing.T) {
		t.Run("JSON", func(t *testing.T) {
			buffer := new(bytes.Buffer)
			newHandler := func(t *testing.T) slog.Handler {
				t.Cleanup(buffer.Reset)

				return sloglambda.NewHandler(buffer, sloglambda.WithLevel(slog.LevelDebug), sloglambda.WithJSON())
			}

			result := func(t *testing.T) map[string]any {
				result := make(map[string]any)
				json.Unmarshal(buffer.Bytes(), &result)
				return result
			}

			slogtest.Run(t, newHandler, result)
		})

		t.Run("Text", func(t *testing.T) {
			t.Skip("Text formatting is not implemented yet")

			buffer := new(bytes.Buffer)
			newHandler := func(t *testing.T) slog.Handler {
				t.Cleanup(buffer.Reset)

				return sloglambda.NewHandler(buffer, sloglambda.WithLevel(slog.LevelDebug), sloglambda.WithText())
			}

			result := func(t *testing.T) map[string]any {
				result := make(map[string]any)
				json.Unmarshal(buffer.Bytes(), &result)
				return result
			}

			slogtest.Run(t, newHandler, result)
		})
	})

	t.Run("WithoutTime", func(t *testing.T) {
		t.Run("JSON", func(t *testing.T) {
			buffer := new(bytes.Buffer)
			logger := slog.New(sloglambda.NewHandler(buffer, sloglambda.WithJSON(), sloglambda.WithoutTime()))

			logger.Info(t.Name())

			assert.NotContains(t, buffer.String(), `"time"`)
		})

		t.Run("Text", func(t *testing.T) {
			t.Skip("Text formatting is not implemented yet")

			buffer := new(bytes.Buffer)
			logger := slog.New(sloglambda.NewHandler(buffer, sloglambda.WithText(), sloglambda.WithoutTime()))

			logger.Info(t.Name())

			assert.NotContains(t, buffer.String(), `time=`)
		})
	})

	t.Run("WithSource", func(t *testing.T) {
		t.Run("JSON", func(t *testing.T) {
			buffer := new(bytes.Buffer)
			logger := slog.New(sloglambda.NewHandler(buffer, sloglambda.WithJSON(), sloglambda.WithSource()))

			logger.Info(t.Name())

			assert.Contains(t, buffer.String(), `"source":{`)
		})

		t.Run("Text", func(t *testing.T) {
			t.Skip("Text formatting is not implemented yet")

			buffer := new(bytes.Buffer)
			logger := slog.New(sloglambda.NewHandler(buffer, sloglambda.WithText(), sloglambda.WithSource()))

			logger.Info(t.Name())

			assert.Contains(t, buffer.String(), `"source":{`)
		})
	})

	t.Run("WithType", func(t *testing.T) {
		t.Run("JSON", func(t *testing.T) {
			buffer := new(bytes.Buffer)
			logger := slog.New(sloglambda.NewHandler(buffer, sloglambda.WithJSON(), sloglambda.WithType(t.Name())))

			logger.Info(t.Name())

			assert.Contains(t, buffer.String(), `"type":"`+t.Name()+`"`)
		})

		t.Run("Text", func(t *testing.T) {
			t.Skip("Text formatting is not implemented yet")

			buffer := new(bytes.Buffer)
			logger := slog.New(sloglambda.NewHandler(buffer, sloglambda.WithText(), sloglambda.WithType(t.Name())))

			logger.Info(t.Name())

			assert.Contains(t, buffer.String(), `type="`+t.Name()+`"`)
		})
	})
}

func BenchmarkJSON(b *testing.B) {
	logger := slog.New(sloglambda.NewHandler(io.Discard, sloglambda.WithJSON())).WithGroup("benchmark").With("format", "json")

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		logger.Info("test", "count", i)
	}
}

func BenchmarkText(b *testing.B) {
	b.Skip("Text formatting is not implemented yet")

	logger := slog.New(sloglambda.NewHandler(io.Discard, sloglambda.WithText())).WithGroup("benchmark").With("format", "text")

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		logger.Info("test", "count", i)
	}
}
