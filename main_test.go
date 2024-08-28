package sloglambda_test

import (
	"os"
	"testing"

	"go.uber.org/goleak"
)

func TestMain(m *testing.M) {
	os.Setenv("AWS_LAMBDA_FUNCTION_NAME", "test-function")
	os.Setenv("AWS_LAMBDA_FUNCTION_VERSION", "$LATEST")
	os.Setenv("AWS_LAMBDA_LOG_FORMAT", "JSON")

	goleak.VerifyTestMain(m)
}
