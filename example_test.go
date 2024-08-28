package sloglambda_test

import (
	"log/slog"
	"os"

	sloglambda "github.com/maddiesch/slog-lambda"
)

func ExampleNewHandler() {
	handler := sloglambda.NewHandler(os.Stdout, sloglambda.WithJSON(), sloglambda.WithoutTime())
	logger := slog.New(handler)

	slog.SetDefault(logger)

	slog.Info("Hello, world!")
	// Output: {"level":"INFO","msg":"Hello, world!","record":{"functionName":"test-function","functionVersion":"$LATEST"},"type":"app.log"}
}
