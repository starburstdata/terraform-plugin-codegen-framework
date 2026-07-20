// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package inject_test

import (
	"io"
	"log/slog"
	"testing"

	codegenresource "github.com/hashicorp/terraform-plugin-codegen-spec/resource"
	"github.com/hashicorp/terraform-plugin-codegen-spec/schema"
	"github.com/hashicorp/terraform-plugin-codegen-spec/spec"

	"github.com/starburstdata/terraform-plugin-codegen-framework/internal/inject"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func ptr(s string) *string { return &s }

func TestID_InjectsWhenMissing(t *testing.T) {
	t.Parallel()

	s := &spec.Specification{
		Resources: codegenresource.Resources{
			{
				Name: "cluster",
				Schema: &codegenresource.Schema{
					Attributes: codegenresource.Attributes{
						{
							Name: "cluster_id",
							String: &codegenresource.StringAttribute{
								ComputedOptionalRequired: schema.Computed,
							},
						},
					},
				},
			},
			{
				Name: "catalog",
				Schema: &codegenresource.Schema{
					Attributes: codegenresource.Attributes{
						{
							Name: "catalog_id",
							String: &codegenresource.StringAttribute{
								ComputedOptionalRequired: schema.Computed,
							},
						},
					},
				},
			},
		},
	}

	inject.ID(s, discardLogger())

	for _, r := range s.Resources {
		if len(r.Schema.Attributes) != 2 {
			t.Fatalf("resource %q: expected 2 attributes, got %d", r.Name, len(r.Schema.Attributes))
		}

		first := r.Schema.Attributes[0]
		if first.Name != "id" {
			t.Fatalf("resource %q: expected first attribute to be %q, got %q", r.Name, "id", first.Name)
		}
		if first.String == nil {
			t.Fatalf("resource %q: expected injected id attribute to be a string attribute", r.Name)
		}
		if first.String.ComputedOptionalRequired != schema.Computed {
			t.Fatalf("resource %q: expected injected id to be computed, got %q", r.Name, first.String.ComputedOptionalRequired)
		}
		if first.String.Description == nil || *first.String.Description != "Terraform import identifier." {
			t.Fatalf("resource %q: unexpected id description: %v", r.Name, first.String.Description)
		}
	}
}

func TestID_PreservesExistingIDAttribute(t *testing.T) {
	t.Parallel()

	s := &spec.Specification{
		Resources: codegenresource.Resources{
			{
				Name: "custom",
				Schema: &codegenresource.Schema{
					Attributes: codegenresource.Attributes{
						{
							Name: "id",
							String: &codegenresource.StringAttribute{
								ComputedOptionalRequired: schema.Required,
								Description:              ptr("custom author-defined id"),
							},
						},
					},
				},
			},
		},
	}

	inject.ID(s, discardLogger())

	r := s.Resources[0]
	if len(r.Schema.Attributes) != 1 {
		t.Fatalf("expected attribute count to remain 1, got %d", len(r.Schema.Attributes))
	}

	attr := r.Schema.Attributes[0]
	if attr.String.ComputedOptionalRequired != schema.Required {
		t.Fatalf("expected existing id to remain required, got %q", attr.String.ComputedOptionalRequired)
	}
	if attr.String.Description == nil || *attr.String.Description != "custom author-defined id" {
		t.Fatalf("expected existing id description to be untouched, got %v", attr.String.Description)
	}
}

func TestID_MixedResources(t *testing.T) {
	t.Parallel()

	s := &spec.Specification{
		Resources: codegenresource.Resources{
			{
				Name: "first",
				Schema: &codegenresource.Schema{
					Attributes: codegenresource.Attributes{
						{Name: "name", String: &codegenresource.StringAttribute{ComputedOptionalRequired: schema.Required}},
					},
				},
			},
			{
				Name: "middle_has_id",
				Schema: &codegenresource.Schema{
					Attributes: codegenresource.Attributes{
						{Name: "id", String: &codegenresource.StringAttribute{ComputedOptionalRequired: schema.Required}},
					},
				},
			},
			{
				Name: "last",
				Schema: &codegenresource.Schema{
					Attributes: codegenresource.Attributes{
						{Name: "name", String: &codegenresource.StringAttribute{ComputedOptionalRequired: schema.Required}},
					},
				},
			},
		},
	}

	inject.ID(s, discardLogger())

	if len(s.Resources[0].Schema.Attributes) != 2 || s.Resources[0].Schema.Attributes[0].Name != "id" {
		t.Fatalf("expected first resource to gain an injected id attribute, got %+v", s.Resources[0].Schema.Attributes)
	}
	if len(s.Resources[1].Schema.Attributes) != 1 {
		t.Fatalf("expected middle resource's existing id to be untouched, got %+v", s.Resources[1].Schema.Attributes)
	}
	if s.Resources[1].Schema.Attributes[0].String.ComputedOptionalRequired != schema.Required {
		t.Fatalf("expected middle resource's id to remain required")
	}
	if len(s.Resources[2].Schema.Attributes) != 2 || s.Resources[2].Schema.Attributes[0].Name != "id" {
		t.Fatalf("expected last resource to gain an injected id attribute, got %+v", s.Resources[2].Schema.Attributes)
	}
}

func TestID_NilSchemaIsNoOp(t *testing.T) {
	t.Parallel()

	s := &spec.Specification{
		Resources: codegenresource.Resources{
			{Name: "no_schema", Schema: nil},
		},
	}

	inject.ID(s, discardLogger())

	if s.Resources[0].Schema != nil {
		t.Fatalf("expected schema to remain nil, got %+v", s.Resources[0].Schema)
	}
}

func TestID_EmptyResourcesIsNoOp(t *testing.T) {
	t.Parallel()

	s := &spec.Specification{
		Resources: codegenresource.Resources{},
	}

	inject.ID(s, discardLogger())

	if len(s.Resources) != 0 {
		t.Fatalf("expected no resources, got %d", len(s.Resources))
	}
}
