// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package resource

import (
	"bytes"
	"fmt"

	"github.com/hashicorp/terraform-plugin-codegen-spec/resource"

	"github.com/starburstdata/terraform-plugin-codegen-framework/internal/convert"
	"github.com/starburstdata/terraform-plugin-codegen-framework/internal/model"
	generatorschema "github.com/starburstdata/terraform-plugin-codegen-framework/internal/schema"
)

type GeneratorDynamicAttribute struct {
	AssociatedExternalType   *generatorschema.AssocExtType
	ComputedOptionalRequired convert.ComputedOptionalRequired
	CustomType               convert.CustomTypePrimitive
	Default                  convert.DefaultCustom
	DeprecationMessage       convert.DeprecationMessage
	Description              convert.Description
	PlanModifiers            convert.PlanModifiers
	Sensitive                convert.Sensitive
	Validators               convert.Validators
}

func NewGeneratorDynamicAttribute(name string, a *resource.DynamicAttribute) (GeneratorDynamicAttribute, error) {
	if a == nil {
		return GeneratorDynamicAttribute{}, fmt.Errorf("*resource.DynamicAttribute is nil")
	}

	// Dynamic attributes have no generated custom type/value or to/from functions, so an
	// associated external type would reference undefined identifiers.
	if a.AssociatedExternalType != nil {
		return GeneratorDynamicAttribute{}, fmt.Errorf("associated_external_type is not supported on dynamic attribute %q", name)
	}

	c := convert.NewComputedOptionalRequired(a.ComputedOptionalRequired)

	ctp := convert.NewCustomTypePrimitive(a.CustomType, a.AssociatedExternalType, name)

	var dc convert.DefaultCustom
	if a.Default != nil {
		dc = convert.NewDefaultCustom(a.Default.Custom)
	}

	dm := convert.NewDeprecationMessage(a.DeprecationMessage)

	d := convert.NewDescription(a.Description)

	pm := convert.NewPlanModifiers(convert.PlanModifierTypeDynamic, a.PlanModifiers.CustomPlanModifiers())

	s := convert.NewSensitive(a.Sensitive)

	v := convert.NewValidators(convert.ValidatorTypeDynamic, a.Validators.CustomValidators())

	return GeneratorDynamicAttribute{
		AssociatedExternalType:   generatorschema.NewAssocExtType(a.AssociatedExternalType),
		ComputedOptionalRequired: c,
		CustomType:               ctp,
		Default:                  dc,
		DeprecationMessage:       dm,
		Description:              d,
		PlanModifiers:            pm,
		Sensitive:                s,
		Validators:               v,
	}, nil
}

func (g GeneratorDynamicAttribute) GeneratorSchemaType() generatorschema.Type {
	return generatorschema.GeneratorDynamicAttribute
}

func (g GeneratorDynamicAttribute) Imports() *generatorschema.Imports {
	imports := generatorschema.NewImports()

	imports.Append(g.CustomType.Imports())

	imports.Append(g.Default.Imports())

	imports.Append(g.PlanModifiers.Imports())

	imports.Append(g.Validators.Imports())

	if g.AssociatedExternalType != nil {
		imports.Append(generatorschema.AssociatedExternalTypeImports())
	}

	imports.Append(g.AssociatedExternalType.Imports())

	return imports
}

func (g GeneratorDynamicAttribute) Equal(ga generatorschema.GeneratorAttribute) bool {
	h, ok := ga.(GeneratorDynamicAttribute)

	if !ok {
		return false
	}

	if !g.AssociatedExternalType.Equal(h.AssociatedExternalType) {
		return false
	}

	if !g.ComputedOptionalRequired.Equal(h.ComputedOptionalRequired) {
		return false
	}

	if !g.CustomType.Equal(h.CustomType) {
		return false
	}

	if !g.Default.Equal(h.Default) {
		return false
	}

	if !g.DeprecationMessage.Equal(h.DeprecationMessage) {
		return false
	}

	if !g.Description.Equal(h.Description) {
		return false
	}

	if !g.PlanModifiers.Equal(h.PlanModifiers) {
		return false
	}

	if !g.Sensitive.Equal(h.Sensitive) {
		return false
	}

	return g.Validators.Equal(h.Validators)
}

func (g GeneratorDynamicAttribute) Schema(name generatorschema.FrameworkIdentifier) (string, error) {
	var b bytes.Buffer

	b.WriteString(fmt.Sprintf("%q: schema.DynamicAttribute{\n", name))
	b.Write(g.CustomType.Schema())
	b.Write(g.ComputedOptionalRequired.Schema())
	b.Write(g.Sensitive.Schema())
	b.Write(g.Description.Schema())
	b.Write(g.DeprecationMessage.Schema())
	b.Write(g.PlanModifiers.Schema())
	b.Write(g.Validators.Schema())
	b.Write(g.Default.Schema())
	b.WriteString("},")

	return b.String(), nil
}

func (g GeneratorDynamicAttribute) ModelField(name generatorschema.FrameworkIdentifier) (model.Field, error) {
	field := model.Field{
		Name:      name.ToPascalCase(),
		TfsdkName: name.ToString(),
		ValueType: model.DynamicValueType,
	}

	customValueType := g.CustomType.ValueType()

	if customValueType != "" {
		field.ValueType = customValueType
	}

	return field, nil
}

// AttrType returns a string representation of a basetypes.DynamicTypable type.
func (g GeneratorDynamicAttribute) AttrType(name generatorschema.FrameworkIdentifier) (string, error) {
	if g.AssociatedExternalType != nil {
		return fmt.Sprintf("%sType{}", name.ToPascalCase()), nil
	}

	return "basetypes.DynamicType{}", nil
}

// AttrValue returns a string representation of a basetypes.DynamicValuable type.
func (g GeneratorDynamicAttribute) AttrValue(name generatorschema.FrameworkIdentifier) string {
	if g.AssociatedExternalType != nil {
		return fmt.Sprintf("%sValue", name.ToPascalCase())
	}

	return "basetypes.DynamicValue"
}
