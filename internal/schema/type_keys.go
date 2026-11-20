// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package schema

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
)

// TypeKeyer separates generated Go type names from Terraform attribute names.
type TypeKeyer interface {
	GeneratorAttribute
	Attributes

	TypeKey() string
	WithTypeKey(key string) GeneratorAttribute
}

type BlockTypeKeyer interface {
	GeneratorBlock
	Blocks

	TypeKey() string
	WithTypeKey(key string) GeneratorBlock
}

// TypeKeyOf returns the identifier the generated Go type names of a nested attribute are derived from.
func TypeKeyOf(a any, name FrameworkIdentifier) FrameworkIdentifier {
	if t, ok := a.(interface{ TypeKey() string }); ok && t.TypeKey() != "" {
		return FrameworkIdentifier(t.TypeKey())
	}

	return name
}

// NestedTypeNames keeps resolved Go names separate from attribute and block keys.
func NestedTypeNames(attrs GeneratorAttributes, blocks GeneratorBlocks) map[string]string {
	names := attrs.TypeNames()
	for name, block := range blocks {
		names[name] = TypeKeyOf(block, FrameworkIdentifier(name)).ToString()
	}
	return names
}

// DisambiguateTypeKeys assigns unique Go names to different shapes without changing Terraform keys.
func DisambiguateTypeKeys(attrs GeneratorAttributes, blocks GeneratorBlocks) error {
	names := typeNames{
		reserved: make(map[string]bool),
		used:     make(map[string]bool),
		shapes:   make(map[string]map[string]string),
		blocks:   make(map[string]bool),
	}
	if err := names.reserve(attrs); err != nil {
		return err
	}
	if err := names.reserveBlocks(blocks); err != nil {
		return err
	}
	if err := names.assignBlocks(blocks, ""); err != nil {
		return err
	}
	return names.assign(attrs, "")
}

type typeNames struct {
	reserved map[string]bool
	used     map[string]bool
	shapes   map[string]map[string]string
	blocks   map[string]bool
}

func (n *typeNames) reserve(attrs GeneratorAttributes) error {
	for _, name := range attrs.SortedKeys() {
		t, ok := attrs[name].(TypeKeyer)
		if !ok {
			if generator, ok := attrs[name].(CustomTypeAndValue); ok {
				code, err := generator.CustomTypeAndValue(name)
				if err != nil {
					return err
				}
				if err := n.reserveDeclarations(code); err != nil {
					return err
				}
			}
			continue
		}

		for _, symbol := range nestedTypeSymbols(TypeKeyOf(t, FrameworkIdentifier(name)).ToString()) {
			n.reserved[symbol] = true
		}
		if err := n.reserve(t.GetAttributes()); err != nil {
			return err
		}
	}
	return nil
}

func (n *typeNames) reserveBlocks(blocks GeneratorBlocks) error {
	for _, name := range blocks.SortedKeys() {
		block, ok := blocks[name].(BlockTypeKeyer)
		if !ok {
			return fmt.Errorf("nested block %q does not support type naming", name)
		}
		key := TypeKeyOf(block, FrameworkIdentifier(name)).ToString()
		for _, symbol := range nestedTypeSymbols(key) {
			n.reserved[symbol] = true
			n.blocks[symbol] = true
		}
		if err := n.reserve(block.GetAttributes()); err != nil {
			return err
		}
		if err := n.reserveBlocks(block.GetBlocks()); err != nil {
			return err
		}
	}
	return nil
}

// Fixed-name custom declarations share the namespace with allocated nested types.
func (n *typeNames) reserveDeclarations(code []byte) error {
	if len(code) == 0 {
		return nil
	}
	file, err := parser.ParseFile(token.NewFileSet(), "", append([]byte("package generated\n"), code...), 0)
	if err != nil {
		return err
	}
	reserve := func(name string) {
		n.reserved[name] = true
		n.used[name] = true
	}
	for _, declaration := range file.Decls {
		switch declaration := declaration.(type) {
		case *ast.FuncDecl:
			if declaration.Recv == nil {
				reserve(declaration.Name.Name)
			}
		case *ast.GenDecl:
			for _, spec := range declaration.Specs {
				if ty, ok := spec.(*ast.TypeSpec); ok {
					reserve(ty.Name.Name)
				}
			}
		}
	}
	return nil
}

func nestedTypeSymbols(key string) []string {
	name := FrameworkIdentifier(key).ToPascalCase()
	return []string{
		name + "Type", name + "Value",
		"New" + name + "Value", "New" + name + "ValueNull",
		"New" + name + "ValueUnknown", "New" + name + "ValueMust",
	}
}

func symbolsConflict(key string, symbols map[string]bool) bool {
	for _, symbol := range nestedTypeSymbols(key) {
		if symbols[symbol] {
			return true
		}
	}
	return false
}

func (n *typeNames) assignBlocks(blocks GeneratorBlocks, parentPath string) error {
	for _, name := range blocks.SortedKeys() {
		path := name
		if parentPath != "" {
			path = parentPath + "_" + name
		}
		block, ok := blocks[name].(BlockTypeKeyer)
		if !ok {
			return fmt.Errorf("nested block %q does not support type naming", name)
		}
		if err := n.assign(block.GetAttributes(), path); err != nil {
			return err
		}
		if err := n.assignBlocks(block.GetBlocks(), path); err != nil {
			return err
		}
		fingerprint, err := generatedTypeFingerprint(block.GetAttributes(), block.GetBlocks())
		if err != nil {
			return err
		}
		key := n.allocate(TypeKeyOf(block, FrameworkIdentifier(name)).ToString(), path, fingerprint, nil)
		blocks[name] = block.WithTypeKey(key)
	}
	return nil
}

func (n *typeNames) assign(attrs GeneratorAttributes, parentPath string) error {
	for _, name := range attrs.SortedKeys() {
		t, ok := attrs[name].(TypeKeyer)
		if !ok {
			continue
		}

		path := name
		if parentPath != "" {
			path = parentPath + "_" + name
		}
		// Resolve children first so parent fingerprints include their final type names.
		if err := n.assign(t.GetAttributes(), path); err != nil {
			return err
		}
		fingerprint, err := typeFingerprint(t)
		if err != nil {
			return err
		}

		key := n.allocate(TypeKeyOf(t, FrameworkIdentifier(name)).ToString(), path, fingerprint, n.blocks)
		attrs[name] = t.WithTypeKey(key)
	}
	return nil
}

func (n *typeNames) allocate(key, path, fingerprint string, reservedDefault map[string]bool) string {
	goName := FrameworkIdentifier(key).ToPascalCase()
	if n.shapes[goName] == nil {
		n.shapes[goName] = make(map[string]string)
	}
	if shared, ok := n.shapes[goName][fingerprint]; ok {
		return shared
	}
	if symbolsConflict(key, n.used) || symbolsConflict(key, reservedDefault) {
		key = path
		for suffix := 2; symbolsConflict(key, n.used) || symbolsConflict(key, n.reserved); suffix++ {
			key = fmt.Sprintf("%s_%d", path, suffix)
		}
	}
	n.shapes[goName][fingerprint] = key
	for _, symbol := range nestedTypeSymbols(key) {
		n.used[symbol] = true
	}
	return key
}

// typeFingerprint describes generated value fields and their resolved types.
func typeFingerprint(t TypeKeyer) (string, error) {
	return generatedTypeFingerprint(t.GetAttributes(), nil)
}

func generatedTypeFingerprint(attrs GeneratorAttributes, blocks GeneratorBlocks) (string, error) {
	attributeTypes, err := attrs.AttributeTypes()

	if err != nil {
		return "", err
	}

	attrTypes, err := attrs.AttrTypes()

	if err != nil {
		return "", err
	}

	attrValues, err := attrs.AttrValues()

	if err != nil {
		return "", err
	}

	collectionTypes, err := attrs.CollectionTypes()

	if err != nil {
		return "", err
	}

	blockTypes, err := blocks.BlockTypes()
	if err != nil {
		return "", err
	}
	for name, ty := range blockTypes {
		attributeTypes[name] = ty
	}
	blockAttrTypes, err := blocks.AttrTypes()
	if err != nil {
		return "", err
	}
	for name, ty := range blockAttrTypes {
		attrTypes[name] = ty
	}
	blockAttrValues, err := blocks.AttrValues()
	if err != nil {
		return "", err
	}
	for name, value := range blockAttrValues {
		attrValues[name] = value
	}

	return fmt.Sprintf("%v|%v|%v|%v", attributeTypes, attrTypes, attrValues, collectionTypes), nil
}
