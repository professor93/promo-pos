package security

import (
	"testing"
)

func TestGetMachineID(t *testing.T) {
	// Get machine ID
	id1, err := GetMachineID()
	if err != nil {
		t.Fatalf("Failed to get machine ID: %v", err)
	}

	if id1 == "" {
		t.Fatal("Machine ID is empty")
	}

	// Verify it's a hex string (SHA256 = 64 hex characters)
	if len(id1) != 64 {
		t.Errorf("Expected machine ID length 64, got %d", len(id1))
	}

	// Verify consistency - calling again should return the same ID
	id2, err := GetMachineID()
	if err != nil {
		t.Fatalf("Failed to get machine ID second time: %v", err)
	}

	if id1 != id2 {
		t.Errorf("Machine ID not consistent.\nFirst:  %s\nSecond: %s", id1, id2)
	}
}

func TestGetMachineID_Caching(t *testing.T) {
	// Clear cache for test
	cachedMachineID = ""

	// First call
	id1, err := GetMachineID()
	if err != nil {
		t.Fatalf("Failed to get machine ID: %v", err)
	}

	// Second call should use cache
	id2, err := GetMachineID()
	if err != nil {
		t.Fatalf("Failed to get cached machine ID: %v", err)
	}

	if id1 != id2 {
		t.Error("Cached machine ID doesn't match original")
	}
}

func TestGetPrimaryMACAddress(t *testing.T) {
	mac, err := getPrimaryMACAddress()
	if err != nil {
		// It's okay if there's no network interface in test environment
		t.Skipf("Skipping MAC address test: %v", err)
	}

	if mac == "" {
		t.Error("MAC address is empty")
	}

	// Verify MAC address format (should contain colons for typical format)
	// e.g., "00:11:22:33:44:55"
	t.Logf("Primary MAC address: %s", mac)
}

func TestGenerateMachineID(t *testing.T) {
	// Generate machine ID
	id, err := generateMachineID()
	if err != nil {
		t.Fatalf("Failed to generate machine ID: %v", err)
	}

	if id == "" {
		t.Fatal("Generated machine ID is empty")
	}

	// Verify it's a hex string
	if len(id) != 64 {
		t.Errorf("Expected machine ID length 64, got %d", len(id))
	}

	// Verify it's hexadecimal
	for _, c := range id {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			t.Errorf("Machine ID contains non-hex character: %c", c)
			break
		}
	}
}

func BenchmarkGetMachineID(b *testing.B) {
	for i := 0; i < b.N; i++ {
		GetMachineID()
	}
}
