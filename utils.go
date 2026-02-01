package fail

import (
	"runtime"
	"strings"
)

// calledBeforeMain returns true if main.main has not been reached yet
func calledBeforeMain() bool {
	var pcs [32]uintptr
	n := runtime.Callers(2, pcs[:])
	frames := runtime.CallersFrames(pcs[:n])

	for {
		frame, more := frames.Next()
		if !more {
			break
		}
		// Check for main.main specifically, or any main function
		if strings.HasSuffix(frame.Function, "main.main") {
			return false // main.main found in stack, we're at runtime
		}
		// In tests, the entry point is testing.tRunner or TestMain
		if strings.Contains(frame.Function, "testing.tRunner") ||
			strings.Contains(frame.Function, "testing.runTests") {
			return false
		}
	}
	return true // main not in stack yet, we're in init/var phase
}

// getCallerInfo returns file and line of the caller at given skip depth
func getCallerInfo(skip int) (string, int) {
	_, file, line, ok := runtime.Caller(skip + 1)
	if !ok {
		return "unknown", 0
	}
	return file, line
}
