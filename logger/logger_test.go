package logger

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestLogger(t *testing.T) {
	t.Setenv("DORA_SERVICE_NAME", "testservice")
	dir := t.TempDir()
	t.Setenv("DORA_LOG_DIR", dir)

	log := Global()
	wg := new(sync.WaitGroup)
	wg.Add(50)
	for i := 0; i < 50; i++ {
		go func() {
			defer wg.Done()
			log.Info().Msgf("test%d", i)
		}()
	}
	wg.Wait()

	file, err := Logfile()
	if err != nil {
		t.Fatal(err)
	}
	if file == nil {
		t.Fatal("logfile is nil")
	}

	time.Sleep(100 * time.Millisecond) // wait for the log file to be written
	file.Seek(0, 0)
	b, err := io.ReadAll(file)
	if err != nil {
		t.Fatal(err)
	}
	if len(b) == 0 {
		t.Fatal("logfile is empty")
	}
	s := string(b)
	for i := 0; i < 50; i++ {
		if !strings.Contains(s, fmt.Sprintf("test%d", i)) {
			t.Fatalf("logfile does not contain test%d", i)
		}
	}
	os.Remove(file.Name())
}
