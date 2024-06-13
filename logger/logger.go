package logger

import (
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/diode"
	"log"
	"os"
	"time"
)

var (
	logFile *os.File
)

// New creates a new zerolog logger with the given log level and log file path.
func New(logLevel, logFilePath string, writeToConsole bool) (zerolog.Logger, error) {
	var (
		err error
		lvl zerolog.Level
	)

	if logFile != nil {
		if !writeToConsole {
			return zerolog.New(logFile).Level(lvl).With().Timestamp().Logger(), nil
		}
		multi := zerolog.MultiLevelWriter(zerolog.ConsoleWriter{Out: os.Stdout}, logFile)
		return zerolog.New(multi).Level(lvl).With().Timestamp().Logger(), nil
	}

	logFile, err = os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return zerolog.Nop(), err
	}

	lvl, err = zerolog.ParseLevel(logLevel)
	if err != nil || len(lvl.String()) == 0 {
		lvl = zerolog.InfoLevel
	}

	if !writeToConsole {
		return zerolog.New(logFile).Level(lvl).With().Timestamp().Logger(), nil
	}

	multi := zerolog.MultiLevelWriter(zerolog.ConsoleWriter{Out: os.Stdout}, logFile)
	return zerolog.New(multi).Level(lvl).With().Timestamp().Logger(), nil
}

// Close closes the log file.
// Close should be called in a defer statement after the logger is created to ensure that
// the log file is closed when the program exits.
func Close() error {
	return logFile.Close()
}

func init() {
	zerolog.TimeFieldFormat = time.RFC3339Nano
}

func TestLogger() zerolog.Logger {
	return zerolog.New(os.Stdout).Level(zerolog.DebugLevel).With().Timestamp().Logger()
}

func NewThreadSafeLogger(logLevel, logFilePath string, writeToConsole bool) (zerolog.Logger, error) {
	var (
		err error
		lvl zerolog.Level
	)

	if logFile != nil {
		wr := diode.NewWriter(logFile, 1000, 10*time.Millisecond, func(missed int) {
			log.Printf("Logger Dropped %d messages", missed)
		})
		if !writeToConsole {
			return zerolog.New(wr).Level(lvl).With().Timestamp().Logger(), nil
		}
		multi := zerolog.MultiLevelWriter(zerolog.ConsoleWriter{Out: os.Stdout}, wr)
		return zerolog.New(multi).Level(lvl).With().Timestamp().Logger(), nil
	}

	logFile, err = os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return zerolog.Nop(), err
	}
	lvl, err = zerolog.ParseLevel(logLevel)
	if err != nil || len(lvl.String()) == 0 {
		lvl = zerolog.InfoLevel
	}

	wr := diode.NewWriter(logFile, 1000, 10*time.Millisecond, func(missed int) {
		log.Printf("Logger Dropped %d messages", missed)
	})

	if !writeToConsole {
		return zerolog.New(wr).Level(lvl).With().Timestamp().Logger(), nil
	}
	multi := zerolog.MultiLevelWriter(zerolog.ConsoleWriter{Out: os.Stdout}, wr)
	return zerolog.New(multi).Level(lvl).With().Timestamp().Logger(), nil
}
