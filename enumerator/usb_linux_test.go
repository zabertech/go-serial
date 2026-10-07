package enumerator

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestReadLatencyTimer(t *testing.T) {
	r := require.New(t)
	dir := t.TempDir()

	r.Nil(readLatencyTimer(dir))

	r.NoError(os.WriteFile(filepath.Join(dir, "latency_timer"), []byte("16\n"), 0o644))
	latency := readLatencyTimer(dir)
	r.NotNil(latency)
	r.Equal(16*time.Millisecond, *latency)

	r.NoError(os.WriteFile(filepath.Join(dir, "latency_timer"), []byte("abc\n"), 0o644))
	r.Nil(readLatencyTimer(dir))
}
