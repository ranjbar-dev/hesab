// ponytail: parallel to adminauth, kept separate on purpose.
package user

import "testing"

func TestNormalizePhone(t *testing.T) {
	for _, v := range []string{"+989370843199", "00989370843199", "09370843199", "9370843199", "937 084 3199", "0937-084-3199"} {
		if got, e := NormalizePhone(v); e != nil || got != "9370843199" {
			t.Fatalf("%q: %q %v", v, got, e)
		}
	}
	for _, v := range []string{"12345", "8370843199", ""} {
		if _, e := NormalizePhone(v); e == nil {
			t.Fatalf("%q accepted", v)
		}
	}
}
