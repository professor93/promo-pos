//go:build linux
// +build linux

package security

import (
	"fmt"
	"os"
	"strings"
)

// getWindowsProductID is not available on Linux
func getWindowsProductID() (string, error) {
	return "", fmt.Errorf("not on Windows")
}

// getCPUInfo retrieves CPU information on Linux
func getCPUInfo() (string, error) {
	// Read /proc/cpuinfo
	data, err := os.ReadFile("/proc/cpuinfo")
	if err != nil {
		return "", err
	}

	lines := strings.Split(string(data), "\n")
	var cpuModel, cpuSerial string

	for _, line := range lines {
		if strings.HasPrefix(line, "model name") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				cpuModel = strings.TrimSpace(parts[1])
			}
		}
		if strings.HasPrefix(line, "Serial") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				cpuSerial = strings.TrimSpace(parts[1])
			}
		}
	}

	if cpuModel != "" {
		if cpuSerial != "" {
			return cpuModel + "|" + cpuSerial, nil
		}
		return cpuModel, nil
	}

	return "", fmt.Errorf("could not determine CPU info")
}

// readMachineIDFromRegistry reads from a file on Linux
func readMachineIDFromRegistry() (string, error) {
	// On Linux, store in /var/lib/posservice/machine_id
	machineIDPath := "/var/lib/posservice/machine_id"

	data, err := os.ReadFile(machineIDPath)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(data)), nil
}

// saveMachineIDToRegistry stores to a file on Linux
func saveMachineIDToRegistry(machineID string) error {
	// On Linux, store in /var/lib/posservice/machine_id
	dir := "/var/lib/posservice"
	machineIDPath := dir + "/machine_id"

	// Create directory if it doesn't exist
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write machine ID to file
	if err := os.WriteFile(machineIDPath, []byte(machineID), 0644); err != nil {
		return fmt.Errorf("failed to write machine ID file: %w", err)
	}

	return nil
}

// platformSpecificID generates platform-specific machine ID data for Linux
func platformSpecificID() ([]string, error) {
	var data []string

	// Try to read /etc/machine-id (systemd)
	machineID, err := os.ReadFile("/etc/machine-id")
	if err == nil && len(machineID) > 0 {
		data = append(data, strings.TrimSpace(string(machineID)))
	}

	// Try to read /var/lib/dbus/machine-id (alternative)
	if len(data) == 0 {
		dbusMachineID, err := os.ReadFile("/var/lib/dbus/machine-id")
		if err == nil && len(dbusMachineID) > 0 {
			data = append(data, strings.TrimSpace(string(dbusMachineID)))
		}
	}

	// CPU Information
	cpuInfo, err := getCPUInfo()
	if err == nil {
		data = append(data, cpuInfo)
	}

	if len(data) == 0 {
		return nil, fmt.Errorf("could not collect Linux-specific data")
	}

	return data, nil
}
