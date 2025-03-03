package metrics

type CollectorType uint8

const (
	_ CollectorType = iota
	CollectorTypeCounter
	CollectorTypeCounterVec
	CollectorTypeGauge
	CollectorTypeGaugeVec
	CollectorTypeHistogram
	CollectorTypeHistogramVec
	CollectorTypeSummary
	CollectorTypeSummaryVec
)
