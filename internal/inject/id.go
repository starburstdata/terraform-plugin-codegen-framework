// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

// Package inject provides Starburst-fork-only generator extensions that
// mutate a parsed spec.Specification prior to code emission.
package inject

import (
	"log/slog"

	codegenresource "github.com/hashicorp/terraform-plugin-codegen-spec/resource"
	"github.com/hashicorp/terraform-plugin-codegen-spec/schema"
	"github.com/hashicorp/terraform-plugin-codegen-spec/spec"
)

const idAttributeDescription = "Terraform import identifier."

// ID mutates s in place, prepending a Computed string attribute named "id"
// to every resource schema that does not already declare an "id" attribute.
// Resources that already declare "id" are left untouched.
func ID(s *spec.Specification, logger *slog.Logger) {
	for i := range s.Resources {
		r := &s.Resources[i]
		if r.Schema == nil {
			continue
		}
		if hasAttribute(r.Schema.Attributes, "id") {
			logger.Debug("skipping id injection; resource already declares id", "resource", r.Name)
			continue
		}

		description := idAttributeDescription
		idAttr := codegenresource.Attribute{
			Name: "id",
			String: &codegenresource.StringAttribute{
				ComputedOptionalRequired: schema.Computed,
				Description:              &description,
			},
		}

		r.Schema.Attributes = append(codegenresource.Attributes{idAttr}, r.Schema.Attributes...)
	}
}

func hasAttribute(attrs codegenresource.Attributes, name string) bool {
	for _, a := range attrs {
		if a.Name == name {
			return true
		}
	}
	return false
}
