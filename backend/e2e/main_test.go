package e2e

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	if err := initShared(); err != nil {
		panic("e2e harness failed to start: " + err.Error())
	}
	code := m.Run()
	closeShared()
	os.Exit(code)
}
