package fail

import "context"

// ----------------------------- //
// -----------Tracing----------- //
// ----------------------------- //

// FIXME make interfaces actually work with any solution

// Tracer allows users to provide their own tracing solution
type Tracer interface {
	Trace(operation string, fn func() error) error
	TraceCtx(ctx context.Context, operation string, fn func(context.Context) error) error
}

// SetTracer sets the custom tracing solution to the registry
func SetTracer(tracer Tracer) {
	global.SetTracer(tracer)
}

func (r *Registry) SetTracer(tracer Tracer) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tracer = tracer
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
