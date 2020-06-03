// Package drivemgr contains a code for managers of storage hardware such as drives
package drivemgr

import api "eos2git.cec.lab.emc.com/ECS/baremetal-csi-plugin.git/api/generated/v1"

/*
List of supported implementations
*/
const (
	DEFAULT string = "HAL"
	REDFISH string = "IDRAC"
	TEST    string = "LOOPBACK"
)

// DriveManager is the interface for managers that provide information about drives on a node
type DriveManager interface {
	// get list of drives
	GetDrivesList() ([]*api.Drive, error)
}