package main

import (
	"testing"
)

func TestGetEnv(t *testing.T) {
	// Test missing environment variable
	_, err := getEnv("NON_EXISTENT_VAR")
	if err == nil {
		t.Error("Expected error for missing environment variable, got nil")
	}

	// Test existing environment variable (set one for testing)
	t.Setenv("TEST_VAR", "test_value")
	val, err := getEnv("TEST_VAR")
	if err != nil {
		t.Errorf("Expected no error for existing environment variable, got: %v", err)
	}
	if val != "test_value" {
		t.Errorf("Expected 'test_value', got: %s", val)
	}
}

func TestMaxSatelliteCount(t *testing.T) {
	// Test that the constant is set to the expected value
	if MaxSatelliteCount != 12 {
		t.Errorf("Expected MaxSatelliteCount to be 12, got: %d", MaxSatelliteCount)
	}
}

func TestGnssFullDataStructSize(t *testing.T) {
	// Test that array sizes are using the constant
	var data GnssFullData
	if len(data.Slmsg) != MaxSatelliteCount {
		t.Errorf("Expected Slmsg array size to be %d, got: %d", MaxSatelliteCount, len(data.Slmsg))
	}
	if len(data.BeidouSlmsg) != MaxSatelliteCount {
		t.Errorf("Expected BeidouSlmsg array size to be %d, got: %d", MaxSatelliteCount, len(data.BeidouSlmsg))
	}
	if len(data.Possl) != MaxSatelliteCount {
		t.Errorf("Expected Possl array size to be %d, got: %d", MaxSatelliteCount, len(data.Possl))
	}
}