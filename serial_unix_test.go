//go:build linux || darwin

package serial

import (
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/sys/unix"
)

var requiredBaudrates = []int{9600, 19200, 38400, 57600, 115200, 460800, 921600}

func TestSetTermSettingsBaudrate(t *testing.T) {
	const otherCflags = unix.CREAD | unix.CLOCAL | unix.CS8 | unix.PARENB
	for _, speed := range requiredBaudrates {
		settings := &unix.Termios{Cflag: otherCflags}
		custom, err := setTermSettingsBaudrate(speed, settings)
		require.NoError(t, err, "baudrate %d", speed)

		expected, standard := baudrateMap[speed]
		require.Equal(t, !standard, custom, "baudrate %d", speed)
		if custom {
			expected = baudrateMap[customBaudratePlaceholder]
		}
		require.Equal(t, toTermiosSpeedType(expected), settings.Ispeed, "baudrate %d", speed)
		require.Equal(t, toTermiosSpeedType(expected), settings.Ospeed, "baudrate %d", speed)
		require.Equal(t, otherCflags|(expected&tcCBAUD), settings.Cflag, "baudrate %d", speed)
	}
}

func TestSetTermSettingsBaudrateReplacesPrevious(t *testing.T) {
	settings := &unix.Termios{}
	_, err := setTermSettingsBaudrate(921600, settings)
	require.NoError(t, err)
	_, err = setTermSettingsBaudrate(9600, settings)
	require.NoError(t, err)
	require.Equal(t, baudrateMap[9600]&tcCBAUD, settings.Cflag)
	require.Equal(t, toTermiosSpeedType(baudrateMap[9600]), settings.Ispeed)
}

func TestSetTermSettingsBaudrateInvalid(t *testing.T) {
	invalid := []int{-1}
	if !customBaudrateSupported {
		invalid = append(invalid, 12345)
	}
	for _, speed := range invalid {
		_, err := setTermSettingsBaudrate(speed, &unix.Termios{})
		require.Error(t, err, "baudrate %d", speed)
		require.Equal(t, InvalidSpeed, err.(*PortError).Code())
	}
}
