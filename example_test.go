package sloglambda_test

import (
	"log/slog"
	"os"

	sloglambda "github.com/maddiesch/slog-lambda"
)

func ExampleWithJSON() {
	handler := sloglambda.NewHandler(os.Stdout, sloglambda.WithJSON(), sloglambda.WithoutTime())
	logger := slog.New(handler)

	slog.SetDefault(logger)

	slog.Info("Hello, world!")
	// Output: {"level":"INFO","msg":"Hello, world!","record":{"functionName":"test-function","version":"$LATEST"},"type":"app.log"}
}

func ExampleWithText() {
	handler := sloglambda.NewHandler(os.Stdout, sloglambda.WithText(), sloglambda.WithoutTime())
	logger := slog.New(handler)

	slog.SetDefault(logger)

	slog.Info("Hello, world!")
	// Output: level="INFO" msg="Hello, world!" record.functionName="test-function" record.version="$LATEST" type="app.log"
}

func ExampleNewHandler() {
	handler := sloglambda.NewHandler(os.Stdout)
	logger := slog.New(handler)

	slog.SetDefault(logger)

	slog.Info("Hello, world!")
}
