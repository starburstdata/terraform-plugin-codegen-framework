// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package schema

import (
	"fmt"
	"sort"
)

// maxTypeKeyRounds bounds the renaming loop in DisambiguateTypeKeys.
const maxTypeKeyRounds = 10

// TypeKeyer is implemented by nested attributes that generate a custom Type and Value. The type key is the
// snake_case identifier the generated Go type names are derived from. It defaults to the attribute name,
// so two nested attributes with the same name but different shapes would generate clashing Go types.
type TypeKeyer interface {
	GeneratorAttribute
	Attributes

	// TypeKey returns the override, or an empty string if the attribute name is used.
	TypeKey() string

	// WithTypeKey returns a copy of the attribute that derives its generated Go type names from key.
	WithTypeKey(key string) GeneratorAttribute
}

// TypeKeyOf returns the identifier the generated Go type names of a nested attribute are derived from.
func TypeKeyOf(a GeneratorAttribute, name FrameworkIdentifier) FrameworkIdentifier {
	if t, ok := a.(TypeKeyer); ok && t.TypeKey() != "" {
		return FrameworkIdentifier(t.TypeKey())
	}

	return name
}

type typeKeyEntry struct {
	container GeneratorAttributes
	name      string
	attr      TypeKeyer
	key       string
	parentKey string
}

// DisambiguateTypeKeys gives nested attributes that share a name, but would generate different Go types,
// distinct type keys. Attributes that generate identical types keep sharing one key, and the first attribute
// of a clashing group (in sorted depth-first order) keeps the plain name, so existing output is unchanged
// unless a clash exists. The attributes are modified in place.
func DisambiguateTypeKeys(attrs GeneratorAttributes) error {
	for round := 0; round < maxTypeKeyRounds; round++ {
		var entries []typeKeyEntry

		collectTypeKeyEntries(attrs, "", &entries)

		byKey := make(map[string][]typeKeyEntry, len(entries))
		var keys []string

		for _, e := range entries {
			if _, ok := byKey[e.key]; !ok {
				keys = append(keys, e.key)
			}

			byKey[e.key] = append(byKey[e.key], e)
		}

		changed := false

		for _, key := range keys {
			group := byKey[key]

			if len(group) < 2 {
				continue
			}

			first, err := typeFingerprint(group[0].attr)

			if err != nil {
				return err
			}

			for _, e := range group[1:] {
				fp, err := typeFingerprint(e.attr)

				if err != nil {
					return err
				}

				if fp == first {
					continue
				}

				newKey := e.key + "_" + e.name

				if e.parentKey != "" {
					newKey = e.parentKey + "_" + e.name
				}

				if newKey == e.key {
					return fmt.Errorf("unable to generate a unique type name for nested attribute %q", e.name)
				}

				e.container[e.name] = e.attr.WithTypeKey(newKey)
				changed = true
			}
		}

		if !changed {
			return nil
		}
	}

	return fmt.Errorf("nested attribute type names did not converge after %d rounds", maxTypeKeyRounds)
}

func collectTypeKeyEntries(attrs GeneratorAttributes, parentKey string, entries *[]typeKeyEntry) {
	for _, name := range attrs.SortedKeys() {
		t, ok := attrs[name].(TypeKeyer)

		if !ok {
			continue
		}

		key := string(TypeKeyOf(t, FrameworkIdentifier(name)))

		*entries = append(*entries, typeKeyEntry{
			container: attrs,
			name:      name,
			attr:      t,
			key:       key,
			parentKey: parentKey,
		})

		collectTypeKeyEntries(t.GetAttributes(), key, entries)
	}
}

// typeFingerprint captures everything that determines the generated Type and Value code of a nested attribute.
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

	names := attrs.SortedKeys()
	sort.Strings(names)

	return fmt.Sprintf("%v|%v|%v|%v|%v", names, attributeTypes, attrTypes, attrValues, collectionTypes), nil
}
