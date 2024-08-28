package sloglambda_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"testing"
	"testing/slogtest"

	sloglambda "github.com/maddiesch/slog-lambda"
)

func TestHandler(t *testing.T) {
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
}
