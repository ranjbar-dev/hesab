package user

import "testing"

func TestValidateNationalID(t *testing.T) {
	for _, tc := range []struct {
		name, value string
		valid       bool
	}{
		{"known good", "0079059740", true}, {"all same", "1111111111", false},
		{"wrong length", "123", false}, {"bad checksum", "0079059749", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if (ValidateNationalID(tc.value) == nil) != tc.valid {
				t.Fatalf("%q validity mismatch", tc.value)
			}
		})
	}
}
