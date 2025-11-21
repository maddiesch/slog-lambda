package sloglambda_test

import (
	"bytes"
	"log/slog"
	"net/http"
	"testing"

	sloglambda "github.com/maddiesch/slog-lambda"
	"github.com/stretchr/testify/assert"
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

	t.Run("http.Request in Group", func(t *testing.T) {
		logger := slog.New(newHandler(t))
		req, _ := http.NewRequest("GET", "/foo/bar", nil)
		logger.InfoContext(t.Context(), t.Name(), slog.Group("http", slog.Any("request", req)))
	})

	t.Run("LogValuer", func(t *testing.T) {
		t.Run("top level attr", func(t *testing.T) {
			logger := slog.New(newHandler(t))
			logger.InfoContext(t.Context(), t.Name(), "value", testLogValuer{"Foo Bar"})
			assert.Contains(t, buffer.String(), `value="Foo Bar"`)
		})

		t.Run("group level attr", func(t *testing.T) {
			logger := slog.New(newHandler(t))
			logger.InfoContext(t.Context(), t.Name(), slog.Group("group", "value", testLogValuer{"Foo Bar"}))
			assert.Contains(t, buffer.String(), `group.value="Foo Bar"`)
		})

		t.Run("nested top level attr", func(t *testing.T) {
			logger := slog.New(newHandler(t))
			logger.InfoContext(t.Context(), t.Name(), "value", nestedTestLogValuer{testLogValuer{"Foo Bar"}})
			assert.Contains(t, buffer.String(), `value="Foo Bar"`)
		})

		t.Run("nested group level attr", func(t *testing.T) {
			logger := slog.New(newHandler(t))
			logger.InfoContext(t.Context(), t.Name(), slog.Group("group", "value", nestedTestLogValuer{testLogValuer{"Foo Bar"}}))
			assert.Contains(t, buffer.String(), `group.value="Foo Bar"`)
		})
	})
}

type nestedTestLogValuer struct {
	Value testLogValuer
}

func (lv nestedTestLogValuer) LogValue() slog.Value {
	return slog.AnyValue(lv.Value)
}

type testLogValuer struct {
	Value string
}

func (lv testLogValuer) LogValue() slog.Value {
	return slog.StringValue(lv.Value)
}

var _ slog.LogValuer = (*testLogValuer)(nil)
