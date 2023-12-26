package otelsql

import (
	"context"
	"database/sql/driver"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	semconv "go.opentelemetry.io/otel/semconv/v1.20.0"
)

// float64Recorder adds a new value to the list of Histogram's records.
type float64Recorder = func(ctx context.Context, value float64, opts ...metric.RecordOption)

// int64Counter adds the value to the counter's sum.
type int64Counter = func(ctx context.Context, value int64, opts ...metric.AddOption)

// methodRecorder records metrics about a sql method.
type methodRecorder interface {
	Record(ctx context.Context, method string, labels ...attribute.KeyValue) func(err error, attrs ...attribute.KeyValue)
}

type queryRecorder func(ctx context.Context, query string, args []driver.NamedValue) []attribute.KeyValue

type methodRecorderImpl struct {
	recordLatency float64Recorder
	countCalls    int64Counter

	attributes []attribute.KeyValue
}

func (r methodRecorderImpl) Record(ctx context.Context, method string, labels ...attribute.KeyValue) func(err error, attrs ...attribute.KeyValue) {
	startTime := time.Now()

	attrs := make([]attribute.KeyValue, 0, len(r.attributes)+len(labels)+2)

	attrs = append(attrs, r.attributes...)
	attrs = append(attrs, labels...)
	attrs = append(attrs, semconv.DBOperationKey.String(method))

	return func(err error, labels ...attribute.KeyValue) {
		elapsedTime := millisecondsSince(startTime)

		if err == nil {
			attrs = append(attrs, labels...)
			attrs = append(attrs, dbSQLStatusOK)
		} else {
			attrs = append(attrs, labels...)
			attrs = append(attrs, dbSQLStatusERROR,
				dbSQLError.String(err.Error()),
			)
		}

		r.countCalls(ctx, 1, metric.WithAttributes(attrs...))
		r.recordLatency(ctx, elapsedTime, metric.WithAttributes(attrs...))
	}
}

func newMethodRecorder(
	latencyRecorder float64Recorder,
	callsCounter int64Counter,
	attrs ...attribute.KeyValue,
) methodRecorderImpl {
	return methodRecorderImpl{
		recordLatency: latencyRecorder,
		countCalls:    callsCounter,
		attributes:    attrs,
	}
}

func recordNoQuery(context.Context, string, []driver.NamedValue) []attribute.KeyValue { return nil }
