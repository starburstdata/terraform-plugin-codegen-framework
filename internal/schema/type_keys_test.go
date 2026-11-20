// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package schema_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-codegen-spec/resource"
	specschema "github.com/hashicorp/terraform-plugin-codegen-spec/schema"

	generatorresource "github.com/starburstdata/terraform-plugin-codegen-framework/internal/resource"
	"github.com/starburstdata/terraform-plugin-codegen-framework/internal/schema"
)

func str(name string) resource.Attribute {
	return resource.Attribute{Name: name, String: &resource.StringAttribute{ComputedOptionalRequired: specschema.Required}}
}

func TestDisambiguateTypeKeys_ReservedName(t *testing.T) {
	t.Parallel()
	got := typeKeys(t, resource.Attributes{
		nested("a", nested("b_nested_columns", str("third_shape")), nested("nested_columns", str("expression"))),
		nested("b", nested("nested_columns", str("field_parser_config"))),
	})
	reserved := keyAt(t, got, "a", "b_nested_columns")
	allocated := keyAt(t, got, "b", "nested_columns")
	if reserved != "b_nested_columns" || allocated != "b_nested_columns_2" {
		t.Fatalf("expected reserved name and suffixed fallback, got %q and %q", reserved, allocated)
	}
}

func TestDisambiguateTypeKeys_ReserveNamesBeforeAllocation(t *testing.T) {
	t.Parallel()
	got := typeKeys(t, resource.Attributes{
		nested("a", nested("columns", str("expression"))),
		nested("b", nested("columns", str("field_parser_config"))),
		nested("b_columns", str("third_shape")),
	})
	reserved := keyAt(t, got, "b_columns")
	allocated := keyAt(t, got, "b", "columns")
	if reserved != "b_columns" || allocated != "b_columns_2" {
		t.Fatalf("expected future name to remain reserved, got %q and %q", reserved, allocated)
	}
}

func TestDisambiguateTypeKeys_NormalizedNames(t *testing.T) {
	t.Parallel()
	got := typeKeys(t, resource.Attributes{
		nested("a", nested("foo_1", str("expression"))),
		nested("b", nested("foo1", str("field_parser_config"))),
	})
	left := schema.FrameworkIdentifier(keyAt(t, got, "a", "foo_1")).ToPascalCase()
	right := schema.FrameworkIdentifier(keyAt(t, got, "b", "foo1")).ToPascalCase()
	if left == right {
		t.Fatalf("different shapes still generate the same Go identifier: %s", left)
	}
}

func TestDisambiguateTypeKeys_ReuseEachShape(t *testing.T) {
	t.Parallel()
	got := typeKeys(t, resource.Attributes{
		nested("a", nested("columns", str("expression"))),
		nested("b", nested("columns", str("field_parser_config"))),
		nested("c", nested("columns", str("field_parser_config"))),
	})
	a, b, c := keyAt(t, got, "a", "columns"), keyAt(t, got, "b", "columns"), keyAt(t, got, "c", "columns")
	if a == b || b != c {
		t.Fatalf("expected one type per shape, got %q, %q, %q", a, b, c)
	}
}

func TestDisambiguateTypeKeys_DeepRepeatedNames(t *testing.T) {
	t.Parallel()
	branch := func(field string) resource.Attribute {
		child := nested("child", str(field))
		for depth := 0; depth < 15; depth++ {
			child = nested("child", child)
		}
		return child
	}
	got := typeKeys(t, resource.Attributes{nested("a", branch("expression")), nested("b", branch("parser"))})
	if _, err := got.AttrTypes(); err != nil {
		t.Fatal(err)
	}
}

func TestDisambiguateTypeKeys_Deterministic(t *testing.T) {
	t.Parallel()
	makeAttrs := func() resource.Attributes {
		return resource.Attributes{
			nested("a", nested("b_columns", str("reserved")), nested("columns", str("expression"))),
			nested("b", nested("columns", str("parser"))),
		}
	}
	first := typeKeys(t, makeAttrs())
	for iteration := 0; iteration < 20; iteration++ {
		got := typeKeys(t, makeAttrs())
		if !first.Equal(got) {
			t.Fatalf("allocation changed on iteration %d", iteration)
		}
	}
}

func nested(name string, attrs ...resource.Attribute) resource.Attribute {
	return resource.Attribute{
		Name: name,
		ListNested: &resource.ListNestedAttribute{
			ComputedOptionalRequired: specschema.Optional,
			NestedObject:             resource.NestedAttributeObject{Attributes: attrs},
		},
	}
}

func typeKeys(t *testing.T, attrs resource.Attributes) schema.GeneratorAttributes {
	t.Helper()

	got, err := generatorresource.NewAttributes(attrs)
	if err != nil {
		t.Fatal(err)
	}

	if err := schema.DisambiguateTypeKeys(got, nil); err != nil {
		t.Fatal(err)
	}

	return got
}

func keyAt(t *testing.T, attrs schema.GeneratorAttributes, path ...string) string {
	t.Helper()

	var a schema.GeneratorAttribute

	for _, p := range path {
		a = attrs[p]
		if a == nil {
			t.Fatalf("missing attribute %q", p)
		}

		if k, ok := a.(schema.Attributes); ok {
			attrs = k.GetAttributes()
		}
	}

	return schema.TypeKeyOf(a, schema.FrameworkIdentifier(path[len(path)-1])).ToString()
}

func TestDisambiguateTypeKeys_SameNameDifferentShape(t *testing.T) {
	t.Parallel()

	got := typeKeys(t, resource.Attributes{
		nested("columns", str("expression"), nested("nested_columns", str("expression"))),
		nested("user_defined_schema", str("field_parser_config"), nested("nested_columns", str("field_parser_config"))),
	})

	a := keyAt(t, got, "columns", "nested_columns")
	b := keyAt(t, got, "user_defined_schema", "nested_columns")

	if a == b {
		t.Fatalf("expected distinct type keys, both are %q", a)
	}

	if a != "nested_columns" {
		t.Errorf("first attribute should keep its name, got %q", a)
	}

	if b != "user_defined_schema_nested_columns" {
		t.Errorf("expected the clashing attribute to be prefixed with its parent, got %q", b)
	}
}

func TestDisambiguateTypeKeys_SameNameSameShapeIsShared(t *testing.T) {
	t.Parallel()

	got := typeKeys(t, resource.Attributes{
		nested("columns", nested("nested_columns", str("expression"))),
		nested("other", nested("nested_columns", str("expression"))),
	})

	if a, b := keyAt(t, got, "columns", "nested_columns"), keyAt(t, got, "other", "nested_columns"); a != "nested_columns" || b != "nested_columns" {
		t.Errorf("identical shapes must keep sharing one type, got %q and %q", a, b)
	}
}

func TestDisambiguateTypeKeys_NoClashLeavesKeysUntouched(t *testing.T) {
	t.Parallel()

	got := typeKeys(t, resource.Attributes{
		nested("columns", str("expression")),
		nested("other", str("name")),
	})

	for _, name := range []string{"columns", "other"} {
		if k := keyAt(t, got, name); k != name {
			t.Errorf("%s: expected untouched key, got %q", name, k)
		}
	}
}
