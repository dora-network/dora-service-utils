package logger_test

import (
	"github.com/dora-network/dora-service-utils/logger"
	"github.com/stretchr/testify/require"
	"os"
	"sync"
	"testing"
	"time"
)

func TestLogger(t *testing.T) {
	l1, err := logger.New("debug", "test.log", false)
	require.NoError(t, err)
	l2, err := logger.NewThreadSafeLogger("debug", "test.log", false)
	require.NoError(t, err)
	wg := sync.WaitGroup{}
	wg.Add(2)
	go func() {
		l1.Info().Msg("test1")
		wg.Done()
	}()

	go func() {
		l2.Info().Msg("test2")
		// Need to give it some time to write to the file
		time.Sleep(20 * time.Millisecond)
		wg.Done()
	}()

	wg.Wait()

	require.NoError(t, logger.Close())

	contents, err := os.ReadFile("test.log")
	require.NoError(t, err)

	require.Contains(t, string(contents), "test1")
	require.Contains(t, string(contents), "test2")
	require.NoError(t, os.Remove("test.log"))
}
