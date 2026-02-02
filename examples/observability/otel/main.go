package main

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	oteltrace "go.opentelemetry.io/otel/trace"
)

var (
	UserUsernameEmpty = fail.ID(0, "USER", 0, true, "UserUsernameEmpty")
	_                 = fail.Form(UserUsernameEmpty, "username cannot be empty", false, nil)
)

type OTelTracer struct {
	Tracer              oteltrace.Tracer
	RecordSystemAsError bool
	RecordDomainAsEvent bool
}

func DefaultOTelTracer() *OTelTracer {
	return &OTelTracer{
		Tracer:              otel.Tracer("fail-example"),
		RecordSystemAsError: true,
		RecordDomainAsEvent: true,
	}
}

func (o *OTelTracer) Trace(operation string, fn func() error) error {
	return o.TraceCtx(context.Background(), operation, func(context.Context) error {
		return fn()
	})
}

func (o *OTelTracer) TraceCtx(ctx context.Context, operation string, fn func(context.Context) error) error {
	ctx, span := o.Tracer.Start(ctx, operation)
	defer span.End()

	err := fn(ctx)
	if err == nil {
		return nil
	}

	o.recordError(span, err)
	return err
}

func (o *OTelTracer) recordError(span oteltrace.Span, err error) {
	e := fail.From(err)

	attrs := []attribute.KeyValue{
		attribute.String("error.id", e.ID.String()),
		attribute.String("error.message", e.Message),
		attribute.Bool("error.is_system", e.IsSystem),
	}

	if e.IsSystem && o.RecordSystemAsError {
		span.SetStatus(codes.Error, e.Message)
		span.SetAttributes(attrs...)
		span.RecordError(e)
		return
	}

	if !e.IsSystem && o.RecordDomainAsEvent {
		span.AddEvent("domain.error", oteltrace.WithAttributes(attrs...))
		return
	}

	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
}

func main() {
	exporter := tracetest.NewInMemoryExporter()
	tp := trace.NewTracerProvider(trace.WithSyncer(exporter))
	otel.SetTracerProvider(tp)

	fail.SetTracer(DefaultOTelTracer())

	fmt.Println("=== OpenTelemetry Tracer Example ===")

	// Use fail.New(ID) for recording to ensure a fresh instance context
	_ = fail.New(UserUsernameEmpty).Record()

	spans := exporter.GetSpans()
	fmt.Printf("✅ Spans recorded: %d\n", len(spans))
	for _, s := range spans {
		fmt.Printf("Span: %s\n", s.Name)
		for _, event := range s.Events {
			fmt.Printf("  Event: %s\n", event.Name)
		}
	}
}
