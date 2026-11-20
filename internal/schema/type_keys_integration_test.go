// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package schema_test

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	"github.com/hashicorp/terraform-plugin-codegen-spec/datasource"
	"github.com/hashicorp/terraform-plugin-codegen-spec/resource"
	specschema "github.com/hashicorp/terraform-plugin-codegen-spec/schema"

	generatordatasource "github.com/starburstdata/terraform-plugin-codegen-framework/internal/datasource"
	generatorresource "github.com/starburstdata/terraform-plugin-codegen-framework/internal/resource"
	"github.com/starburstdata/terraform-plugin-codegen-framework/internal/schema"
)

func assertUniqueGeneratedSymbols(t *testing.T, generated schema.GeneratorSchema) {
	t.Helper()
	assertTypeKeysIdempotent(t, generated)
	code, err := generated.CustomTypeValueBytes()
	if err != nil {
		t.Fatal(err)
	}
	file, err := parser.ParseFile(token.NewFileSet(), "", append([]byte("package generated\n"), code...), 0)
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	record := func(name string) {
		if seen[name] {
			t.Errorf("duplicate generated Go symbol %s", name)
		}
		seen[name] = true
	}
	for _, declaration := range file.Decls {
		switch declaration := declaration.(type) {
		case *ast.FuncDecl:
			if declaration.Recv == nil {
				record(declaration.Name.Name)
			}
		case *ast.GenDecl:
			for _, spec := range declaration.Specs {
				if ty, ok := spec.(*ast.TypeSpec); ok {
					record(ty.Name.Name)
				}
			}
		}
	}
}

func assertTypeKeysIdempotent(t *testing.T, generated schema.GeneratorSchema) {
	t.Helper()
	before, err := generated.CustomTypeValueBytes()
	if err != nil {
		t.Fatal(err)
	}
	if err := schema.DisambiguateTypeKeys(generated.Attributes, generated.Blocks); err != nil {
		t.Fatal(err)
	}
	after, err := generated.CustomTypeValueBytes()
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Error("repeated type allocation changed generated output")
	}
}

func checkResourceAndDataSource(t *testing.T, input resource.Resource, checks ...func(*testing.T, schema.GeneratorSchema)) {
	t.Helper()
	t.Run("resource", func(t *testing.T) {
		generated, err := generatorresource.NewSchema(input)
		if err != nil {
			t.Fatal(err)
		}
		assertUniqueGeneratedSymbols(t, generated)
		for _, check := range checks {
			check(t, generated)
		}
	})
	t.Run("datasource", func(t *testing.T) {
		raw, err := json.Marshal(input)
		if err != nil {
			t.Fatal(err)
		}
		var equivalent datasource.DataSource
		if err := json.Unmarshal(raw, &equivalent); err != nil {
			t.Fatal(err)
		}
		generated, err := generatordatasource.NewSchema(equivalent)
		if err != nil {
			t.Fatal(err)
		}
		assertUniqueGeneratedSymbols(t, generated)
		for _, check := range checks {
			check(t, generated)
		}
	})
}

func TestTypeNames_ReserveFutureBlocks(t *testing.T) {
	t.Parallel()
	checkResourceAndDataSource(t, resource.Resource{
		Name: "example",
		Schema: &resource.Schema{Blocks: resource.Blocks{
			{
				Name: "a_holder",
				SingleNested: &resource.SingleNestedBlock{
					Attributes: resource.Attributes{nested("b_columns", str("field_parser_config"))},
				},
			},
			{
				Name: "b_columns",
				ListNested: &resource.ListNestedBlock{
					NestedObject: resource.NestedBlockObject{
						Attributes: resource.Attributes{str("block_field")},
					},
				},
			},
		}},
	}, func(t *testing.T, generated schema.GeneratorSchema) {
		if key := schema.TypeKeyOf(generated.Blocks["b_columns"], "b_columns"); key != "b_columns" {
			t.Fatalf("an earlier attribute stole the fixed block name: %s", key)
		}
	})
}

func TestTypeNames_ReserveBlockDeclarations(t *testing.T) {
	t.Parallel()
	checkResourceAndDataSource(t, resource.Resource{
		Name: "example",
		Schema: &resource.Schema{
			Attributes: resource.Attributes{
				nested("a", nested("columns", str("expression"))),
				nested("b", nested("columns", str("field_parser_config"))),
			},
			Blocks: resource.Blocks{{
				Name: "b_columns",
				ListNested: &resource.ListNestedBlock{
					ComputedOptionalRequired: specschema.Optional,
					NestedObject: resource.NestedBlockObject{
						Attributes: resource.Attributes{str("block_field")},
					},
				},
			}},
		},
	})
}

func TestTypeNames_ReserveAssociatedCustomDeclarations(t *testing.T) {
	t.Parallel()
	checkResourceAndDataSource(t, resource.Resource{
		Name: "example",
		Schema: &resource.Schema{
			Attributes: resource.Attributes{
				nested("a", nested("columns", str("expression"))),
				nested("b", nested("columns", str("field_parser_config"))),
				{
					Name: "b_columns",
					String: &resource.StringAttribute{
						ComputedOptionalRequired: specschema.Optional,
						AssociatedExternalType:   &specschema.AssociatedExternalType{Type: "string"},
					},
				},
			},
		},
	})
}

func TestTypeNames_ConstructorAndValueDeclarationsShareNamespace(t *testing.T) {
	t.Parallel()
	checkResourceAndDataSource(t, resource.Resource{
		Name: "example",
		Schema: &resource.Schema{Attributes: resource.Attributes{
			nested("columns", str("expression")),
			nested("new_columns", str("field_parser_config")),
		}},
	})
}

func TestTypeNames_DifferentBlockShapes(t *testing.T) {
	t.Parallel()
	wrapper := func(name, field string) resource.Block {
		return resource.Block{
			Name: name,
			SingleNested: &resource.SingleNestedBlock{
				Blocks: resource.Blocks{{
					Name: "columns",
					ListNested: &resource.ListNestedBlock{
						NestedObject: resource.NestedBlockObject{
							Attributes: resource.Attributes{str(field)},
						},
					},
				}},
			},
		}
	}
	checkResourceAndDataSource(t, resource.Resource{
		Name: "example",
		Schema: &resource.Schema{
			Blocks: resource.Blocks{wrapper("a", "expression"), wrapper("b", "field_parser_config")},
		},
	})
}

func TestTypeNames_NamespaceCombinations(t *testing.T) {
	t.Parallel()
	attributeNames := []string{"columns", "new_columns", "foo1", "foo_1", "b_columns", "b_columns_2"}
	blockNames := []string{"columns", "b_columns", "new_columns", "foo1"}
	for _, left := range attributeNames {
		for _, right := range attributeNames {
			for _, blockName := range blockNames {
				t.Run(left+"/"+right+"/"+blockName, func(t *testing.T) {
					t.Parallel()
					checkResourceAndDataSource(t, resource.Resource{
						Name: "example",
						Schema: &resource.Schema{
							Attributes: resource.Attributes{
								nested("a", nested(left, str("expression"))),
								nested("b", nested(right, str("field_parser_config"))),
							},
							Blocks: resource.Blocks{{
								Name: blockName,
								ListNested: &resource.ListNestedBlock{
									NestedObject: resource.NestedBlockObject{
										Attributes: resource.Attributes{str("block_field")},
									},
								},
							}},
						},
					})
				})
			}
		}
	}
}
