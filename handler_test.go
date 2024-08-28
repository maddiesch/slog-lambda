package sloglambda_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"strconv"
	"strings"
	"testing"
	"testing/slogtest"

	"github.com/aws/aws-lambda-go/lambdacontext"
	sloglambda "github.com/maddiesch/slog-lambda"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
			buffer := new(bytes.Buffer)
			newHandler := func(t *testing.T) slog.Handler {
				t.Cleanup(buffer.Reset)

				return sloglambda.NewHandler(buffer, sloglambda.WithLevel(slog.LevelDebug), sloglambda.WithText())
			}

			result := func(t *testing.T) map[string]any {
				result := make(map[string]any)

				unquote := func(s string) string {
					if len(s) == 0 || s[0] != '"' {
						return s
					}

					s, err := strconv.Unquote(s)
					require.NoError(t, err)
					return s
				}

				parts := strings.Split(strings.TrimSpace(buffer.String()), " ")
				for _, entry := range parts {
					parts := strings.SplitN(entry, "=", 2)
					path := strings.Split(parts[0], ".")
					if len(path) == 1 {
						result[path[0]] = unquote(parts[1])
						continue
					}

					v := result

					for i := 0; i < len(path)-1; i++ {
						if _, ok := v[path[i]]; !ok {
							v[path[i]] = make(map[string]any)
						}
						v = v[path[i]].(map[string]any)
					}

					v[path[len(path)-1]] = unquote(parts[1])
				}

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
			buffer := new(bytes.Buffer)
			logger := slog.New(sloglambda.NewHandler(buffer, sloglambda.WithText(), sloglambda.WithSource()))

			logger.Info(t.Name())

			assert.Contains(t, buffer.String(), `source.function=`)
			assert.Contains(t, buffer.String(), `source.file=`)
			assert.Contains(t, buffer.String(), `source.line=`)
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
			buffer := new(bytes.Buffer)
			logger := slog.New(sloglambda.NewHandler(buffer, sloglambda.WithText(), sloglambda.WithType(t.Name())))

			logger.Info(t.Name())

			assert.Contains(t, buffer.String(), `type="`+t.Name()+`"`)
		})
	})

	t.Run("given a lambda context", func(t *testing.T) {
		ctx := lambdacontext.NewContext(context.Background(), &lambdacontext.LambdaContext{
			AwsRequestID: "abc-123",
		})

		t.Run("JSON", func(t *testing.T) {
			buffer := new(bytes.Buffer)
			logger := slog.New(sloglambda.NewHandler(buffer, sloglambda.WithJSON()))

			logger.InfoContext(ctx, t.Name())

			assert.Contains(t, buffer.String(), `"requestId":"abc-123"`)
		})

		t.Run("Text", func(t *testing.T) {
			buffer := new(bytes.Buffer)
			logger := slog.New(sloglambda.NewHandler(buffer, sloglambda.WithText()))

			logger.InfoContext(ctx, t.Name())

			assert.Contains(t, buffer.String(), `record.requestId="abc-123"`)
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
	logger := slog.New(sloglambda.NewHandler(io.Discard, sloglambda.WithText())).WithGroup("benchmark").With("format", "text")

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		logger.Info("test", "count", i)
	}
}
