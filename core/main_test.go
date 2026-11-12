package core

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	Defaults = os.DirFS("..")
	os.Exit(m.Run())
}
