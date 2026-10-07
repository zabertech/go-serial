//
// Copyright 2014-2017 Cristian Maglie. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
//

package serial

import "golang.org/x/sys/unix"

const devFolder = "/dev"
const regexFilter = "^(cu|tty)\\..*"

const ioctlTcgetattr = unix.TIOCGETA
const ioctlTcsetattr = unix.TIOCSETA
const ioctlTcflsh = unix.TIOCFLUSH

// IOSSIOSPEED from IOKit/serial/ioss.h, used to set baudrates that are not
// accepted by TIOCSETA (anything above 230400)
const ioctlIOSSIOSPEED = 0x80045402

// Baudrates not in baudrateMap are applied with setCustomBaudrate
func setTermSettingsBaudrate(speed int, settings *termSettings) error {
	baudrate, ok := baudrateMap[speed]
	if !ok {
		if speed <= 0 {
			return &PortError{code: InvalidSpeed}
		}
		settings.customBaudrate = speed
		return nil
	}
	settings.customBaudrate = 0
	settings.termios.Ispeed = toTermiosSpeedType(baudrate)
	settings.termios.Ospeed = toTermiosSpeedType(baudrate)
	return nil
}

func (port *unixPort) setCustomBaudrate(speed int) error {
	return unix.IoctlSetPointerInt(port.handle, ioctlIOSSIOSPEED, speed)
}
