package serial

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSetTermSettingsBaudrateTooLarge(t *testing.T) {
	err := setTermSettingsBaudrate(math.MaxInt32+1, &termSettings{})
	require.Error(t, err)
	require.Equal(t, InvalidSpeed, err.(*PortError).Code())
}
