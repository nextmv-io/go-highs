package mip

import (
	"os"
	"testing"

	"github.com/nextmv-io/sdk/golden"
)

// This test will generate the given template and use the generated main.py
// file to run against the input.json file. The output will be compared against
// the given golden file. In addition, the shell command will be tested to make
// sure that it runs without errors and the stderr log is as expected.

const template = "order-fulfillment-gosdk"

func TestMain(m *testing.M) {
	golden.SetupTemplateTest(template)
	code := m.Run()
	golden.Teardown(template, "output.json")
	os.Exit(code)
}

func TestGolden(t *testing.T) {
	golden.FileTest(
		t,
		"testdata/input.json",
		golden.Config{
			Args: []string{
				"-solve.duration",
				"30s",
			},
			// TODO: fully compare the solution again once we have a stable input
			DedicatedComparison: []string{
				".version.sdk",
				".statistics.result.custom.delivery_costs",
				".statistics.result.custom.handling_costs",
				".statistics.result.value",
			},
			TransientFields: []golden.TransientField{
				{
					Key:         ".version.sdk",
					Replacement: golden.StableVersion,
				},
				{
					Key:         ".statistics.result.duration",
					Replacement: golden.StableFloat,
				},
				{
					Key:         ".statistics.run.duration",
					Replacement: golden.StableFloat,
				},
			},
			Thresholds: golden.Tresholds{
				Float: 0.1,
			},
		},
	)
}
