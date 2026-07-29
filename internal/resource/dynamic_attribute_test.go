// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package resource

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-codegen-spec/resource"
	specschema "github.com/hashicorp/terraform-plugin-codegen-spec/schema"

	"github.com/starburstdata/terraform-plugin-codegen-framework/internal/convert"
	"github.com/starburstdata/terraform-plugin-codegen-framework/internal/model"
)

func TestGeneratorDynamicAttribute_New(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		input         *resource.DynamicAttribute
		expected      GeneratorDynamicAttribute
		expectedError error
	}{
		"nil": {
			input:         nil,
			expectedError: nil,
		},
		"computed": {
			input: &resource.DynamicAttribute{
				ComputedOptionalRequired: "computed",
			},
			expected: GeneratorDynamicAttribute{
				ComputedOptionalRequired: convert.NewComputedOptionalRequired(specschema.Computed),
				CustomType:               convert.NewCustomTypePrimitive(nil, nil, "name"),
				DeprecationMessage:       convert.NewDeprecationMessage(nil),
				Description:              convert.NewDescription(nil),
				PlanModifiers:            convert.NewPlanModifiers(convert.PlanModifierTypeDynamic, nil),
				Sensitive:                convert.NewSensitive(nil),
				Validators:               convert.NewValidators(convert.ValidatorTypeDynamic, nil),
			},
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, err := NewGeneratorDynamicAttribute("name", testCase.input)

			if testCase.input == nil {
				if err == nil {
					t.Fatalf("expected error for nil input, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}

			if diff := cmp.Diff(got, testCase.expected); diff != "" {
				t.Errorf("unexpected difference: %s", diff)
			}
		})
	}
}

func TestGeneratorDynamicAttribute_Schema(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		input         GeneratorDynamicAttribute
		expected      string
		expectedError error
	}{
		"computed": {
			input: GeneratorDynamicAttribute{
				ComputedOptionalRequired: convert.NewComputedOptionalRequired(specschema.Computed),
			},
			expected: `"dynamic_attribute": schema.DynamicAttribute{
Computed: true,
},`,
		},
		"optional": {
			input: GeneratorDynamicAttribute{
				ComputedOptionalRequired: convert.NewComputedOptionalRequired(specschema.Optional),
			},
			expected: `"dynamic_attribute": schema.DynamicAttribute{
Optional: true,
},`,
		},
		"required": {
			input: GeneratorDynamicAttribute{
				ComputedOptionalRequired: convert.NewComputedOptionalRequired(specschema.Required),
			},
			expected: `"dynamic_attribute": schema.DynamicAttribute{
Required: true,
},`,
		},
		"sensitive": {
			input: GeneratorDynamicAttribute{
				Sensitive: convert.NewSensitive(pointer(true)),
			},
			expected: `"dynamic_attribute": schema.DynamicAttribute{
Sensitive: true,
},`,
		},
		"description": {
			input: GeneratorDynamicAttribute{
				Description: convert.NewDescription(pointer("description")),
			},
			expected: `"dynamic_attribute": schema.DynamicAttribute{
Description: "description",
MarkdownDescription: "description",
},`,
		},
		"deprecation-message": {
			input: GeneratorDynamicAttribute{
				DeprecationMessage: convert.NewDeprecationMessage(pointer("deprecated")),
			},
			expected: `"dynamic_attribute": schema.DynamicAttribute{
DeprecationMessage: "deprecated",
},`,
		},
		"custom-type": {
			input: GeneratorDynamicAttribute{
				CustomType: convert.NewCustomTypePrimitive(
					&specschema.CustomType{
						Type: "my_custom_type",
					},
					nil,
					"dynamic_attribute",
				),
			},
			expected: `"dynamic_attribute": schema.DynamicAttribute{
CustomType: my_custom_type,
},`,
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, err := testCase.input.Schema("dynamic_attribute")

			if diff := cmp.Diff(err, testCase.expectedError, equateErrorMessage); diff != "" {
				t.Errorf("unexpected error: %s", diff)
			}

			if diff := cmp.Diff(got, testCase.expected); diff != "" {
				t.Errorf("unexpected difference: %s", diff)
			}
		})
	}
}

func TestGeneratorDynamicAttribute_ModelField(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		input         GeneratorDynamicAttribute
		expected      model.Field
		expectedError error
	}{
		"default": {
			expected: model.Field{
				Name:      "DynamicAttribute",
				ValueType: "types.Dynamic",
				TfsdkName: "dynamic_attribute",
			},
		},
		"custom-type": {
			input: GeneratorDynamicAttribute{
				CustomType: convert.NewCustomTypePrimitive(
					&specschema.CustomType{
						ValueType: "my_custom_value_type",
					},
					nil,
					"",
				),
			},
			expected: model.Field{
				Name:      "DynamicAttribute",
				ValueType: "my_custom_value_type",
				TfsdkName: "dynamic_attribute",
			},
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, err := testCase.input.ModelField("dynamic_attribute")

			if diff := cmp.Diff(err, testCase.expectedError, equateErrorMessage); diff != "" {
				t.Errorf("unexpected error: %s", diff)
			}

			if diff := cmp.Diff(got, testCase.expected); diff != "" {
				t.Errorf("unexpected difference: %s", diff)
			}
		})
	}
}
