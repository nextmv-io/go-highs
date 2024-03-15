// © 2019-present nextmv.io inc

package main_test

import (
	"os"
	"testing"
	"time"

	"github.com/nextmv-io/sdk/golden"
)

func TestMain(m *testing.M) {
	golden.Setup()
	code := m.Run()
	golden.Teardown("input.json")
	os.Exit(code)
}

// TestGolden executes a golden file test, where the .json input is fed and an
// output is expected.
func TestGolden(t *testing.T) {
	golden.FileTests(
		t,
		"testdata",
		golden.Config{
			Args: []string{
				"-solve.duration", "10s",
			},
			TransientFields: []golden.TransientField{
				{Key: ".version.sdk", Replacement: golden.StableVersion},
				{Key: ".version.go-mip", Replacement: golden.StableVersion},
				{Key: ".statistics.result.duration", Replacement: golden.StableFloat},
				{Key: ".statistics.run.duration", Replacement: golden.StableFloat},
			},
			Thresholds: golden.Tresholds{
				Float:    0.01,
				Time:     time.Duration(5) * time.Second,
				Duration: time.Duration(5) * time.Second,
			},
		},
	)
}
