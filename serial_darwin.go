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

// tcCBAUD is the mask of the baudrate bits in Cflag: the baudrate is stored
// only in Ispeed/Ospeed, so no Cflag bits are used
const tcCBAUD = 0

// IOSSIOSPEED from IOKit/serial/ioss.h, used to set baudrates that are not
// accepted by TIOCSETA (anything above 230400)
const ioctlIOSSIOSPEED = 0x80045402

const customBaudrateSupported = true

func (port *unixPort) setCustomBaudrate(speed int) error {
	return unix.IoctlSetPointerInt(port.handle, ioctlIOSSIOSPEED, speed)
}
