package sloglambda_test

import (
	"bytes"
	"log/slog"
	"net/http"
	"testing"

	sloglambda "github.com/maddiesch/slog-lambda"
)

func TestValueTypes(t *testing.T) {
	buffer := new(bytes.Buffer)
	newHandler := func(t testing.TB) slog.Handler {
		t.Cleanup(buffer.Reset)

		return sloglambda.NewHandler(buffer, sloglambda.WithLevel(slog.LevelDebug), sloglambda.WithText())
	}

	t.Run("http.Request", func(t *testing.T) {
		logger := slog.New(newHandler(t))
		req, _ := http.NewRequest("GET", "/foo/bar", nil)
		logger.InfoContext(t.Context(), t.Name(), slog.Any("request", req))
	})
}
