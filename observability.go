package fail

import "context"

// ----------------------------- //
// -----------Tracing----------- //
// ----------------------------- //

// Tracer allows users to provide their own tracing solution
type Tracer interface {
	// Record records an error occurrence (simple version)
	Record(err *Error) *Error

	// RecordCtx records an error with context (for spans, baggage, etc.)
	RecordCtx(ctx context.Context, err *Error) *Error
}

// SetTracer sets the custom tracing solution to the global registry
func SetTracer(tracer Tracer) {
	global.SetTracer(tracer)
}

func (r *Registry) SetTracer(tracer Tracer) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tracer = tracer
}

// Record automatically traces the error using the configured tracer
func Record(e *Error) *Error {
	global.hooks.runTrace(e, map[string]any{
		"id":        e.ID.String(),
		"domain":    e.ID.Domain(),
		"level":     e.ID.Level(),
		"message":   e.Message,
		"is_system": e.IsSystem,
		"source":    "record",
	})

	global.mu.RLock()
	tracer := global.tracer
	global.mu.RUnlock()

	if tracer != nil {
		_ = tracer.Record(e)
	}

	return e
}

func RecordCtx(ctx context.Context, e *Error) *Error {
	global.hooks.runTrace(e, map[string]any{
		"id":        e.ID.String(),
		"domain":    e.ID.Domain(),
		"level":     e.ID.Level(),
		"message":   e.Message,
		"is_system": e.IsSystem,
		"source":    "recordCtx",
	})

	global.mu.RLock()
	tracer := global.tracer
	global.mu.RUnlock()

	if tracer != nil {
		_ = tracer.RecordCtx(ctx, e)
	}

	return e
}

// Record automatically traces the error using the configured tracer
func (e *Error) Record() *Error {
	// Get registry first (with fallback)
	reg := e.registry
	if reg == nil {
		reg = global
	}

	reg.hooks.runTrace(e, map[string]any{
		"id":        e.ID.String(),
		"domain":    e.ID.Domain(),
		"level":     e.ID.Level(),
		"message":   e.Message,
		"is_system": e.IsSystem,
		"source":    "record",
	})

	global.mu.RLock()
	tracer := global.tracer
	global.mu.RUnlock()

	if tracer != nil {
		_ = tracer.Record(e)
	}

	return e
}

func (e *Error) RecordCtx(ctx context.Context) *Error {
	// Get registry first (with fallback)
	reg := e.registry
	if reg == nil {
		reg = global
	}

	reg.hooks.runTrace(e, map[string]any{
		"id":        e.ID.String(),
		"domain":    e.ID.Domain(),
		"level":     e.ID.Level(),
		"message":   e.Message,
		"is_system": e.IsSystem,
		"source":    "recordCtx",
	})

	global.mu.RLock()
	tracer := global.tracer
	global.mu.RUnlock()

	if tracer != nil {
		_ = tracer.RecordCtx(ctx, e)
	}

	return e
}

// ------------------------------ //
// -------------Logs------------- //
// ------------------------------ //

// Logger allows users to provide their own logging solution
type Logger interface {
	Log(err *Error)
	LogCtx(ctx context.Context, err *Error)
}

// SetLogger sets the custom logging solution to the registry
func SetLogger(logger Logger) {
	global.SetLogger(logger)
}

func (r *Registry) SetLogger(logger Logger) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.logger = logger
}

// Log automatically logs the error using the configured logger
func (e *Error) Log() *Error {
	// Get registry first (with fallback)
	reg := e.registry
	if reg == nil {
		reg = global
	}

	// Run hook regardless of logger config
	reg.hooks.runLog(e, map[string]any{
		"id":        e.ID.String(),
		"domain":    e.ID.Domain(),
		"level":     e.ID.Level(),
		"message":   e.Message,
		"is_system": e.IsSystem,
		"source":    "log",
	})

	// Logging is separate concern
	reg.mu.RLock()
	logger := reg.logger
	reg.mu.RUnlock()

	if logger != nil {
		logger.Log(e)
	}

	return e
}

func (e *Error) LogCtx(ctx context.Context) *Error {
	// Get registry first (with fallback)
	reg := e.registry
	if reg == nil {
		reg = global
	}

	// Run hook regardless of logger config
	reg.hooks.runLog(e, map[string]any{
		"id":        e.ID.String(),
		"domain":    e.ID.Domain(),
		"level":     e.ID.Level(),
		"message":   e.Message,
		"is_system": e.IsSystem,
		"source":    "logCtx",
	})

	// Logging is separate concern
	reg.mu.RLock()
	logger := reg.logger
	reg.mu.RUnlock()

	if logger != nil {
		logger.LogCtx(ctx, e)
	}

	return e
}

// LogAndRecord logs and traces the error
func (e *Error) LogAndRecord() *Error {
	return e.Log().Record()
}

func (e *Error) LogAndRecordCtx(ctx context.Context) *Error {
	return e.LogCtx(ctx).RecordCtx(ctx)
}
