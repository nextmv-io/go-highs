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
				"-solve.mip.gap.absolute", "80",
				"-solve.mip.gap.relative", "0.4",
				"-solve.control.float", "mip_heuristic_effort=0.7",
				"-solve.control.int", "mip_max_nodes=200,threads=1",
				"-solve.control.string", "presolve=off",
			},
			TransientFields: []golden.TransientField{
				{Key: ".version.sdk", Replacement: golden.StableVersion},
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
