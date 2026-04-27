// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package convert

import "testing"

func TestIsCredentialField(t *testing.T) {
	t.Parallel()

	cases := map[string]bool{
		// write-only credentials the API never returns
		"password":               true,
		"credentials_key":        true,
		"credentialsKey":         true,
		"private_key":            true,
		"privateKey":             true,
		"private_key_passphrase": true,
		"passphrase":             true,
		"token":                  true,

		// excluded from heuristic - false positives or API does return them
		"sync_token":      false,
		"syncToken":       false,
		"auth_type":       false,
		"authType":        false,
		"access_key":      false, // API returns it; identifier paired with secret_key
		"accessKey":       false,
		"glue_access_key": false,
		"secret_key":      false, // currently returned; reassess separately
		"glue_secret_key": false,
		"api_key":         false, // not used in this provider; can be added if needed
		"client_secret":   false,

		// non-credential names
		"name":        false,
		"description": false,
		"catalog_id":  false,
		"username":    false,
		"role_name":   false,
		"endpoint":    false,
		"hosts":       false,
		"port":        false,
		"validate":    false,
	}

	for name, want := range cases {
		got := IsCredentialField(name)
		if got != want {
			t.Errorf("IsCredentialField(%q) = %v, want %v", name, got, want)
		}
	}
}
