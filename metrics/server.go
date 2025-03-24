package metrics

import (
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
)

const (
	defaultReadTimeout       = time.Minute
	defaultReadHeaderTimeout = time.Minute
	defaultPort              = 8080
)

type Server struct {
	mu                    sync.Mutex
	srv                   *http.Server
	reg                   *prometheus.Registry
	log                   zerolog.Logger
	enabled               bool
	opts                  promhttp.HandlerOpts
	host                  string
	port                  int
	path                  string
	isRunning             bool
	httpReadTimeout       time.Duration
	httpReadHeaderTimeout time.Duration
}

func defaultServer() *Server {
	return &Server{
		enabled:               true,
		log:                   zerolog.Nop(),
		reg:                   prometheus.NewRegistry(),
		opts:                  promhttp.HandlerOpts{},
		port:                  defaultPort,
		path:                  "/metrics",
		httpReadTimeout:       defaultReadTimeout,
		httpReadHeaderTimeout: defaultReadHeaderTimeout,
	}
}

func NewServer(opts ...Option) *Server {
	s := defaultServer()
	applyOpts(s, opts)
	return s
}

func applyOpts(s *Server, opts []Option) {
	for _, opt := range opts {
		opt(s)
	}
}

func (s *Server) Path() string {
	if s.path == "" {
		return "/metrics"
	}
	return s.path
}

func (s *Server) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.enabled {
		return ErrMetricsDisabled
	}

	if s.srv != nil && s.isRunning {
		return ErrMetricsRunning
	}

	handler := promhttp.HandlerFor(s.reg, s.opts)
	mux := http.NewServeMux()
	mux.Handle(s.Path(), handler)

	s.srv = &http.Server{
		Addr:              fmt.Sprintf("%s:%d", s.host, s.port),
		Handler:           mux,
		ReadTimeout:       s.httpReadTimeout,
		ReadHeaderTimeout: s.httpReadHeaderTimeout,
	}

	go func() {
		s.log.Info().
			Str("host", s.host).
			Int("port", s.port).
			Str("path", s.Path()).
			Msg("starting metrics server")

		if err := s.srv.ListenAndServe(); err != nil && errors.Is(err, http.ErrServerClosed) {
			s.log.Error().Err(err).Msg("failed to start metrics server")
		}
	}()

	s.isRunning = true

	return nil
}

func (s *Server) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.enabled {
		return ErrMetricsDisabled
	}

	if s.srv == nil || !s.isRunning {
		return ErrMetricsNotRunning
	}

	if err := s.srv.Close(); err != nil {
		return err
	}

	s.isRunning = false
	return nil
}

func (s *Server) Registry() *prometheus.Registry {
	return s.reg
}

func (s *Server) HttpServer() *http.Server {
	return s.srv
}

func (s *Server) Register(instrumentation *Instrumentation) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, c := range instrumentation.Collectors() {
		if err := s.reg.Register(c); err != nil {
			return fmt.Errorf("failed to register collector: %w", err)
		}
	}

	return nil
}

func StartMetricsServer(config Config, instrumentation *Instrumentation, logger zerolog.Logger, version string) (err error) {
	metricsSvr := NewServer(
		WithLogger(logger),
		WithEnabled(config.Enabled),
		WithHost(config.Host),
		WithPort(config.Port),
		WithHttpTimeout(config.HttpTimeout),
		WithHttpHeaderTimeout(config.HttpHeaderTimeout),
	)
	if err = metricsSvr.Register(instrumentation); err != nil {
		logger.Err(err).
			Msg("failed to start metrics server")
		return
	}

	instrumentation.GaugeVecs[InstrumentationTypeVersion].With(prometheus.Labels{"version": version}).Set(1)
	if err = metricsSvr.Start(); err != nil {
		logger.Err(err).
			Msg("failed to start metrics server")
	}

	return
}
