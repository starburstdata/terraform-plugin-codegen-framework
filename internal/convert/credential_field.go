// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package convert

import "strings"

// IsCredentialField reports whether an attribute name matches a write-only
// credential pattern. Resource string attributes matching this set are emitted
// as WriteOnly + Sensitive + Optional in the generated schema, so the value
// lives in config only and the provider can reconcile import-then-apply
// without sending empty credentials to the API.
//
// Restricted to fields the Galaxy API treats as write-only (returns
// "<Value is encrypted>" or omits from GET responses). Identifier-style
// credentials like access_key are not included because the API does return
// them - they should be Sensitive but not WriteOnly.
func IsCredentialField(name string) bool {
	lower := strings.ToLower(name)

	exceptions := map[string]bool{
		"synctoken":  true,
		"sync_token": true,
		"authtype":   true,
		"auth_type":  true,
	}
	if exceptions[lower] {
		return false
	}

	patterns := []string{
		"password",
		"passphrase",
		"credential",
		"private_key", "privatekey",
	}
	for _, p := range patterns {
		if strings.Contains(lower, p) {
			return true
		}
	}

	if strings.Contains(lower, "token") && !strings.Contains(lower, "sync") {
		return true
	}

	return false
}
