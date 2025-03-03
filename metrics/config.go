package metrics

import "time"

type Config struct {
	Enabled           bool          `mapstructure:"enabled"`
	Path              string        `mapstructure:"path"`
	Host              string        `mapstructure:"host"`
	Port              int           `mapstructure:"port"`
	HttpTimeout       time.Duration `mapstructure:"http_timeout"`
	HttpHeaderTimeout time.Duration `mapstructure:"http_header_timeout"`
}

func DefaultConfig() Config {
	return Config{
		Enabled:           true,
		Path:              "/metrics",
		Host:              "",
		Port:              8081,
		HttpTimeout:       time.Minute,
		HttpHeaderTimeout: time.Minute,
	}
}
