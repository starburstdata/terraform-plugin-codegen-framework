// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package schema

import "fmt"

// TypeKeyer separates generated Go type names from Terraform attribute names.
type TypeKeyer interface {
	GeneratorAttribute
	Attributes

	TypeKey() string
	WithTypeKey(key string) GeneratorAttribute
}

// TypeKeyOf returns the identifier the generated Go type names of a nested attribute are derived from.
func TypeKeyOf(a GeneratorAttribute, name FrameworkIdentifier) FrameworkIdentifier {
	if t, ok := a.(TypeKeyer); ok && t.TypeKey() != "" {
		return FrameworkIdentifier(t.TypeKey())
	}

	return name
}

// DisambiguateTypeKeys assigns unique Go names to different shapes without changing Terraform keys.
func DisambiguateTypeKeys(attrs GeneratorAttributes) error {
	names := typeNames{
		reserved: make(map[string]bool),
		used:     make(map[string]bool),
		shapes:   make(map[string]map[string]string),
	}
	names.reserve(attrs)
	return names.assign(attrs, "")
}

type typeNames struct {
	reserved map[string]bool
	used     map[string]bool
	shapes   map[string]map[string]string
}

func (n *typeNames) reserve(attrs GeneratorAttributes) {
	for _, name := range attrs.SortedKeys() {
		t, ok := attrs[name].(TypeKeyer)
		if !ok {
			continue
		}

		n.reserved[TypeKeyOf(t, FrameworkIdentifier(name)).ToPascalCase()] = true
		n.reserve(t.GetAttributes())
	}
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

		key := TypeKeyOf(t, FrameworkIdentifier(name)).ToString()
		goName := FrameworkIdentifier(key).ToPascalCase()
		if n.shapes[goName] == nil {
			n.shapes[goName] = make(map[string]string)
		}
		if shared, ok := n.shapes[goName][fingerprint]; ok {
			attrs[name] = t.WithTypeKey(shared)
			continue
		}
		if n.used[goName] {
			key = path
			for suffix := 2; n.used[FrameworkIdentifier(key).ToPascalCase()] || n.reserved[FrameworkIdentifier(key).ToPascalCase()]; suffix++ {
				key = fmt.Sprintf("%s_%d", path, suffix)
			}
		}
		n.shapes[goName][fingerprint] = key
		n.used[FrameworkIdentifier(key).ToPascalCase()] = true
		attrs[name] = t.WithTypeKey(key)
	}
	return nil
}

// typeFingerprint describes generated value fields and their resolved types.
func typeFingerprint(t TypeKeyer) (string, error) {
	attrs := t.GetAttributes()

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

	return fmt.Sprintf("%v|%v|%v|%v", attributeTypes, attrTypes, attrValues, collectionTypes), nil
}
