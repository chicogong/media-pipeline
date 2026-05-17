package prober

import "testing"

// TestParseInt64Valid tests parseInt64 with a valid numeric string.
func TestParseInt64Valid(t *testing.T) {
	got := parseInt64("1234567890")
	if got != 1234567890 {
		t.Errorf("parseInt64(%q) = %d, want 1234567890", "1234567890", got)
	}
}

// TestParseInt64Empty tests that parseInt64 returns 0 for an empty string.
func TestParseInt64Empty(t *testing.T) {
	got := parseInt64("")
	if got != 0 {
		t.Errorf("parseInt64(%q) = %d, want 0", "", got)
	}
}

// TestParseInt64Invalid tests that parseInt64 returns 0 for a non-numeric string.
func TestParseInt64Invalid(t *testing.T) {
	got := parseInt64("not-a-number")
	if got != 0 {
		t.Errorf("parseInt64(%q) = %d, want 0", "not-a-number", got)
	}
}

// TestParseIntValid tests parseInt with a valid numeric string.
func TestParseIntValid(t *testing.T) {
	got := parseInt("48000")
	if got != 48000 {
		t.Errorf("parseInt(%q) = %d, want 48000", "48000", got)
	}
}

// TestParseIntEmpty tests that parseInt returns 0 for an empty string.
func TestParseIntEmpty(t *testing.T) {
	got := parseInt("")
	if got != 0 {
		t.Errorf("parseInt(%q) = %d, want 0", "", got)
	}
}

// TestParseIntInvalid tests that parseInt returns 0 for a non-numeric string.
func TestParseIntInvalid(t *testing.T) {
	got := parseInt("abc")
	if got != 0 {
		t.Errorf("parseInt(%q) = %d, want 0", "abc", got)
	}
}

// TestParseFrameRateNumDen tests parseFrameRate with the canonical "num/den" form.
func TestParseFrameRateNumDen(t *testing.T) {
	got := parseFrameRate("30000/1001")
	want := 30000.0 / 1001.0
	if got != want {
		t.Errorf("parseFrameRate(%q) = %f, want %f", "30000/1001", got, want)
	}
}

// TestParseFrameRatePlainFloat tests parseFrameRate when the input is a plain float string.
func TestParseFrameRatePlainFloat(t *testing.T) {
	got := parseFrameRate("29.97")
	if got != 29.97 {
		t.Errorf("parseFrameRate(%q) = %f, want 29.97", "29.97", got)
	}
}

// TestParseFrameRateEmpty tests that parseFrameRate returns 0 for an empty string.
func TestParseFrameRateEmpty(t *testing.T) {
	got := parseFrameRate("")
	if got != 0 {
		t.Errorf("parseFrameRate(%q) = %f, want 0", "", got)
	}
}

// TestParseFrameRateZeroDenominator tests that parseFrameRate returns 0 when the denominator is zero.
func TestParseFrameRateZeroDenominator(t *testing.T) {
	got := parseFrameRate("30/0")
	if got != 0 {
		t.Errorf("parseFrameRate(%q) = %f, want 0", "30/0", got)
	}
}

// TestParseFrameRateMalformedFraction tests parseFrameRate with non-numeric parts in the fraction.
func TestParseFrameRateMalformedFraction(t *testing.T) {
	got := parseFrameRate("abc/def")
	if got != 0 {
		t.Errorf("parseFrameRate(%q) = %f, want 0", "abc/def", got)
	}
}

// TestParseFrameRateMalformedPlain tests parseFrameRate with a malformed plain (non-fraction) string.
func TestParseFrameRateMalformedPlain(t *testing.T) {
	got := parseFrameRate("not-a-rate")
	if got != 0 {
		t.Errorf("parseFrameRate(%q) = %f, want 0", "not-a-rate", got)
	}
}
