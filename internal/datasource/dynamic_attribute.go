// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package datasource

import (
	"bytes"
	"fmt"

	"github.com/hashicorp/terraform-plugin-codegen-spec/datasource"

	"github.com/starburstdata/terraform-plugin-codegen-framework/internal/convert"
	"github.com/starburstdata/terraform-plugin-codegen-framework/internal/model"
	"github.com/starburstdata/terraform-plugin-codegen-framework/internal/schema"
)

type GeneratorDynamicAttribute struct {
	AssociatedExternalType   *schema.AssocExtType
	ComputedOptionalRequired convert.ComputedOptionalRequired
	CustomType               convert.CustomTypePrimitive
	DeprecationMessage       convert.DeprecationMessage
	Description              convert.Description
	Sensitive                convert.Sensitive
	Validators               convert.Validators
}

func NewGeneratorDynamicAttribute(name string, a *datasource.DynamicAttribute) (GeneratorDynamicAttribute, error) {
	if a == nil {
		return GeneratorDynamicAttribute{}, fmt.Errorf("*datasource.DynamicAttribute is nil")
	}

	c := convert.NewComputedOptionalRequired(a.ComputedOptionalRequired)

	ctp := convert.NewCustomTypePrimitive(a.CustomType, a.AssociatedExternalType, name)

	d := convert.NewDescription(a.Description)

	dm := convert.NewDeprecationMessage(a.DeprecationMessage)

	s := convert.NewSensitive(a.Sensitive)

	v := convert.NewValidators(convert.ValidatorTypeDynamic, a.Validators.CustomValidators())

	return GeneratorDynamicAttribute{
		AssociatedExternalType:   schema.NewAssocExtType(a.AssociatedExternalType),
		ComputedOptionalRequired: c,
		CustomType:               ctp,
		DeprecationMessage:       dm,
		Description:              d,
		Sensitive:                s,
		Validators:               v,
	}, nil
}

func (g GeneratorDynamicAttribute) GeneratorSchemaType() schema.Type {
	return schema.GeneratorDynamicAttribute
}

func (g GeneratorDynamicAttribute) Imports() *schema.Imports {
	imports := schema.NewImports()

	imports.Append(g.CustomType.Imports())

	imports.Append(g.Validators.Imports())

	if g.AssociatedExternalType != nil {
		imports.Append(schema.AssociatedExternalTypeImports())
	}

	imports.Append(g.AssociatedExternalType.Imports())

	return imports
}

func (g GeneratorDynamicAttribute) Equal(ga schema.GeneratorAttribute) bool {
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

	if !g.DeprecationMessage.Equal(h.DeprecationMessage) {
		return false
	}

	if !g.Description.Equal(h.Description) {
		return false
	}

	if !g.Sensitive.Equal(h.Sensitive) {
		return false
	}

	return g.Validators.Equal(h.Validators)
}

func (g GeneratorDynamicAttribute) Schema(name schema.FrameworkIdentifier) (string, error) {
	var b bytes.Buffer

	b.WriteString(fmt.Sprintf("%q: schema.DynamicAttribute{\n", name))
	b.Write(g.CustomType.Schema())
	b.Write(g.ComputedOptionalRequired.Schema())
	b.Write(g.Sensitive.Schema())
	b.Write(g.Description.Schema())
	b.Write(g.DeprecationMessage.Schema())
	b.Write(g.Validators.Schema())
	b.WriteString("},")

	return b.String(), nil
}

func (g GeneratorDynamicAttribute) ModelField(name schema.FrameworkIdentifier) (model.Field, error) {
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
func (g GeneratorDynamicAttribute) AttrType(name schema.FrameworkIdentifier) (string, error) {
	if g.AssociatedExternalType != nil {
		return fmt.Sprintf("%sType{}", name.ToPascalCase()), nil
	}

	return "basetypes.DynamicType{}", nil
}

// AttrValue returns a string representation of a basetypes.DynamicValuable type.
func (g GeneratorDynamicAttribute) AttrValue(name schema.FrameworkIdentifier) string {
	if g.AssociatedExternalType != nil {
		return fmt.Sprintf("%sValue", name.ToPascalCase())
	}

	return "basetypes.DynamicValue"
}
