// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package resource

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-codegen-spec/resource"
	specschema "github.com/hashicorp/terraform-plugin-codegen-spec/schema"

	generatorschema "github.com/starburstdata/terraform-plugin-codegen-framework/internal/schema"
)

// TestGeneratorStringAttribute_CredentialAutoOverride verifies that string attributes
// whose name matches a credential pattern are emitted as Optional + Sensitive + WriteOnly,
// even when the spec marks them Required. This is what makes import-then-apply work for
// catalog credentials: state holds null after import, framework reads from req.Config.
func TestGeneratorStringAttribute_CredentialAutoOverride(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		attrName                     string
		specComputedOptionalRequired string
		wantWriteOnly                bool
		wantSensitive                bool
		wantRequired                 bool
		wantOptional                 bool
	}{
		"credentials_key required-in-spec -> stays Required, gains WriteOnly+Sensitive": {
			attrName:                     "credentials_key",
			specComputedOptionalRequired: "required",
			wantWriteOnly:                true,
			wantSensitive:                true,
			wantRequired:                 true,
			wantOptional:                 false,
		},
		"password required-in-spec -> stays Required, gains WriteOnly+Sensitive": {
			attrName:                     "password",
			specComputedOptionalRequired: "required",
			wantWriteOnly:                true,
			wantSensitive:                true,
			wantRequired:                 true,
			wantOptional:                 false,
		},
		"private_key optional-in-spec (multi-auth) -> stays Optional, gains WriteOnly+Sensitive": {
			attrName:                     "private_key",
			specComputedOptionalRequired: "optional",
			wantWriteOnly:                true,
			wantSensitive:                true,
			wantRequired:                 false,
			wantOptional:                 true,
		},
		"password computed_optional-in-spec (multi-auth) -> Optional (Computed dropped), gains WriteOnly+Sensitive": {
			attrName:                     "password",
			specComputedOptionalRequired: "computed_optional",
			wantWriteOnly:                true,
			wantSensitive:                true,
			wantRequired:                 false,
			wantOptional:                 true,
		},
		"name (non-credential) required -> stays Required, no WriteOnly/Sensitive": {
			attrName:                     "name",
			specComputedOptionalRequired: "required",
			wantWriteOnly:                false,
			wantSensitive:                false,
			wantRequired:                 true,
			wantOptional:                 false,
		},
		"sync_token excluded from heuristic -> stays as-is": {
			attrName:                     "sync_token",
			specComputedOptionalRequired: "computed",
			wantWriteOnly:                false,
			wantSensitive:                false,
			wantRequired:                 false,
			wantOptional:                 false,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, err := NewGeneratorStringAttribute(tc.attrName, &resource.StringAttribute{
				ComputedOptionalRequired: specschema.ComputedOptionalRequired(tc.specComputedOptionalRequired),
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			schema, err := got.Schema(generatorschema.FrameworkIdentifier(tc.attrName))
			if err != nil {
				t.Fatalf("unexpected schema error: %v", err)
			}

			hasWriteOnly := strings.Contains(schema, "WriteOnly: true")
			if hasWriteOnly != tc.wantWriteOnly {
				t.Errorf("WriteOnly: got %v, want %v\nschema:\n%s", hasWriteOnly, tc.wantWriteOnly, schema)
			}

			hasSensitive := strings.Contains(schema, "Sensitive: true")
			if hasSensitive != tc.wantSensitive {
				t.Errorf("Sensitive: got %v, want %v\nschema:\n%s", hasSensitive, tc.wantSensitive, schema)
			}

			hasOptional := strings.Contains(schema, "Optional: true")
			if hasOptional != tc.wantOptional {
				t.Errorf("Optional: got %v, want %v\nschema:\n%s", hasOptional, tc.wantOptional, schema)
			}

			hasRequired := strings.Contains(schema, "Required: true")
			if hasRequired != tc.wantRequired {
				t.Errorf("Required: got %v, want %v\nschema:\n%s", hasRequired, tc.wantRequired, schema)
			}
		})
	}
}
