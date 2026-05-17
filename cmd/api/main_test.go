package main

import "testing"

// TestGetEnvReturnsValue verifies that getEnv returns the environment variable's value when set.
func TestGetEnvReturnsValue(t *testing.T) {
	t.Setenv("TEST_GET_ENV_KEY", "hello")
	got := getEnv("TEST_GET_ENV_KEY", "fallback")
	if got != "hello" {
		t.Errorf("getEnv() = %q, want %q", got, "hello")
	}
}

// TestGetEnvReturnsFallback verifies that getEnv returns the fallback when the variable is not set.
func TestGetEnvReturnsFallback(t *testing.T) {
	// Ensure the variable is absent (t.Setenv restores after test, but we never set it here).
	got := getEnv("TEST_GET_ENV_UNSET_XYZ123", "default-value")
	if got != "default-value" {
		t.Errorf("getEnv() = %q, want %q", got, "default-value")
	}
}
