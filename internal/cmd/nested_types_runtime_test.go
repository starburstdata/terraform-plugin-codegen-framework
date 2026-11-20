// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package cmd_test

import (
	_ "embed"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/hashicorp/cli"
	"github.com/starburstdata/terraform-plugin-codegen-framework/internal/cmd"
)

//go:embed testdata/nested_types/runtime_test.go.txt
var nestedTypesRuntimeTest string

func TestGenerateAll_NestedTypeRoundTrip(t *testing.T) {
	t.Parallel()
	scenarios := []struct{ blockKind, placement string }{{}}
	for _, blockKind := range []string{"list_nested", "set_nested", "single_nested"} {
		for _, placement := range []string{"top", "nested"} {
			scenarios = append(scenarios, struct{ blockKind, placement string }{blockKind, placement})
		}
	}
	for _, kind := range []string{"list_nested", "map_nested", "set_nested", "single_nested"} {
		for _, scenario := range scenarios {
			name := kind
			if scenario.blockKind != "" {
				name += "/" + scenario.blockKind + "/" + scenario.placement
			}
			t.Run(name, func(t *testing.T) {
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
				block := func(name, kind string, attributes, blocks []map[string]any) map[string]any {
					body := map[string]any{}
					if len(attributes) != 0 {
						body["attributes"] = attributes
					}
					if len(blocks) != 0 {
						body["blocks"] = blocks
					}
					if kind != "single_nested" {
						body = map[string]any{"nested_object": body}
					}
					return map[string]any{"name": name, kind: body}
				}
				entry := func(mode string) map[string]any {
					schema := map[string]any{"attributes": []map[string]any{
						nested("a", "single_nested", mode, nested("columns", kind, mode, stringAttribute("expression", mode))),
						nested("b", "single_nested", mode, nested("columns", kind, mode, stringAttribute("field_parser_config", mode))),
					}}
					if scenario.blockKind != "" {
						fixed := block("b_columns", scenario.blockKind, []map[string]any{stringAttribute("block_field", mode)}, nil)
						if scenario.placement == "nested" {
							fixed = block("wrapper", "single_nested", nil, []map[string]any{fixed})
						}
						schema["blocks"] = []map[string]any{
							fixed,
							block("alternate", "single_nested", nil, []map[string]any{
								block("b_columns", scenario.blockKind, []map[string]any{stringAttribute("alternate_field", mode)}, nil),
							}),
							block("holder", scenario.blockKind, []map[string]any{
								nested("b_columns", kind, mode, stringAttribute("field_parser_config", mode)),
							}, nil),
						}
					}
					return map[string]any{"name": "example", "schema": schema}
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
					source, blockTests, _ := strings.Cut(nestedTypesRuntimeTest, "// BLOCK_TESTS\n")
					if scenario.blockKind != "" {
						source += blockTests
						source = strings.ReplaceAll(source, "NESTED_BLOCK", strconv.FormatBool(scenario.placement == "nested"))
					}
					source = strings.ReplaceAll(source, "package generated", "package "+target.pkg)
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
}
