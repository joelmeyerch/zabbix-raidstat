package main

import "github.com/ps78674/zabbix-raidstat/plugins/internal/storcli"

// GetControllersIDs - get number of controllers in the system
func GetControllersIDs(execPath string) []string {
	return storcli.GetControllersIDs(execPath)
}

// GetLogicalDrivesIDs - get number of logical drives for controller with ID 'controllerID'
func GetLogicalDrivesIDs(execPath string, controllerID string) []string {
	return storcli.GetLogicalDrivesIDs(execPath, controllerID)
}

// GetPhysicalDrivesIDs - get number of physical drives for controller with ID 'controllerID'
func GetPhysicalDrivesIDs(execPath string, controllerID string) []string {
	return storcli.GetPhysicalDrivesIDs(execPath, controllerID)
}

// GetControllerStatus - get controller status
func GetControllerStatus(execPath string, controllerID string, indent int) []byte {
	return storcli.GetControllerStatus(execPath, controllerID, indent)
}

// GetLDStatus - get logical drive status
func GetLDStatus(execPath string, controllerID string, deviceID string, indent int) []byte {
	return storcli.GetLDStatus(execPath, controllerID, deviceID, indent)
}

// GetPDStatus - get physical drive status
func GetPDStatus(execPath string, controllerID string, deviceID string, indent int) []byte {
	return storcli.GetPDStatus(execPath, controllerID, deviceID, indent)
}

func main() {}
