package metrics

import (
	"connectrpc.com/connect"
	"context"
	"google.golang.org/grpc"
	"time"
)

func NewConnectInterceptor(
	instrumentation *Instrumentation,
) connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (
			connect.AnyResponse, error,
		) {
			instrumentation.CounterVecs[InstrumentationTypeHttpRequestCount].
				WithLabelValues(req.HTTPMethod(), req.Spec().Procedure).Inc()
			start := time.Now()
			resp, err := next(ctx, req)
			elapsed := time.Since(start).Milliseconds()
			instrumentation.HistogramVecs[InstrumentationTypeHttpRequestDuration].
				WithLabelValues(req.Spec().Procedure).Observe(float64(elapsed))
			if err != nil {
				instrumentation.CounterVecs[InstrumentationTypeHttpRequestFailure].
					WithLabelValues(req.Spec().Procedure).Inc()
			} else {
				instrumentation.CounterVecs[InstrumentationTypeHttpRequestSuccess].
					WithLabelValues(req.Spec().Procedure).Inc()
			}
			return resp, err
		}
	}
}

func NewGrpcInterceptor(
	instrumentation *Instrumentation,
) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		instrumentation.CounterVecs[InstrumentationTypeGrpcRequestCount].WithLabelValues(info.FullMethod).Inc()
		start := time.Now()
		resp, err = handler(ctx, req)
		elapsed := time.Since(start).Milliseconds()
		instrumentation.HistogramVecs[InstrumentationTypeGrpcRequestDuration].WithLabelValues(info.FullMethod).Observe(float64(elapsed))
		if err != nil {
			instrumentation.CounterVecs[InstrumentationTypeGrpcRequestFailure].WithLabelValues(info.FullMethod).Inc()
		} else {
			instrumentation.CounterVecs[InstrumentationTypeGrpcRequestSuccess].WithLabelValues(info.FullMethod).Inc()
		}
		return resp, err
	}
}
