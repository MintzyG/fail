package translators

import (
	"fail"
	"net/http"
)

// HTTPResponse represents an HTTP error response
type HTTPResponse struct {
	StatusCode  int                    `json:"-"`
	ErrorID     string                 `json:"error_id,omitempty"`
	Message     string                 `json:"message"`
	Traces      []string               `json:"traces,omitempty"`
	Debug       []string               `json:"debug,omitempty"`
	Validations []fail.ValidationError `json:"validations,omitempty"`
	Meta        map[string]any         `json:"meta,omitempty"`
}

// HTTPTranslator converts errors to HTTP responses
type HTTPTranslator struct {
	IncludeErrorID  bool
	IncludeTraces   bool
	IncludeDebug    bool
	IncludeMeta     bool
	CustomStatusMap map[fail.ErrorID]int // Override status codes for specific errors
}

// HTTPResponseTranslator creates a production-ready HTTP translator
func HTTPResponseTranslator() *HTTPTranslator {
	return &HTTPTranslator{
		IncludeErrorID:  true,
		IncludeTraces:   false,
		IncludeDebug:    false,
		IncludeMeta:     false,
		CustomStatusMap: make(map[fail.ErrorID]int),
	}
}

// DevelopmentHTTPTranslator creates a translator with all debug info
func DevelopmentHTTPTranslator() *HTTPTranslator {
	return &HTTPTranslator{
		IncludeErrorID:  true,
		IncludeTraces:   true,
		IncludeDebug:    true,
		IncludeMeta:     true,
		CustomStatusMap: make(map[fail.ErrorID]int),
	}
}

func (h *HTTPTranslator) Name() string {
	return "http"
}

func (h *HTTPTranslator) Translate(err *fail.Error) any {
	resp := HTTPResponse{
		StatusCode: h.getStatusCode(err),
		Message:    err.Message,
	}

	if h.IncludeErrorID {
		resp.ErrorID = err.ID.String()
	}

	if err.Meta != nil {
		// Extract common metadata
		if h.IncludeTraces {
			if traces, ok := err.Meta["traces"].([]string); ok {
				resp.Traces = traces
			}
		}

		if h.IncludeDebug {
			if debug, ok := err.Meta["debug"].([]string); ok {
				resp.Debug = debug
			}
		}

		// Always include validations if present
		if validations, ok := err.Meta["validations"].([]fail.ValidationError); ok {
			resp.Validations = validations
		}

		// Include remaining meta if enabled
		if h.IncludeMeta {
			resp.Meta = make(map[string]any)
			for k, v := range err.Meta {
				// Skip already-extracted fields
				if k == "traces" || k == "debug" || k == "validations" {
					continue
				}
				resp.Meta[k] = v
			}
		}
	}

	return resp
}

// getStatusCode determines the HTTP status code for an error
func (h *HTTPTranslator) getStatusCode(err *fail.Error) int {
	// Check custom mapping first
	if status, exists := h.CustomStatusMap[err.ID]; exists {
		return status
	}

	// Infer from error domain (convention-based)
	domain := err.ID.Domain()

	// Common domains -> status codes
	switch domain {
	case "AUTH":
		return http.StatusUnauthorized
	case "PERM", "FORB":
		return http.StatusForbidden
	case "NFND", "NOTF":
		return http.StatusNotFound
	case "CONF":
		return http.StatusConflict
	case "RATE":
		return http.StatusTooManyRequests
	case "VALD", "INVL", "BADR", "USER":
		return http.StatusBadRequest
	}

	// System errors -> 500
	if err.IsSystem {
		return http.StatusInternalServerError
	}

	// Domain errors default to 400
	return http.StatusBadRequest
}

// WithCustomStatus adds a custom status code mapping
func (h *HTTPTranslator) WithCustomStatus(id fail.ErrorID, statusCode int) *HTTPTranslator {
	h.CustomStatusMap[id] = statusCode
	return h
}

// HTTPTranslatorOption is a functional option for HTTPTranslator
type HTTPTranslatorOption func(*HTTPTranslator)

// WithErrorID enables/disables error ID in response
func WithErrorID(include bool) HTTPTranslatorOption {
	return func(h *HTTPTranslator) {
		h.IncludeErrorID = include
	}
}

// WithTraces enables/disables traces in response
func WithTraces(include bool) HTTPTranslatorOption {
	return func(h *HTTPTranslator) {
		h.IncludeTraces = include
	}
}

// WithDebug enables/disables debug info in response
func WithDebug(include bool) HTTPTranslatorOption {
	return func(h *HTTPTranslator) {
		h.IncludeDebug = include
	}
}

// WithMeta enables/disables metadata in response
func WithMeta(include bool) HTTPTranslatorOption {
	return func(h *HTTPTranslator) {
		h.IncludeMeta = include
	}
}

// NewHTTPTranslator creates a customized HTTP translator
func NewHTTPTranslator(opts ...HTTPTranslatorOption) *HTTPTranslator {
	t := HTTPResponseTranslator()
	for _, opt := range opts {
		opt(t)
	}
	return t
}
