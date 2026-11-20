// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package resource

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-codegen-spec/resource"
	specschema "github.com/hashicorp/terraform-plugin-codegen-spec/schema"
)

func nestedList(name string, attrs ...resource.Attribute) resource.Attribute {
	return resource.Attribute{
		Name: name,
		ListNested: &resource.ListNestedAttribute{
			ComputedOptionalRequired: specschema.Optional,
			NestedObject:             resource.NestedAttributeObject{Attributes: attrs},
		},
	}
}

func requiredString(name string) resource.Attribute {
	return resource.Attribute{Name: name, String: &resource.StringAttribute{ComputedOptionalRequired: specschema.Required}}
}

// Two nested attributes named nested_columns with different shapes must generate two Go types, and the
// schema and model must reference the matching one.
func TestNewSchema_SameNestedNameDifferentShapeGeneratesDistinctTypes(t *testing.T) {
	t.Parallel()

	s, err := NewSchema(resource.Resource{
		Name: "example",
		Schema: &resource.Schema{
			Attributes: resource.Attributes{
				nestedList("columns", requiredString("expression"), nestedList("nested_columns", requiredString("expression"))),
				nestedList("user_defined_schema", requiredString("field_parser_config"), nestedList("nested_columns", requiredString("field_parser_config"))),
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	types, err := s.CustomTypeValueBytes()
	if err != nil {
		t.Fatal(err)
	}

	out := string(types)

	for _, want := range []string{"type NestedColumnsType struct", "type UserDefinedSchemaNestedColumnsType struct"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected generated code to contain %q", want)
		}
	}

	schemaCode, err := s.Attributes.Schema()
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(schemaCode, "UserDefinedSchemaNestedColumnsType{") {
		t.Errorf("expected the schema of the clashing attribute to use its own custom type:\n%s", schemaCode)
	}
}
