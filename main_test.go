package sloglambda_test

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	os.Setenv("AWS_LAMBDA_FUNCTION_NAME", "test-function")
	os.Setenv("AWS_LAMBDA_FUNCTION_VERSION", "$LATEST")

	exitCode := m.Run()
	os.Exit(exitCode)
}
