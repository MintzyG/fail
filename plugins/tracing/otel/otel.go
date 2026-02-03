package otel

import (
	"context"
	"fmt"

	"github.com/MintzyG/fail/v3"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// Tracer implements fail.Tracer using OpenTelemetry
type Tracer struct {
	tracer trace.Tracer
	config Config
}

// RecordMode defines how to record errors
type RecordMode int

const (
	// RecordAsEvent always records as span event
	RecordAsEvent RecordMode = iota
	// RecordAsStatus always sets span status
	RecordAsStatus
	// RecordSmart decides based on error type (system vs domain)
	RecordSmart
)

// Config configures the OpenTelemetry tracer
type Config struct {
	// TracerName is the name to use for the tracer (default: "fail")
	TracerName string

	// CreateSpanIfMissing creates the span if its missing instead of skipping on a nil span
	CreateSpanIfMissing bool

	// Mode determines recording behavior
	Mode RecordMode

	// SystemRecordMode overrides Mode for system errors (nil = use Mode)
	SystemRecordMode *RecordMode

	// DomainRecordMode overrides Mode for domain errors (nil = use Mode)
	DomainRecordMode *RecordMode

	// IncludeTrace adds stack trace to span attributes
	IncludeTrace bool

	// AttributePrefix prefixes all attributes (default: "error")
	// do not include the dot '.' in the prefix
	AttributePrefix string

	// EventName customizes the event name (default: "error")
	EventName string

	// StatusDescription customizes status description
	// nil = use err.GetRendered()
	StatusDescription func(*fail.Error) string

	// CustomRecord allows full override of recording logic
	// if set, all other options are ignored
	CustomRecord func(span trace.Span, err *fail.Error, attrs []attribute.KeyValue) *fail.Error
}

// New creates a new OpenTelemetry tracer
func New(opts ...Option) *Tracer {
	config := Config{
		TracerName:      "fail",
		Mode:            RecordSmart,
		AttributePrefix: "error",
		EventName:       "error",
	}

	for _, opt := range opts {
		opt(&config)
	}

	return &Tracer{
		tracer: otel.Tracer(config.TracerName),
		config: config,
	}
}

// Option configures the tracer
type Option func(*Config)

// WithTracerName sets the tracer name
func WithTracerName(name string) Option {
	return func(c *Config) {
		c.TracerName = name
	}
}

// WithMode sets the default recording mode
func WithMode(mode RecordMode) Option {
	return func(c *Config) {
		c.Mode = mode
	}
}

// WithSystemMode sets mode specifically for system errors
func WithSystemMode(mode RecordMode) Option {
	return func(c *Config) {
		c.SystemRecordMode = &mode
	}
}

// WithDomainMode sets mode specifically for domain errors
func WithDomainMode(mode RecordMode) Option {
	return func(c *Config) {
		c.DomainRecordMode = &mode
	}
}

// WithStackTrace includes stack traces in attributes
func WithStackTrace() Option {
	return func(c *Config) {
		c.IncludeTrace = true
	}
}

// WithAttributePrefix sets the attribute prefix
func WithAttributePrefix(prefix string) Option {
	return func(c *Config) {
		c.AttributePrefix = prefix
	}
}

// WithEventName customizes the event name
func WithEventName(name string) Option {
	return func(c *Config) {
		c.EventName = name
	}
}

// WithStatusDescription customizes status description
func WithStatusDescription(fn func(*fail.Error) string) Option {
	return func(c *Config) {
		c.StatusDescription = fn
	}
}

// WithCustomRecord provides full control over recording
func WithCustomRecord(fn func(trace.Span, *fail.Error, []attribute.KeyValue) *fail.Error) Option {
	return func(c *Config) {
		c.CustomRecord = fn
	}
}

// Record records an error (creates a new span)
func (t *Tracer) Record(err *fail.Error) *fail.Error {
	ctx := context.Background()
	return t.RecordCtx(ctx, err)
}

// RecordCtx records an error with context (adds to current span if exists)
func (t *Tracer) RecordCtx(ctx context.Context, err *fail.Error) *fail.Error {
	span := trace.SpanFromContext(ctx)

	if !span.IsRecording() {
		return err
	}

	attrs := t.buildAttributes(err)

	// Custom recorder takes full control
	if t.config.CustomRecord != nil {
		return t.config.CustomRecord(span, err, attrs)
	}

	// Determine mode based on error type and config
	mode := t.resolveMode(err)

	switch mode {
	case RecordAsStatus:
		desc := err.GetRendered()
		if t.config.StatusDescription != nil {
			desc = t.config.StatusDescription(err)
		}
		span.SetStatus(codes.Error, desc)
		span.SetAttributes(attrs...)

	case RecordAsEvent:
		span.AddEvent(t.config.EventName, trace.WithAttributes(attrs...))

	case RecordSmart:
		// Should not happen if resolveMode works correctly
		span.AddEvent(t.config.EventName, trace.WithAttributes(attrs...))
	}

	return err
}

// resolveMode determines which mode to use based on error type and config
func (t *Tracer) resolveMode(err *fail.Error) RecordMode {
	// Check for specific overrides first
	if err.IsSystem && t.config.SystemRecordMode != nil {
		return *t.config.SystemRecordMode
	}
	if !err.IsSystem && t.config.DomainRecordMode != nil {
		return *t.config.DomainRecordMode
	}

	// Apply default mode, with special handling for Smart
	if t.config.Mode == RecordSmart {
		if err.IsSystem {
			return RecordAsStatus // System errors = status
		}
		return RecordAsEvent // Domain errors = event
	}

	return t.config.Mode
}

// buildAttributes creates span attributes from error (same as before)
func (t *Tracer) buildAttributes(err *fail.Error) []attribute.KeyValue {
	prefix := t.config.AttributePrefix

	attrs := []attribute.KeyValue{
		attribute.String(prefix+".id", err.ID.String()),
		attribute.Int(prefix+".level", err.ID.Level()),
		attribute.String(prefix+".domain", err.ID.Domain()),
		attribute.String(prefix+".message", err.GetRendered()),
		attribute.Bool(prefix+".is_system", err.IsSystem),
		attribute.Bool(prefix+".is_registered", err.IsRegistered()),
	}

	// Add internal message if present
	if internal := fail.GetInternalMessage(err); internal != "" {
		attrs = append(attrs, attribute.String(prefix+".internal_message", internal))
	}

	// Add metadata
	if err.Meta != nil {
		for key, value := range err.Meta {
			attrKey := prefix + ".meta." + key

			switch v := value.(type) {
			case string:
				attrs = append(attrs, attribute.String(attrKey, v))
			case int:
				attrs = append(attrs, attribute.Int(attrKey, v))
			case int64:
				attrs = append(attrs, attribute.Int64(attrKey, v))
			case float64:
				attrs = append(attrs, attribute.Float64(attrKey, v))
			case bool:
				attrs = append(attrs, attribute.Bool(attrKey, v))
			default:
				attrs = append(attrs, attribute.String(attrKey, fmt.Sprintf("%v", v)))
			}
		}
	}

	// Add stack trace if configured
	if t.config.IncludeTrace {
		if traces, ok := fail.GetTraces(err); ok {
			for i, tt := range traces {
				attrs = append(attrs,
					attribute.String(fmt.Sprintf("%s.trace.%d", prefix, i), tt),
				)
			}
		}
	}

	// Add validation errors if present
	if validations, ok := fail.GetValidations(err); ok {
		for i, v := range validations {
			attrs = append(attrs,
				attribute.String(fmt.Sprintf("%s.validation.%d.field", prefix, i), v.Field),
				attribute.String(fmt.Sprintf("%s.validation.%d.message", prefix, i), v.Message),
			)
		}
	}

	return attrs
}
