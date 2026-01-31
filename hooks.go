package fail

import (
	"fmt"
	"runtime"
	"sync"
)

type HookType int

const (
	HookCreate HookType = iota
	HookLog
	HookTrace
	HookMap
	HookWrap
	HookFrom
	HookForm
	HookTranslate
	HookMatch
)

// Hooks manages lifecycle callbacks for errors
// Access via Registry.Hooks
type Hooks struct {
	mu          sync.RWMutex
	onCreate    []func(*Error, map[string]any)
	onLog       []func(*Error, map[string]any)
	onTrace     []func(*Error, map[string]any)
	OnMap       []func(*Error, map[string]any)
	onWrap      []func(*Error, error)
	onFrom      []func(error, *Error)
	onForm      []func(ErrorID, *Error)
	onTranslate []func(*Error, map[string]any)
	onMatch     []func(*Error, map[string]any)
}

// Frame represents a single stack frame for error traces
type Frame struct {
	Function string `json:"function"`
	File     string `json:"file"`
	Line     int    `json:"line"`
	Package  string `json:"package,omitempty"`
}

// CaptureStack creates a []Frame from runtime
func CaptureStack(skip int) []Frame {
	var frames []Frame
	pc := make([]uintptr, 32) // Capture last 32 frames
	n := runtime.Callers(skip+1, pc)

	for i := 0; i < n; i++ {
		fn := runtime.FuncForPC(pc[i])
		if fn == nil {
			continue
		}
		file, line := fn.FileLine(pc[i])
		frames = append(frames, Frame{
			Function: fn.Name(),
			File:     file,
			Line:     line,
		})
	}
	return frames
}

// On is a global convenience for setting hooks
func On(t HookType, fn any) {
	global.hooks.On(t, fn)
}

// On is a convenience for setting hooks on custom registries
func (r *Registry) On(t HookType, fn any) {
	r.hooks.On(t, fn)
}

// On registers a hook with compile-time friendly type validation (no reflect)
// Panics immediately if function signature doesn't match HookType
func (h *Hooks) On(t HookType, fn any) {
	switch t {
	case HookCreate:
		f, ok := fn.(func(*Error, map[string]any))
		if !ok {
			panic(fmt.Sprintf("HookCreate requires func(*Error, map[string]any), got %T", fn))
		}
		h.mu.Lock()
		h.onCreate = append(h.onCreate, f)
		h.mu.Unlock()

	case HookLog:
		f, ok := fn.(func(*Error, map[string]any))
		if !ok {
			panic(fmt.Sprintf("HookLog requires func(*Error, map[string]any), got %T", fn))
		}
		h.mu.Lock()
		h.onLog = append(h.onLog, f)
		h.mu.Unlock()

	case HookTrace:
		f, ok := fn.(func(*Error, map[string]any))
		if !ok {
			panic(fmt.Sprintf("HookTrace requires func(*Error, map[string]any), got %T", fn))
		}
		h.mu.Lock()
		h.onTrace = append(h.onTrace, f)
		h.mu.Unlock()

	case HookMap:
		f, ok := fn.(func(*Error, map[string]any))
		if !ok {
			panic(fmt.Sprintf("HookMap requires func(*Error, map[string]any), got %T", fn))
		}
		h.mu.Lock()
		h.OnMap = append(h.OnMap, f)
		h.mu.Unlock()

	case HookWrap:
		f, ok := fn.(func(*Error, error))
		if !ok {
			panic(fmt.Sprintf("HookWrap requires func(*Error, error), got %T", fn))
		}
		h.mu.Lock()
		h.onWrap = append(h.onWrap, f)
		h.mu.Unlock()

	case HookFrom:
		f, ok := fn.(func(error, *Error))
		if !ok {
			panic(fmt.Sprintf("HookFrom requires func(error, *Error), got %T", fn))
		}
		h.mu.Lock()
		h.onFrom = append(h.onFrom, f)
		h.mu.Unlock()

	case HookForm:
		f, ok := fn.(func(ErrorID, *Error))
		if !ok {
			panic(fmt.Sprintf("HookForm requires func(ErrorID, *Error), got %T", fn))
		}
		h.mu.Lock()
		h.onForm = append(h.onForm, f)
		h.mu.Unlock()

	case HookTranslate:
		f, ok := fn.(func(*Error, map[string]any))
		if !ok {
			panic(fmt.Sprintf("HookTranslate requires func(*Error, map[string]any), got %T", fn))
		}
		h.mu.Lock()
		h.onTranslate = append(h.onTranslate, f)
		h.mu.Unlock()

	case HookMatch:
		f, ok := fn.(func(*Error, map[string]any))
		if !ok {
			panic(fmt.Sprintf("HookMatch requires func(*Error, map[string]any), got %T", fn))
		}
		h.mu.Lock()
		h.onMatch = append(h.onMatch, f)
		h.mu.Unlock()

	default:
		panic(fmt.Sprintf("unknown hook type: %d", t))
	}
}

// Execution methods (called internally)
func (h *Hooks) runCreate(err *Error, data map[string]any) {
	h.mu.RLock()
	hooks := h.onCreate
	h.mu.RUnlock()
	for _, fn := range hooks {
		fn(err, data)
	}
}

func (h *Hooks) runLog(err *Error, data map[string]any) {
	h.mu.RLock()
	hooks := h.onLog
	h.mu.RUnlock()
	for _, fn := range hooks {
		fn(err, data)
	}
}

func (h *Hooks) runTrace(err *Error, data map[string]any) {
	h.mu.RLock()
	hooks := h.onTrace
	h.mu.RUnlock()
	for _, fn := range hooks {
		fn(err, data)
	}
}

func (h *Hooks) runMap(err *Error, data map[string]any) {
	h.mu.RLock()
	hooks := h.OnMap
	h.mu.RUnlock()
	for _, fn := range hooks {
		fn(err, data)
	}
}

func (h *Hooks) runWrap(wrapper *Error, wrapped error) {
	h.mu.RLock()
	hooks := h.onWrap
	h.mu.RUnlock()
	for _, fn := range hooks {
		fn(wrapper, wrapped)
	}
}

func (h *Hooks) runFrom(original error, converted *Error) {
	h.mu.RLock()
	hooks := h.onFrom
	h.mu.RUnlock()
	for _, fn := range hooks {
		fn(original, converted)
	}
}

func (h *Hooks) runForm(id ErrorID, template *Error) {
	h.mu.RLock()
	hooks := h.onForm
	h.mu.RUnlock()
	for _, fn := range hooks {
		fn(id, template)
	}
}

func (h *Hooks) runTranslate(err *Error, data map[string]any) {
	h.mu.RLock()
	hooks := h.onTranslate
	h.mu.RUnlock()
	for _, fn := range hooks {
		fn(err, data)
	}
}

func (h *Hooks) runMatch(err *Error, data map[string]any) {
	h.mu.RLock()
	hooks := h.onMatch
	h.mu.RUnlock()
	for _, fn := range hooks {
		fn(err, data)
	}
}

// IDE-friendly convenience wrappers

func OnCreate(fn func(*Error, map[string]any))    { On(HookCreate, fn) }
func OnLog(fn func(*Error, map[string]any))       { On(HookLog, fn) }
func OnTrace(fn func(*Error, map[string]any))     { On(HookTrace, fn) }
func OnMap(fn func(*Error, map[string]any))       { On(HookMap, fn) }
func OnWrap(fn func(*Error, error))               { On(HookWrap, fn) }
func OnFrom(fn func(error, *Error))               { On(HookFrom, fn) }
func OnForm(fn func(ErrorID, *Error))             { On(HookForm, fn) }
func OnTranslate(fn func(*Error, map[string]any)) { On(HookTranslate, fn) }
func OnMatch(fn func(*Error, map[string]any))     { On(HookMatch, fn) }
