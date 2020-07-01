package loopbackmgr

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	"eos2git.cec.lab.emc.com/ECS/baremetal-csi-plugin.git/pkg/mocks"
)

var logger = logrus.New()

func TestLoopBackManager_getLoopBackDeviceName(t *testing.T) {
	var mockexec = &mocks.GoMockExecutor{}
	var manager = NewLoopBackManager(mockexec, logger)

	file := "/tmp/test"
	loop := "/dev/loop18"
	mockexec.On("RunCmd", fmt.Sprintf(checkLoopBackDeviceCmdTmpl, file)).
		Return(loop+": []: ("+file+")", "", nil)
	device, err := manager.GetLoopBackDeviceName(file)

	assert.Equal(t, "/dev/loop18", device)
	assert.Nil(t, err)
}

func TestLoopBackManager_getLoopBackDeviceName_NotFound(t *testing.T) {
	var mockexec = &mocks.GoMockExecutor{}
	var manager = NewLoopBackManager(mockexec, logger)

	file := "/tmp/test"
	mockexec.On("RunCmd", fmt.Sprintf(checkLoopBackDeviceCmdTmpl, file)).
		Return("", "", nil)
	device, err := manager.GetLoopBackDeviceName(file)
	assert.Equal(t, "", device)
	assert.Nil(t, err)
}

func TestLoopBackManager_getLoopBackDeviceName_Fail(t *testing.T) {
	var mockexec = &mocks.GoMockExecutor{}
	var manager = NewLoopBackManager(mockexec, logger)

	file := "/tmp/test"
	error := errors.New("losetup: command not found")
	mockexec.On("RunCmd", fmt.Sprintf(checkLoopBackDeviceCmdTmpl, file)).
		Return("", "", error)
	device, err := manager.GetLoopBackDeviceName(file)
	assert.Equal(t, "", device)
	assert.Equal(t, error, err)
}

func TestLoopBackManager_CleanupLoopDevices(t *testing.T) {
	var mockexec = &mocks.GoMockExecutor{}
	var manager = NewLoopBackManager(mockexec, logger)

	for _, device := range manager.devices {
		mockexec.On("RunCmd", fmt.Sprintf(detachLoopBackDeviceCmdTmpl, device.devicePath)).
			Return("", "", nil)
		mockexec.On("RunCmd", fmt.Sprintf(deleteFileCmdTmpl, device.fileName)).
			Return("", "", nil)
	}

	manager.CleanupLoopDevices()
}

func TestLoopBackManager_UpdateDevicesFromLocalConfig(t *testing.T) {
	var mockexec = &mocks.GoMockExecutor{}
	var manager = NewLoopBackManager(mockexec, logger)

	manager.updateDevicesFromConfig()

	assert.Equal(t, defaultNumberOfDevices, len(manager.devices))
}

func TestLoopBackManager_UpdateDevicesFromSetConfig(t *testing.T) {
	var mockexec = &mocks.GoMockExecutor{}
	var manager = NewLoopBackManager(mockexec, logger)
	testSN := "testSN"
	testNodeID := "testNode"
	testConfigPath := "/tmp/config.yaml"

	config := []byte("defaultDrivePerNodeCount: 3")
	err := ioutil.WriteFile(testConfigPath, config, 0777)
	assert.Nil(t, err)
	defer func() {
		_ = os.Remove(testConfigPath)
	}()

	manager.readAndSetConfig(testConfigPath)
	manager.updateDevicesFromConfig()

	assert.Equal(t, 3, len(manager.devices))

	manager.nodeID = testNodeID
	config = []byte("nodes:\n" +
		fmt.Sprintf("- nodeID: %s\n", testNodeID) +
		fmt.Sprintf("  driveCount: %d\n", 5) +
		"  drives:\n" +
		fmt.Sprintf("  - serialNumber: %s\n", testSN))
	err = ioutil.WriteFile(testConfigPath, config, 0777)
	assert.Nil(t, err)

	manager.readAndSetConfig(testConfigPath)
	manager.updateDevicesFromConfig()

	assert.Equal(t, 5, len(manager.devices))

	found := false
	for _, device := range manager.devices {
		if device.SerialNumber == testSN {
			found = true
		}
	}

	assert.Equal(t, true, found)
}

func TestLoopBackManager_updateDevicesFromSetConfigWithSize(t *testing.T) {
	var mockexec = &mocks.GoMockExecutor{}

	var manager = NewLoopBackManager(mockexec, logger)

	config := []byte("defaultDriveSize: 30Mi \ndefaultDrivePerNodeCount: 3")
	testConfigPath := "/tmp/config.yaml"
	err := ioutil.WriteFile(testConfigPath, config, 0777)
	assert.Nil(t, err)

	defer func() {
		_ = os.Remove(testConfigPath)
	}()
	for _, device := range manager.devices {
		device.devicePath = "/dev/sda"
		mockexec.On("RunCmd", fmt.Sprintf(detachLoopBackDeviceCmdTmpl, device.devicePath)).
			Return("", "", nil)
		mockexec.On("RunCmd", fmt.Sprintf(deleteFileCmdTmpl, device.fileName)).
			Return("", "", nil)
	}
	manager.readAndSetConfig(testConfigPath)
	manager.updateDevicesFromConfig()

	for _, device := range manager.devices {
		assert.Equal(t, device.Size, "30Mi")
	}

}

func TestLoopBackManager_overrideDevicesFromSetConfigWithSize(t *testing.T) {
	var mockexec = &mocks.GoMockExecutor{}
	var manager = NewLoopBackManager(mockexec, logger)
	testSN := "testSN"
	testNodeID := "testNode"
	testConfigPath := "/tmp/config.yaml"
	config := []byte("defaultDriveSize: 30Mi \ndefaultDrivePerNodeCount: 3\nnodes:\n" +
		fmt.Sprintf("- nodeID: %s\n", testNodeID) +
		fmt.Sprintf("  driveCount: %d\n", 5) +
		"  drives:\n" +
		fmt.Sprintf("  - serialNumber: %s\n", testSN) +
		fmt.Sprintf("    size: %s\n", "40Mi"))
	err := ioutil.WriteFile(testConfigPath, config, 0777)
	assert.Nil(t, err)

	defer func() {
		_ = os.Remove(testConfigPath)
	}()
	manager.nodeID = testNodeID
	for _, device := range manager.devices {
		mockexec.On("RunCmd", fmt.Sprintf(detachLoopBackDeviceCmdTmpl, device.devicePath)).
			Return("", "", nil)
		mockexec.On("RunCmd", fmt.Sprintf(deleteFileCmdTmpl, device.fileName)).
			Return("", "", nil)
	}

	manager.readAndSetConfig(testConfigPath)
	manager.updateDevicesFromConfig()

	config = []byte("defaultDriveSize: 30Mi \ndefaultDrivePerNodeCount: 3\nnodes:\n" +
		fmt.Sprintf("- nodeID: %s\n", testNodeID) +
		fmt.Sprintf("  driveCount: %d\n", 5) +
		"  drives:\n" +
		fmt.Sprintf("  - serialNumber: %s\n", testSN))
	err = ioutil.WriteFile(testConfigPath, config, 0777)
	assert.Nil(t, err)

	for _, device := range manager.devices {
		if device.SerialNumber == testSN {
			device.devicePath = "/dev/sda"
			mockexec.On("RunCmd", fmt.Sprintf(detachLoopBackDeviceCmdTmpl, device.devicePath)).
				Return("", "", nil)
			mockexec.On("RunCmd", fmt.Sprintf(deleteFileCmdTmpl, device.fileName)).
				Return("", "", nil)
		}
	}
	manager.readAndSetConfig(testConfigPath)
	manager.updateDevicesFromConfig()

	assert.Nil(t, err)

	for _, device := range manager.devices {
		if device.SerialNumber == testSN {
			assert.Equal(t, device.Size, "30Mi")
		}
	}

}

func TestLoopBackManager_overrideDevicesFromNodeConfig(t *testing.T) {
	var mockexec = &mocks.GoMockExecutor{}
	var manager = NewLoopBackManager(mockexec, logger)

	// Initialize manager with local default settings
	manager.updateDevicesFromConfig()

	assert.Equal(t, defaultNumberOfDevices, len(manager.devices))

	indexOfDeviceToOverride := 0
	newVID := "newVID"
	// The first device should be overrode
	// The second device should be added
	devices := []*LoopBackDevice{
		{SerialNumber: manager.devices[indexOfDeviceToOverride].SerialNumber, VendorID: newVID},
		{SerialNumber: "newDevice"},
	}

	manager.overrideDevicesFromNodeConfig(defaultNumberOfDevices+1, devices)

	assert.Equal(t, manager.devices[indexOfDeviceToOverride].VendorID, newVID)
	assert.Equal(t, defaultNumberOfDevices+1, len(manager.devices))
}

func TestLoopBackManager_overrideDeviceWithSizeChanging(t *testing.T) {
	var mockexec = &mocks.GoMockExecutor{}
	var manager = NewLoopBackManager(mockexec, logger)
	// Initialize manager with local default settings
	manager.updateDevicesFromConfig()

	assert.Equal(t, defaultNumberOfDevices, len(manager.devices))

	indexOfDeviceToOverride := 0
	newSize := "200Mi"
	fakeDevicePath := "/dev/loop0"
	fakeFileName := "loopback.img"
	manager.devices[indexOfDeviceToOverride].devicePath = fakeDevicePath
	manager.devices[indexOfDeviceToOverride].fileName = fakeFileName

	mockexec.On("RunCmd", fmt.Sprintf(detachLoopBackDeviceCmdTmpl, fakeDevicePath)).
		Return("", "", nil)
	mockexec.On("RunCmd", fmt.Sprintf(deleteFileCmdTmpl, fakeFileName)).
		Return("", "", nil)

	// Change size of device to override
	devices := []*LoopBackDevice{
		{SerialNumber: manager.devices[indexOfDeviceToOverride].SerialNumber, Size: newSize},
	}

	manager.overrideDevicesFromNodeConfig(defaultNumberOfDevices, devices)
	assert.Equal(t, manager.devices[indexOfDeviceToOverride].Size, newSize)
}

//func TestLoopBackManager_GetDrivesList(t *testing.T) {
//	var mockexec = &mocks.GoMockExecutor{}
//	var manager = NewLoopBackManager(mockexec, logger)
//	fakeDevicePath := "/dev/loop"
//
//	manager.updateDevicesFromConfig()
//	for i, device := range manager.devices {
//		device.devicePath = fmt.Sprintf(fakeDevicePath+"%d", i)
//	}
//	indexOfDriveToOffline := 0
//	manager.devices[indexOfDriveToOffline].Removed = true
//	drives, err := manager.GetDrivesList()
//
//	assert.Nil(t, err)
//	assert.Equal(t, defaultNumberOfDevices, len(drives))
//	assert.Equal(t, apiV1.DriveStatusOffline, drives[indexOfDriveToOffline].Status)
//}

func TestLoopBackManager_attemptToRecoverDevicesFromConfig(t *testing.T) {
	testImagesPath := "/tmp/images"
	err := os.Mkdir(testImagesPath, 0777)
	assert.Nil(t, err)
	defer func() {
		// cleanup fake images
		_ = os.RemoveAll(testImagesPath)
	}()

	var mockexec = &mocks.GoMockExecutor{}
	var manager = NewLoopBackManager(mockexec, logger)
	// Clean devices after default initialization in constructor
	manager.devices = make([]*LoopBackDevice, 0)

	// image file that should be ignored during recovery
	ignoredImage := fmt.Sprintf("%s/%s", testImagesPath, "random.img")
	_, err = os.Create(ignoredImage)
	mockexec.On("RunCmd", fmt.Sprintf(deleteFileCmdTmpl, ignoredImage)).Return("", "", nil)
	assert.Nil(t, err)

	// image of device that should be recovered from default config
	testSerialNumber1 := "12345"
	_, err = os.Create(fmt.Sprintf("%s/%s-%s.img", testImagesPath, manager.hostname, testSerialNumber1))
	assert.Nil(t, err)

	// image of device that should be recovered from node config
	testSerialNumber2 := "56789"
	nonDefaultVID := "non-default-VID"
	_, err = os.Create(fmt.Sprintf("%s/%s-%s.img", testImagesPath, manager.hostname, testSerialNumber2))
	assert.Nil(t, err)

	// set manager's node config
	manager.config = &Config{
		DefaultDriveCount: 3,
		Nodes: []*Node{
			{
				Drives: []*LoopBackDevice{
					{
						SerialNumber: fmt.Sprintf("LOOPBACK%s", testSerialNumber2),
						VendorID:     nonDefaultVID,
					},
				},
			},
		},
	}

	manager.attemptToRecoverDevices(testImagesPath)
	assert.Equal(t, len(manager.devices), 2)

	var recoveredDeviceVID string
	for _, device := range manager.devices {
		if strings.Contains(device.SerialNumber, testSerialNumber2) {
			recoveredDeviceVID = device.VendorID
			break
		}
	}
	assert.Equal(t, recoveredDeviceVID, nonDefaultVID)
}

func TestLoopBackManager_attemptToRecoverDevicesFromDefaults(t *testing.T) {
	testImagesPath := "/tmp/images"
	err := os.Mkdir(testImagesPath, 0777)
	assert.Nil(t, err)
	defer func() {
		// cleanup fake images
		_ = os.RemoveAll(testImagesPath)
	}()

	var mockexec = &mocks.GoMockExecutor{}
	var manager = NewLoopBackManager(mockexec, logger)
	// Clean devices after default initialization in constructor
	manager.devices = make([]*LoopBackDevice, 0)

	// image of device that should be recovered from default config
	testSerialNumber := "12345"
	_, err = os.Create(fmt.Sprintf("%s/%s-%s.img", testImagesPath, manager.hostname, testSerialNumber))
	assert.Nil(t, err)

	manager.attemptToRecoverDevices(testImagesPath)
	assert.Equal(t, len(manager.devices), 1)
}
