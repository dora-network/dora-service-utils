package metrics

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
)

type (
	Option                func(*Server)
	InstrumentationOption func(instrumentation *Instrumentation)
)

func WithLogger(logger zerolog.Logger) Option {
	return func(s *Server) {
		s.log = logger
	}
}

func WithEnabled(enabled bool) Option {
	return func(s *Server) {
		s.enabled = enabled
	}
}

func WithRegistry(reg *prometheus.Registry) Option {
	return func(s *Server) {
		s.reg = reg
	}
}

func WithServer(server *http.Server) Option {
	return func(s *Server) {
		s.srv = server
	}
}

func WithPrometheusHandlerOpts(opts promhttp.HandlerOpts) Option {
	return func(s *Server) {
		s.opts = opts
	}
}

func WithPath(path string) Option {
	return func(s *Server) {
		s.path = path
	}
}

func WithPort(port int) Option {
	return func(s *Server) {
		s.port = port
	}
}

func WithHost(host string) Option {
	return func(s *Server) {
		s.host = host
	}
}

func WithHttpTimeout(timeout time.Duration) Option {
	return func(s *Server) {
		s.httpReadTimeout = timeout
	}
}

func WithHttpHeaderTimeout(timeout time.Duration) Option {
	return func(s *Server) {
		s.httpReadHeaderTimeout = timeout
	}
}

func WithCounter(instrumentationType InstrumentationType, name, help string) InstrumentationOption {
	return func(instrumentation *Instrumentation) {
		instrumentation.Counters[instrumentationType] = prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: instrumentation.namespace,
			Name:      name,
			Help:      help,
		})
	}
}

func WithCounterVec(instrumentationType InstrumentationType, name, help string, labels []string) InstrumentationOption {
	return func(instrumentation *Instrumentation) {
		instrumentation.CounterVecs[instrumentationType] = prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: instrumentation.namespace,
			Name:      name,
			Help:      help,
		}, labels)
	}
}

func WithGauge(instrumentationType InstrumentationType, name, help string) InstrumentationOption {
	return func(instrumentation *Instrumentation) {
		instrumentation.Gauges[instrumentationType] = prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: instrumentation.namespace,
			Name:      name,
			Help:      help,
		})
	}
}

func WithGaugeVec(instrumentationType InstrumentationType, name, help string, labels []string) InstrumentationOption {
	return func(instrumentation *Instrumentation) {
		instrumentation.GaugeVecs[instrumentationType] = prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: instrumentation.namespace,
			Name:      name,
			Help:      help,
		}, labels)
	}
}

func WithHistogram(instrumentationType InstrumentationType, name, help string, buckets []float64) InstrumentationOption {
	return func(instrumentation *Instrumentation) {
		instrumentation.Histograms[instrumentationType] = prometheus.NewHistogram(prometheus.HistogramOpts{
			Namespace: instrumentation.namespace,
			Name:      name,
			Help:      help,
			Buckets:   buckets,
		})
	}
}

func WithHistogramVec(instrumentationType InstrumentationType, name, help string, labels []string, buckets []float64) InstrumentationOption {
	return func(instrumentation *Instrumentation) {
		instrumentation.HistogramVecs[instrumentationType] = prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: instrumentation.namespace,
			Name:      name,
			Help:      help,
			Buckets:   buckets,
		}, labels)
	}
}

func WithSummary(instrumentationType InstrumentationType, name, help string, objectives map[float64]float64) InstrumentationOption {
	return func(instrumentation *Instrumentation) {
		instrumentation.Summaries[instrumentationType] = prometheus.NewSummary(prometheus.SummaryOpts{
			Namespace:  instrumentation.namespace,
			Name:       name,
			Help:       help,
			Objectives: objectives,
		})
	}
}

func WithSummaryVec(instrumentationType InstrumentationType, name, help string, labels []string, objectives map[float64]float64) InstrumentationOption {
	return func(instrumentation *Instrumentation) {
		instrumentation.SummaryVecs[instrumentationType] = prometheus.NewSummaryVec(prometheus.SummaryOpts{
			Namespace:  instrumentation.namespace,
			Name:       name,
			Help:       help,
			Objectives: objectives,
		}, labels)
	}
}
