//
// Copyright 2014-2026 Cristian Maglie. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
//

package enumerator

import (
	"fmt"
	"regexp"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

func parseDeviceID(deviceID string, details *PortDetails) {
	// Windows stock USB-CDC driver
	if len(deviceID) >= 3 && deviceID[:3] == "USB" {
		re := regexp.MustCompile(`VID_(....)&PID_(....)(\\(\w+)$)?`).FindAllStringSubmatch(deviceID, -1)
		if re == nil || len(re[0]) < 2 {
			// Silently ignore unparsable strings
			return
		}
		details.IsUSB = true
		details.VID = re[0][1]
		details.PID = re[0][2]
		if len(re[0]) >= 4 {
			details.SerialNumber = re[0][4]
		}
		return
	}

	// FTDI driver
	if len(deviceID) >= 7 && deviceID[:7] == "FTDIBUS" {
		re := regexp.MustCompile(`VID_(....)\+PID_(....)(\+(\w+))?`).FindAllStringSubmatch(deviceID, -1)
		if re == nil || len(re[0]) < 2 {
			// Silently ignore unparsable strings
			return
		}
		details.IsUSB = true
		details.VID = re[0][1]
		details.PID = re[0][2]
		if len(re[0]) >= 4 {
			details.SerialNumber = re[0][4]
		}
		return
	}

	// Other unidentified device type
}

// setupapi based
// --------------

//sys setupDiGetClassDevs(guid *windows.GUID, enumerator *string, hwndParent uintptr, flags windows.DIGCF) (set windows.DevInfo, err error) = setupapi.SetupDiGetClassDevsW
//sys setupDiDestroyDeviceInfoList(set windows.DevInfo) (err error) = setupapi.SetupDiDestroyDeviceInfoList
//sys setupDiOpenDevRegKey(set windows.DevInfo, devInfo *windows.DevInfoData, scope windows.DICS_FLAG, hwProfile uint32, keyType windows.DIREG, samDesired uint32) (hkey syscall.Handle, err error) = setupapi.SetupDiOpenDevRegKey

//sys cmGetParent(outParentDev *windows.DEVINST, dev windows.DEVINST, flags uint32) (cmErr cmError) = cfgmgr32.CM_Get_Parent
//sys cmGetDeviceIDSize(outLen *uint32, dev windows.DEVINST, flags uint32) (cmErr cmError) = cfgmgr32.CM_Get_Device_ID_Size
//sys cmGetDeviceID(dev windows.DEVINST, buffer unsafe.Pointer, bufferSize uint32, flags uint32) (err cmError) = cfgmgr32.CM_Get_Device_IDW
//sys cmMapCrToWin32Err(cmErr cmError, defaultErr uint32) (err uint32) = cfgmgr32.CM_MapCrToWin32Err

type cmError uint32

func cmConvertError(cmErr cmError) error {
	if cmErr == 0 {
		return nil
	}
	winErr := cmMapCrToWin32Err(cmErr, 0)
	return fmt.Errorf("error %d", winErr)
}

func getParent(dev windows.DEVINST) (windows.DEVINST, error) {
	var res windows.DEVINST
	cmErr := cmGetParent(&res, dev, 0)
	return res, cmConvertError(cmErr)
}

func getDeviceID(dev windows.DEVINST) (string, error) {
	var size uint32
	cmErr := cmGetDeviceIDSize(&size, dev, 0)
	if err := cmConvertError(cmErr); err != nil {
		return "", err
	}
	buff := make([]uint16, size)
	cmErr = cmGetDeviceID(dev, unsafe.Pointer(&buff[0]), uint32(len(buff)), 0)
	if err := cmConvertError(cmErr); err != nil {
		return "", err
	}
	return windows.UTF16ToString(buff[:]), nil
}

type deviceInfo struct {
	set  windows.DevInfo
	data *windows.DevInfoData
}

func getDeviceInfo(set windows.DevInfo, index int) (*deviceInfo, error) {
	data, err := windows.SetupDiEnumDeviceInfo(set, index)
	if err != nil {
		return nil, err
	}
	return &deviceInfo{set: set, data: data}, nil
}

func (dev *deviceInfo) getInstanceID() (string, error) {
	return windows.SetupDiGetDeviceInstanceId(dev.set, dev.data)
}

func (dev *deviceInfo) openDevRegKey(scope windows.DICS_FLAG, hwProfile uint32, keyType windows.DIREG, samDesired uint32) (syscall.Handle, error) {
	return setupDiOpenDevRegKey(dev.set, dev.data, scope, hwProfile, keyType, samDesired)
}

func nativeGetDetailedPortsList() ([]*PortDetails, error) {
	guids, err := windows.SetupDiClassGuidsFromNameEx("Ports", "")
	if err != nil {
		return nil, &PortEnumerationError{causedBy: err}
	}

	var res []*PortDetails
	for _, g := range guids {
		devsSet, err := setupDiGetClassDevs(&g, nil, 0, windows.DIGCF_PRESENT)
		if err != nil {
			return nil, &PortEnumerationError{causedBy: err}
		}
		defer setupDiDestroyDeviceInfoList(devsSet)

		for i := 0; ; i++ {
			device, err := getDeviceInfo(devsSet, i)
			if err != nil {
				break
			}
			details := &PortDetails{}
			portName, err := retrievePortNameFromDevInfo(device)
			if err != nil {
				continue
			}
			if len(portName) < 3 || portName[0:3] != "COM" {
				// Accept only COM ports
				continue
			}
			details.Name = portName

			if err := retrievePortDetailsFromDevInfo(device, details); err != nil {
				return nil, &PortEnumerationError{causedBy: err}
			}
			res = append(res, details)
		}
	}
	return res, nil
}

func retrievePortNameFromDevInfo(device *deviceInfo) (string, error) {
	h, err := device.openDevRegKey(windows.DICS_FLAG_GLOBAL, 0, windows.DIREG_DEV, windows.KEY_READ)
	if err != nil {
		return "", err
	}
	defer syscall.RegCloseKey(h)

	var name [1024]uint16
	nameP := (*byte)(unsafe.Pointer(&name[0]))
	nameSize := uint32(len(name) * 2)
	if err := syscall.RegQueryValueEx(h, syscall.StringToUTF16Ptr("PortName"), nil, nil, nameP, &nameSize); err != nil {
		return "", err
	}
	return syscall.UTF16ToString(name[:]), nil
}

func retrievePortDetailsFromDevInfo(device *deviceInfo, details *PortDetails) error {
	deviceID, err := device.getInstanceID()
	if err != nil {
		return err
	}
	parseDeviceID(deviceID, details)

	// On composite USB devices the serial number is usually reported on the parent
	// device, so let's navigate up one level and see if we can get this information
	if details.IsUSB && details.SerialNumber == "" {
		if parentInfo, err := getParent(device.data.DevInst); err == nil {
			if parentDeviceID, err := getDeviceID(parentInfo); err == nil {
				d := &PortDetails{}
				parseDeviceID(parentDeviceID, d)
				if details.VID == d.VID && details.PID == d.PID {
					details.SerialNumber = d.SerialNumber
				}
			}
		}
	}

	return nil
}
