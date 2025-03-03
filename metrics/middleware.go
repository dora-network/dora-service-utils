package metrics

import (
	"net/http"
	"time"
)

type HttpMiddleware struct {
	instrumentationType InstrumentationType
	instrumentation     *Instrumentation
}

func NewHttpMiddleware(instrumentationType InstrumentationType, instrumentation *Instrumentation) *HttpMiddleware {
	return &HttpMiddleware{
		instrumentationType: instrumentationType,
		instrumentation:     instrumentation,
	}
}

func (m *HttpMiddleware) Handle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		m.instrumentation.CounterVecs[m.instrumentationType].
			WithLabelValues(r.Method, r.URL.Path).Inc()
		start := time.Now()
		next.ServeHTTP(w, r)
		elapsed := time.Since(start).Milliseconds()
		m.instrumentation.HistogramVecs[m.instrumentationType].
			WithLabelValues(r.Method, r.URL.Path).Observe(float64(elapsed))
	})
}
