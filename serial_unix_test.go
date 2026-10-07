//go:build linux || darwin

package serial

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/sys/unix"
)

var requiredBaudrates = []int{9600, 19200, 38400, 57600, 115200, 460800, 921600}

func TestSetTermSettingsBaudrate(t *testing.T) {
	const otherCflags = unix.CREAD | unix.CLOCAL | unix.CS8 | unix.PARENB
	for _, speed := range requiredBaudrates {
		settings := &termSettings{termios: unix.Termios{Cflag: otherCflags}}
		err := setTermSettingsBaudrate(speed, settings)
		require.NoError(t, err, "baudrate %d", speed)

		if expected, standard := baudrateMap[speed]; standard {
			require.Equal(t, 0, settings.customBaudrate, "baudrate %d", speed)
			require.Equal(t, toTermiosSpeedType(expected), settings.termios.Ispeed, "baudrate %d", speed)
			require.Equal(t, toTermiosSpeedType(expected), settings.termios.Ospeed, "baudrate %d", speed)
			want := unix.Termios{Cflag: otherCflags}
			if runtime.GOOS == "linux" {
				want.Cflag |= expected
			}
			require.Equal(t, want.Cflag, settings.termios.Cflag, "baudrate %d", speed)
		} else {
			require.Equal(t, speed, settings.customBaudrate, "baudrate %d", speed)
			require.Equal(t, unix.Termios{Cflag: otherCflags}, settings.termios, "baudrate %d", speed)
		}
	}
}

func TestSetTermSettingsBaudrateReplacesPrevious(t *testing.T) {
	settings := &termSettings{}
	require.NoError(t, setTermSettingsBaudrate(921600, settings))
	require.NoError(t, setTermSettingsBaudrate(9600, settings))
	require.Equal(t, 0, settings.customBaudrate)
	if runtime.GOOS == "linux" {
		require.Equal(t, baudrateMap[9600], settings.termios.Cflag)
	} else {
		require.Zero(t, settings.termios.Cflag)
	}
	require.Equal(t, toTermiosSpeedType(baudrateMap[9600]), settings.termios.Ispeed)
}

func TestSetTermSettingsBaudrateInvalid(t *testing.T) {
	invalid := []int{-1}
	if runtime.GOOS == "linux" {
		invalid = append(invalid, 12345)
	}
	for _, speed := range invalid {
		err := setTermSettingsBaudrate(speed, &termSettings{})
		require.Error(t, err, "baudrate %d", speed)
		require.Equal(t, InvalidSpeed, err.(*PortError).Code())
	}
}
