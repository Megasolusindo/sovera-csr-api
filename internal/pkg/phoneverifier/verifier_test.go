package phoneverifier

import (
	"testing"
)

func TestPhoneVerifier(t *testing.T) {
	v := DefaultVerifier

	tests := []struct {
		input    string
		wantValid bool
		wantNorm  string
	}{
		{"08123456789", true, "08123456789"},
		{"+628123456789", true, "08123456789"},
		{"6281234567890", true, "081234567890"},
		{"(021) 789-1234", true, "0217891234"},
		{"0857-1234-5678", true, "085712345678"},
		{"+1 415 555 2671", true, "+14155552671"},

		// Invalid cases
		{"", false, ""},
		{"not_a_phone", false, "not_a_phone"},
		{"0812", false, "0812"},
		{"123456", false, "123456"},
		{"0000000000", false, "0000000000"},
		{"11111111111", false, "11111111111"},
		{"123456789", false, "123456789"},
	}

	for _, tt := range tests {
		gotValid, gotNorm, err := v.Verify(tt.input)
		if gotValid != tt.wantValid {
			t.Errorf("Verify(%q) valid = %v (err: %v), wantValid %v", tt.input, gotValid, err, tt.wantValid)
		}
		if tt.wantValid && gotNorm != tt.wantNorm {
			t.Errorf("Verify(%q) norm = %q, wantNorm %q", tt.input, gotNorm, tt.wantNorm)
		}
	}
}
