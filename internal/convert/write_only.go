// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package convert

type WriteOnly struct {
	writeOnly *bool
}

func NewWriteOnly(w *bool) WriteOnly {
	return WriteOnly{
		writeOnly: w,
	}
}

func (w WriteOnly) Equal(other WriteOnly) bool {
	if w.writeOnly == nil && other.writeOnly == nil {
		return true
	}

	if w.writeOnly == nil || other.writeOnly == nil {
		return false
	}

	return *w.writeOnly == *other.writeOnly
}

func (w WriteOnly) IsWriteOnly() bool {
	if w.writeOnly == nil {
		return false
	}

	return *w.writeOnly
}

func (w WriteOnly) Schema() []byte {
	if w.IsWriteOnly() {
		return []byte("WriteOnly: true,\n")
	}

	return nil
}
