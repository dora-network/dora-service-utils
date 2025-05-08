package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/diode"
	"gitlab.com/efronlicht/enve"
)

var (
	global     atomic.Pointer[zerolog.Logger] // global, shared logger.
	once       sync.Once                      // guards global.
	logFile    *os.File
	logfileErr error
)

func must[T any](v T, err error) T {
	if err != nil {
		log.Fatal(err)
	}
	return v
}

// Logfile returns the log file for this instance of the program, if any.
// It is safe to call this function from multiple goroutines, but accesses to the file are not synchronized.
// You generally shouldn't use this function directly.
func Logfile() (*os.File, error) {
	initLogger()
	return logFile, logfileErr
}

func initLogger() {
	once.Do(func() {
		servicename := enve.StringOr("DORA_SERVICE_NAME", "dora-unknown")

		var w io.Writer
		var logger zerolog.Logger

		// init w and the global logFile.
		// we'll log both to a log file ($DORA_LOG_DIR/<service_name>_YYYY-MM-DD_HH:MM:SS.log) and to stdout.
		// the logger should never crash or stall the program, so we use a diode to buffer the writes.
		// if the file can't be created, we'll just log to stdout and warn the user.
		{
			const size, pollInterval = 1024, 15 * time.Millisecond
			dir := enve.StringOr("DORA_LOG_DIR", ".")

			if logfileErr = os.MkdirAll(dir, 0o755); logfileErr != nil {
				defer logger.Warn().Err(logfileErr).Msg("logfile is not being used, check DORA_LOG_DIR and DORA_LOG_LEVEL env vars")
				w = diode.NewWriter(os.Stdout, size, pollInterval, func(missed int) { log.Printf("diode: dropped %d log messages", missed) }) // use diode so we don't have racy writes and safely drop messages
			} else if logFile, logfileErr = os.Create(filepath.Join(dir, fmt.Sprintf("%s_%s.log", servicename, time.Now().Format(time.RFC3339)))); logfileErr != nil {
				defer logger.Warn().Err(logfileErr).Msg("logfile is not being used, check DORA_LOG_DIR and DORA_LOG_LEVEL env vars")
				w = diode.NewWriter(os.Stdout, size, pollInterval, func(missed int) { log.Printf("diode: dropped %d log messages", missed) }) // use diode so we don't have racy writes and safely drop messages
			}

			// we have a file. wrap it in a zstd writer, then multiplex it with stdout, wrapping it in diode so it's thread-safe.
			w = diode.NewWriter(io.MultiWriter(logFile, os.Stdout), size, pollInterval, func(missed int) { log.Printf("diode: dropped %d log messages", missed) }) // use diode so we don't have racy writes and safely drop messages

		}
		logger = zerolog.New(w).
			Level(enve.Or(zerolog.ParseLevel, "DORA_LOG_LEVEL", zerolog.InfoLevel)).
			With().
			Timestamp().
			Str("instance_id", must(uuid.NewV7()).String()).Str("service", servicename).
			Logger()

		// write debug logs that give metadata about this program and it's logger
		dbglogger := logger.With().
			Int("gomaxprocs", runtime.GOMAXPROCS(0)).
			Str("goarch", runtime.GOARCH).
			Str("goos", runtime.GOOS).
			Str("user", enve.StringOr("USER", "unknown")).
			Logger()

		info, ok := debug.ReadBuildInfo()
		if ok {
			dbglogger.Info().Any("buildinfo", info).Msg("buildinfo dump")
		}
		dbglogger.Info().Msg("logger init")
		global.Store(&logger)
	})
}

// Global returns the global logger. This function initializes the logger exactly once.
// It is safe to call this function from multiple goroutines.
// The Global logger relies on the following environment variables:
//
//   - DORA_LOG_DIR: directory to write the log file to, defaults to the current directory. logs will be written to a file named "<service_name>_<timestamp>.log".
//   - DORA_LOG_LEVEL: the log level, defaults to "info". Possible values are "debug", "info", "warn", "error", "fatal", "panic".
//   - DORA_SERVICE_NAME: the name of the service, defaults to "dora-unknown"
//   - USER: the user running the service, defaults to "unknown"
//
// This list may change in the future.
func Global() *zerolog.Logger {
	initLogger()
	return global.Load()
}

// Add fields to the global logger, thread-safe. Avoid this where possible, but sometimes it's handy.
func AddFieldsToGlobal(fields map[string]any) {
	for {
		old := Global()
		newentry := old.With()
		for k, v := range fields {
			newentry = newentry.Any(k, v)
		}
		new := newentry.Logger()

		if global.CompareAndSwap(old, &new) {
			return
		}
		continue
	}
}
