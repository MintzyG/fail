package observability

import (
	"context"
	"errors"
	"fmt"

	"fail"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// OTelTracer integrates with OpenTelemetry
type OTelTracer struct {
	Tracer              trace.Tracer
	RecordSystemAsError bool // If true, system errors set span status to Error
	RecordDomainAsEvent bool // If true, domain errors are recorded as events
}

// DefaultOTelTracer returns a sensible default configuration
func DefaultOTelTracer() *OTelTracer {
	return &OTelTracer{
		Tracer:              otel.Tracer("fail"),
		RecordSystemAsError: true,
		RecordDomainAsEvent: true,
	}
}

// Trace implements observability.Tracer without context
func (o *OTelTracer) Trace(operation string, fn func() error) error {
	return o.TraceCtx(context.Background(), operation, func(context.Context) error {
		return fn()
	})
}

// TraceCtx implements observability.Tracer with context propagation
func (o *OTelTracer) TraceCtx(ctx context.Context, operation string, fn func(context.Context) error) error {
	if o.Tracer == nil {
		o.Tracer = otel.Tracer("fail")
	}

	ctx, span := o.Tracer.Start(ctx, operation)
	defer span.End()

	err := fn(ctx)
	if err == nil {
		return nil
	}

	o.recordError(span, err)
	return err
}

func (o *OTelTracer) recordError(span trace.Span, err error) {
	if span == nil || err == nil {
		return
	}

	e := fail.From(err)

	attrs := []attribute.KeyValue{
		attribute.String("error.id", e.ID.String()),
		attribute.String("error.message", e.Message),
		attribute.Bool("error.is_system", e.IsSystem),
	}

	if e.InternalMessage != "" {
		attrs = append(attrs, attribute.String("error.internal_message", e.InternalMessage))
	}

	if e.Meta != nil {
		for k, v := range e.Meta {
			attrs = append(attrs, attribute.String("error.meta."+k, fmt.Sprint(v)))
		}
	}

	if e.IsSystem && o.RecordSystemAsError {
		span.SetStatus(codes.Error, e.Message)
		span.SetAttributes(attrs...)

		if e.Cause != nil {
			span.RecordError(e.Cause)
		}
		return
	}

	if !e.IsSystem && o.RecordDomainAsEvent {
		span.AddEvent("domain.error", trace.WithAttributes(attrs...))
		return
	}

	// Fallback for non-fail errors
	var failErr *fail.Error
	if errors.As(err, &failErr) {
		span.RecordError(failErr)
		span.SetStatus(codes.Error, failErr.Message)
		return
	}

	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
}

var UserUsernameEmpty = fail.ID("UserUsernameEmpty", "USER", true, 0)
var ErrUserUsernameEmpty = fail.Form(UserUsernameEmpty, "username cannot be empty", false, nil)

var UserIDNotFound = fail.ID("UserIDNotFound", "ADMIN", true, 0)
var ErrUserNotFound = fail.Form(UserIDNotFound, "user not found", false, nil)

func CreateUser(tracer fail.Tracer, name string) error {
	return tracer.Trace("user.create", func() error {
		if name == "" {
			return ErrUserUsernameEmpty
		}

		// pretend we saved to DB here
		return nil
	})
}

func CreateUserCtx(ctx context.Context, tracer fail.Tracer, name string) error {
	return tracer.TraceCtx(ctx, "user.create", func(ctx context.Context) error {
		if name == "" {
			return ErrUserUsernameEmpty
		}

		// pretend we saved to DB here
		return nil
	})
}

type UserService struct {
	tracer fail.Tracer
}

func NewUserService(tracer fail.Tracer) *UserService {
	return &UserService{tracer: tracer}
}

func (s *UserService) CreateUser(name string) error {
	return s.tracer.Trace("user.create", func() error {
		if name == "" {
			return ErrUserUsernameEmpty
		}

		// pretend DB insert here
		return nil
	})
}

func (s *UserService) DeleteUser(ctx context.Context, id string) error {
	return s.tracer.TraceCtx(ctx, "user.delete", func(ctx context.Context) error {
		if id == "" {
			return ErrUserNotFound
		}

		// pretend DB delete here
		return nil
	})
}
