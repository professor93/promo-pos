//go:build windows
// +build windows

package security

import (
	"fmt"

	"golang.org/x/sys/windows/registry"
)

// getWindowsProductID retrieves the Windows Product ID from registry
func getWindowsProductID() (string, error) {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE,
		`SOFTWARE\Microsoft\Windows NT\CurrentVersion`,
		registry.QUERY_VALUE)
	if err != nil {
		return "", err
	}
	defer k.Close()

	productID, _, err := k.GetStringValue("ProductId")
	if err != nil {
		return "", err
	}

	return productID, nil
}

// getCPUInfo retrieves CPU information from Windows registry
func getCPUInfo() (string, error) {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE,
		`HARDWARE\DESCRIPTION\System\CentralProcessor\0`,
		registry.QUERY_VALUE)
	if err != nil {
		return "", err
	}
	defer k.Close()

	// Try to get ProcessorNameString
	cpuName, _, err := k.GetStringValue("ProcessorNameString")
	if err != nil {
		return "", err
	}

	// Try to get Identifier for additional uniqueness
	identifier, _, err := k.GetStringValue("Identifier")
	if err == nil {
		cpuName += "|" + identifier
	}

	return cpuName, nil
}

// readMachineIDFromRegistry reads the stored machine ID from Windows registry
func readMachineIDFromRegistry() (string, error) {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, registryPath, registry.QUERY_VALUE)
	if err != nil {
		return "", err
	}
	defer k.Close()

	machineID, _, err := k.GetStringValue(machineIDKey)
	if err != nil {
		return "", err
	}

	return machineID, nil
}

// saveMachineIDToRegistry stores the machine ID in Windows registry
func saveMachineIDToRegistry(machineID string) error {
	k, exists, err := registry.CreateKey(registry.LOCAL_MACHINE, registryPath, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()

	if !exists {
		// Key was created
	}

	return k.SetStringValue(machineIDKey, machineID)
}

// platformSpecificID generates platform-specific machine ID data
func platformSpecificID() ([]string, error) {
	var data []string

	// Windows Product ID
	productID, err := getWindowsProductID()
	if err == nil {
		data = append(data, productID)
	}

	// CPU Information
	cpuInfo, err := getCPUInfo()
	if err == nil {
		data = append(data, cpuInfo)
	}

	if len(data) == 0 {
		return nil, fmt.Errorf("could not collect Windows-specific data")
	}

	return data, nil
}
