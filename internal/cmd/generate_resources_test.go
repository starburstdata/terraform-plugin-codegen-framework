// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package cmd_test

import (
	"testing"

	"github.com/hashicorp/cli"
	"github.com/starburstdata/terraform-plugin-codegen-framework/internal/cmd"
)

func TestGenerateResourcesCommand(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		irInputPath   string
		goldenFileDir string
		injectID      bool
	}{
		"custom_and_external": {
			irInputPath:   "testdata/custom_and_external/ir.json",
			goldenFileDir: "testdata/custom_and_external/resources_output",
		},
		"inject_id": {
			irInputPath:   "testdata/inject_id/ir.json",
			goldenFileDir: "testdata/inject_id/resources_output_inject_id",
			injectID:      true,
		},
	}
	for name, testCase := range testCases {

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			testOutputDir := t.TempDir()
			mockUi := cli.NewMockUi()
			c := cmd.GenerateResourcesCommand{
				UI: mockUi,
			}

			args := []string{
				"--input", testCase.irInputPath,
				"--package", "generated",
				"--output", testOutputDir,
			}
			if testCase.injectID {
				args = append(args, "--inject-id")
			}

			exitCode := c.Run(args)
			if exitCode != 0 {
				t.Fatalf("unexpected error running `generate resources` cmd: %s", mockUi.ErrorWriter.String())
			}

			compareDirectories(t, testCase.goldenFileDir, testOutputDir)
		})
	}
}
