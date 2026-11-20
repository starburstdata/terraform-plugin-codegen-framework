// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package cmd_test

import (
	_ "embed"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/cli"
	"github.com/starburstdata/terraform-plugin-codegen-framework/internal/cmd"
)

//go:embed testdata/nested_types/runtime_test.go.txt
var nestedTypesRuntimeTest string

func TestGenerateAll_NestedTypeRoundTrip(t *testing.T) {
	t.Parallel()
	for _, kind := range []string{"list_nested", "map_nested", "set_nested", "single_nested"} {
		t.Run(kind, func(t *testing.T) {
			t.Parallel()
			dir, err := os.MkdirTemp(".", "nested-types-") //nolint:usetesting // Keep generated module scratch inside the checkout.
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				if err := os.RemoveAll(dir); err != nil {
					t.Error(err)
				}
			})
			stringAttribute := func(name, mode string) map[string]any {
				return map[string]any{"name": name, "string": map[string]any{"computed_optional_required": mode}}
			}
			nested := func(name, kind, mode string, children ...map[string]any) map[string]any {
				body := map[string]any{"computed_optional_required": mode}
				if kind == "single_nested" {
					body["attributes"] = children
				} else {
					body["nested_object"] = map[string]any{"attributes": children}
				}
				return map[string]any{"name": name, kind: body}
			}
			entry := func(mode string) map[string]any {
				return map[string]any{"name": "example", "schema": map[string]any{"attributes": []map[string]any{
					nested("a", "single_nested", mode, nested("columns", kind, mode, stringAttribute("expression", mode))),
					nested("b", "single_nested", mode, nested("columns", kind, mode, stringAttribute("field_parser_config", mode))),
				}}}
			}
			ir, err := json.Marshal(map[string]any{
				"version": "0.1", "provider": map[string]any{"name": "example"},
				"resources": []map[string]any{entry("optional")}, "datasources": []map[string]any{entry("computed")},
			})
			if err != nil {
				t.Fatal(err)
			}
			input := filepath.Join(dir, "ir.json")
			if err := os.WriteFile(input, ir, 0600); err != nil {
				t.Fatal(err)
			}
			ui := cli.NewMockUi()
			generator := cmd.GenerateAllCommand{UI: ui}
			if status := generator.Run([]string{"--input", input, "--output", dir}); status != 0 {
				t.Fatalf("generation failed: %s", ui.ErrorWriter.String())
			}
			for _, target := range []struct{ pkg, function string }{
				{"resource_example", "ExampleResourceSchema"},
				{"datasource_example", "ExampleDataSourceSchema"},
			} {
				source := strings.ReplaceAll(nestedTypesRuntimeTest, "package generated", "package "+target.pkg)
				source = strings.ReplaceAll(source, "SCHEMA", target.function)
				if err := os.WriteFile(filepath.Join(dir, target.pkg, "runtime_test.go"), []byte(source), 0600); err != nil {
					t.Fatal(err)
				}
			}
			module := "module example.com/generated\n\ngo 1.23.0\n\nrequire github.com/hashicorp/terraform-plugin-framework v1.15.1\n"
			if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(module), 0600); err != nil {
				t.Fatal(err)
			}
			check := exec.Command("go", "test", "-mod=mod", "./...", "-count=1")
			check.Dir = dir
			output, err := check.CombinedOutput()
			if err != nil {
				t.Fatalf("generated code failed runtime tests: %v\n%s", err, output)
			}
		})
	}
}
