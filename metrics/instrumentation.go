package metrics

import "github.com/prometheus/client_golang/prometheus"

// InstrumentationType is the type of instrumentation the metric is capturing
// Use this type to define your own instrumentation types e.g.:
// const (
//     InstrumentationTypeHttpRequests InstrumentationType = iota
//     InstrumentationTypeDatabaseQueries
//     InstrumentationTypeCacheHits
//     InstrumentationTypeCacheMisses
// )
type InstrumentationType uint64

const (
	InstrumentationTypeVersion InstrumentationType = iota
	InstrumentationTypeHttpRequestCount
	InstrumentationTypeHttpRequestDuration
	InstrumentationTypeHttpRequestSuccess
	InstrumentationTypeHttpRequestFailure
	InstrumentationTypeGrpcRequestCount
	InstrumentationTypeGrpcRequestDuration
	InstrumentationTypeGrpcRequestSuccess
	InstrumentationTypeGrpcRequestFailure
	InstrumentationTypeDbRequestCount
	InstrumentationTypeDbRequestDuration
	InstrumentationTypeDbRequestSuccess
	InstrumentationTypeDbRequestFailure
	InstrumentationTypeCacheRequestCount
	InstrumentationTypeCacheRequestDuration
	InstrumentationTypeCacheRequestSuccess
	InstrumentationTypeCacheRequestFailure
	InstrumentationTypeSubscriberCount
	InstrumentationTypeNetworkRequestCount
	InstrumentationTypeNetworkRequestDuration
	InstrumentationTypeNetworkRequestSuccess
	InstrumentationTypeNetworkRequestFailure
)

type Instrumentation struct {
	namespace     string
	Counters      map[InstrumentationType]prometheus.Counter
	CounterVecs   map[InstrumentationType]*prometheus.CounterVec
	Gauges        map[InstrumentationType]prometheus.Gauge
	GaugeVecs     map[InstrumentationType]*prometheus.GaugeVec
	Histograms    map[InstrumentationType]prometheus.Histogram
	HistogramVecs map[InstrumentationType]*prometheus.HistogramVec
	Summaries     map[InstrumentationType]prometheus.Summary
	SummaryVecs   map[InstrumentationType]*prometheus.SummaryVec
}

func NewInstrumentation(namespace string, opts ...InstrumentationOption) *Instrumentation {
	instrumentation := &Instrumentation{
		namespace:     namespace,
		Counters:      make(map[InstrumentationType]prometheus.Counter),
		CounterVecs:   make(map[InstrumentationType]*prometheus.CounterVec),
		Gauges:        make(map[InstrumentationType]prometheus.Gauge),
		GaugeVecs:     make(map[InstrumentationType]*prometheus.GaugeVec),
		Histograms:    make(map[InstrumentationType]prometheus.Histogram),
		HistogramVecs: make(map[InstrumentationType]*prometheus.HistogramVec),
		Summaries:     make(map[InstrumentationType]prometheus.Summary),
		SummaryVecs:   make(map[InstrumentationType]*prometheus.SummaryVec),
	}

	for _, opt := range opts {
		opt(instrumentation)
	}
	return instrumentation
}

func (i *Instrumentation) Collectors() (collectors []prometheus.Collector) {
	for _, counters := range i.Counters {
		collectors = append(collectors, counters)
	}
	for _, counterVecs := range i.CounterVecs {
		collectors = append(collectors, counterVecs)
	}
	for _, gauges := range i.Gauges {
		collectors = append(collectors, gauges)
	}
	for _, gaugeVecs := range i.GaugeVecs {
		collectors = append(collectors, gaugeVecs)
	}
	for _, histograms := range i.Histograms {
		collectors = append(collectors, histograms)
	}
	for _, histogramVecs := range i.HistogramVecs {
		collectors = append(collectors, histogramVecs)
	}
	for _, summaries := range i.Summaries {
		collectors = append(collectors, summaries)
	}
	for _, summaryVecs := range i.SummaryVecs {
		collectors = append(collectors, summaryVecs)
	}
	return
}
