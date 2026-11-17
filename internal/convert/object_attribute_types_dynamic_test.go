// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package convert

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	specschema "github.com/hashicorp/terraform-plugin-codegen-spec/schema"
)

func TestObjectAttributeTypes_Dynamic(t *testing.T) {
	t.Parallel()

	o := NewObjectAttributeTypes(specschema.ObjectAttributeTypes{
		{Name: "payload", Dynamic: &specschema.DynamicType{}},
	})

	if diff := cmp.Diff(string(o.AttributeTypes()), `"payload": types.DynamicType,`); diff != "" {
		t.Errorf("unexpected difference: %s", diff)
	}

	if len(o.Imports().All()) == 0 {
		t.Error("expected attr and types imports for a dynamic attribute type")
	}
}
