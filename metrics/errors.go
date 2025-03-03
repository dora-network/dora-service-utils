package metrics

import "errors"

var (
	ErrMetricsDisabled   = errors.New("metrics server is disabled")
	ErrMetricsRunning    = errors.New("metrics server is already running")
	ErrMetricsNotRunning = errors.New("metrics server is not running")
)
