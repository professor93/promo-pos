package security

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"sync"
)

const (
	registryPath = `SOFTWARE\POSService` // Windows only
	machineIDKey = "MachineID"
)

var (
	cachedMachineID string
	machineIDMutex  sync.RWMutex
)

// GetMachineID returns the unique machine identifier
// It generates a machine ID from platform-specific sources and MAC address
// The ID is cached in memory and stored persistently
func GetMachineID() (string, error) {
	// Check cached value first
	machineIDMutex.RLock()
	if cachedMachineID != "" {
		machineIDMutex.RUnlock()
		return cachedMachineID, nil
	}
	machineIDMutex.RUnlock()

	// Try to read from persistent storage
	machineIDMutex.Lock()
	defer machineIDMutex.Unlock()

	// Double-check after acquiring write lock
	if cachedMachineID != "" {
		return cachedMachineID, nil
	}

	machineID, err := readMachineIDFromRegistry()
	if err == nil && machineID != "" {
		cachedMachineID = machineID
		return machineID, nil
	}

	// Generate new machine ID
	machineID, err = generateMachineID()
	if err != nil {
		return "", fmt.Errorf("failed to generate machine ID: %w", err)
	}

	// Store persistently
	if err := saveMachineIDToRegistry(machineID); err != nil {
		// Log warning but don't fail - we can still use the generated ID
		fmt.Printf("Warning: failed to save machine ID: %v\n", err)
	}

	cachedMachineID = machineID
	return machineID, nil
}

// generateMachineID creates a unique machine identifier
func generateMachineID() (string, error) {
	var data []string

	// Get platform-specific data (implemented in machine_id_windows.go and machine_id_linux.go)
	platformData, err := platformSpecificID()
	if err == nil {
		data = append(data, platformData...)
	}

	// MAC Address of primary network interface (cross-platform)
	macAddr, err := getPrimaryMACAddress()
	if err == nil {
		data = append(data, macAddr)
	}

	// Hostname (cross-platform)
	hostname, err := os.Hostname()
	if err == nil {
		data = append(data, hostname)
	}

	if len(data) == 0 {
		return "", fmt.Errorf("could not collect any machine-specific data")
	}

	// Combine all data and hash with SHA256
	combined := ""
	for _, d := range data {
		combined += d + "|"
	}

	hash := sha256.Sum256([]byte(combined))
	return hex.EncodeToString(hash[:]), nil
}

// getPrimaryMACAddress gets the MAC address of the first non-loopback interface
func getPrimaryMACAddress() (string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}

	for _, iface := range interfaces {
		// Skip loopback and down interfaces
		if iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagUp == 0 {
			continue
		}

		// Get MAC address
		mac := iface.HardwareAddr.String()
		if mac != "" {
			return mac, nil
		}
	}

	return "", fmt.Errorf("no valid network interface found")
}
