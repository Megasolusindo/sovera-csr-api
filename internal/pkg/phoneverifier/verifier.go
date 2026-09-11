package phoneverifier

import (
	"fmt"
	"regexp"
	"strings"
)

// PhoneVerifier validates and normalizes phone numbers (Indonesian mobile/landline & international E.164).
type PhoneVerifier struct{}

// DefaultVerifier singleton instance
var DefaultVerifier = &PhoneVerifier{}

var nonDigitOrPlusRegex = regexp.MustCompile(`[^\d+]`)

// Normalize cleans up spaces, dashes, parentheses, dots and standardizes Indonesian phone formats.
func (v *PhoneVerifier) Normalize(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	// Remove spaces, hyphens, dots, parentheses
	cleaned := strings.ReplaceAll(raw, " ", "")
	cleaned = strings.ReplaceAll(cleaned, "-", "")
	cleaned = strings.ReplaceAll(cleaned, ".", "")
	cleaned = strings.ReplaceAll(cleaned, "(", "")
	cleaned = strings.ReplaceAll(cleaned, ")", "")
	cleaned = strings.ReplaceAll(cleaned, "/", "")

	// Standardize Indonesian prefix 62 -> 08 for mobile or 02x/03x for landline if started with 62
	if strings.HasPrefix(cleaned, "+62") {
		cleaned = "0" + cleaned[3:]
	} else if strings.HasPrefix(cleaned, "62") {
		cleaned = "0" + cleaned[2:]
	}

	return cleaned
}

// Verify checks whether raw input is a valid phone number.
// Returns (isValid, normalizedPhone, error).
func (v *PhoneVerifier) Verify(raw string) (bool, string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return false, "", fmt.Errorf("phone number is empty")
	}

	// Reject if raw contains alphabetic letters or non-phone symbols
	for _, ch := range raw {
		if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') {
			return false, raw, fmt.Errorf("phone number contains invalid letters: '%s'", raw)
		}
	}

	normalized := v.Normalize(raw)

	// Extract digits only for length check
	digitsOnly := strings.ReplaceAll(normalized, "+", "")
	if len(digitsOnly) < 7 || len(digitsOnly) > 15 {
		return false, normalized, fmt.Errorf("phone number length must be between 7 and 15 digits (got %d digits)", len(digitsOnly))
	}

	// Reject dummy sequence numbers (e.g. 000000000, 111111111, 123456789)
	if isDummyNumber(digitsOnly) {
		return false, normalized, fmt.Errorf("phone number is a dummy or placeholder pattern: '%s'", raw)
	}

	// Validate Indonesian or E.164 Prefix
	// Indonesian Mobile: 08xx (10 to 13 digits)
	// Indonesian Landline: 021, 022, 031, 0274, 061, 0251, 024, etc. (7 to 12 digits)
	// International E.164: +<country_code><digits>
	if strings.HasPrefix(normalized, "08") {
		if len(digitsOnly) < 10 || len(digitsOnly) > 14 {
			return false, normalized, fmt.Errorf("indonesian mobile number must be 10 to 14 digits (got %d)", len(digitsOnly))
		}
		return true, normalized, nil
	}

	if strings.HasPrefix(normalized, "0") {
		// General Indonesian landline (02x, 03x, 04x, 05x, 06x, 07x, 09x)
		if len(digitsOnly) < 7 || len(digitsOnly) > 13 {
			return false, normalized, fmt.Errorf("indonesian landline number must be 7 to 13 digits (got %d)", len(digitsOnly))
		}
		return true, normalized, nil
	}

	if strings.HasPrefix(normalized, "+") {
		return true, normalized, nil
	}

	// Valid digits fallback (e.g. 628xxx without plus)
	return true, normalized, nil
}

// isDummyNumber detects repeated or sequential dummy numbers like "0000000000", "123456789"
func isDummyNumber(digits string) bool {
	if len(digits) == 0 {
		return true
	}

	// All same digits
	allSame := true
	firstChar := digits[0]
	for i := 1; i < len(digits); i++ {
		if digits[i] != firstChar {
			allSame = false;
			break
		}
	}
	if allSame {
		return true
	}

	// Sequential 123456789 / 0123456789
	if digits == "123456789" || digits == "0123456789" || digits == "987654321" {
		return true
	}

	return false
}
