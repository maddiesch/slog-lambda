# slog-lambda

[![current](https://img.shields.io/github/v/tag/maddiesch/slog-lambda.svg)](https://github.com/maddiesch/slog-lambda/releases)
[![codecov](https://codecov.io/gh/maddiesch/slog-lambda/graph/badge.svg?token=G27YB5T5J3)](https://codecov.io/gh/maddiesch/slog-lambda)
[![Doc](https://godoc.org/github.com/maddiesch/slog-lambda?status.svg)](https://pkg.go.dev/github.com/maddiesch/slog-lambda)
[![License](https://img.shields.io/github/license/maddiesch/slog-lambda)](./LICENSE)

## AWS Lambda `slog.Handler`

A [`log/slog`](https://pkg.go.dev/log/slog) handler for [AWS Lambda advanced logging controls](https://docs.aws.amazon.com/lambda/latest/dg/monitoring-cloudwatchlogs-advanced.html).

```go
package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	sloglambda "github.com/maddiesch/slog-lambda"
)

func init() {
	logger := slog.New(sloglambda.NewHandler(os.Stdout))
	slog.SetDefault(logger)
}

func main() {
	lambda.Start(func(ctx context.Context, event any) error {
		slog.InfoContext(ctx, "Lambda Invoked", slog.Any("event", event))

		return nil
	})
}
```
