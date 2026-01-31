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

// HTTPTranslator converts registry errors to HTTP responses
type HTTPTranslator struct {
	IncludeErrorID  bool
	IncludeTraces   bool
	IncludeDebug    bool
	IncludeMeta     bool
	CustomStatusMap map[fail.ErrorID]int // Override status codes for specific errors
}

// Prebuilt production translator
func HTTPResponseTranslator() *HTTPTranslator {
	return &HTTPTranslator{
		IncludeErrorID:  true,
		IncludeTraces:   false,
		IncludeDebug:    false,
		IncludeMeta:     false,
		CustomStatusMap: make(map[fail.ErrorID]int),
	}
}

// Prebuilt development translator with full debug info
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

func (h *HTTPTranslator) Supports(err *fail.Error) bool {
	// Only support trusted registry errors
	return err != nil && err.IsTrusted()
}

var CannotTranslateToHTTP = fail.ID("TRCannotTranslateToHTTP", "TR", false, 5)
var ErrCannotTranslateToHTTP = fail.Form(CannotTranslateToHTTP, "cannot translate error to http", true, nil)

// Translate converts a fail.Error to an HTTPResponse
func (h *HTTPTranslator) Translate(err *fail.Error) (any, error) {
	if !h.Supports(err) {
		return nil, fail.New(CannotTranslateToHTTP).With(err)
	}

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
		if validations, ok := err.Meta["validations"].([]fail.ValidationError); ok {
			resp.Validations = validations
		}
		// Include remaining meta if enabled
		if h.IncludeMeta {
			resp.Meta = make(map[string]any)
			for k, v := range err.Meta {
				if k == "traces" || k == "debug" || k == "validations" {
					continue
				}
				resp.Meta[k] = v
			}
		}
	}

	return resp, nil
}

// getStatusCode returns the HTTP status code
func (h *HTTPTranslator) getStatusCode(err *fail.Error) int {
	if status, exists := h.CustomStatusMap[err.ID]; exists {
		return status
	}

	domain := err.ID.Domain()

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

	if err.IsSystem {
		return http.StatusInternalServerError
	}

	return http.StatusBadRequest
}

// WithCustomStatus adds a custom HTTP status for a specific ErrorID
func (h *HTTPTranslator) WithCustomStatus(id fail.ErrorID, status int) *HTTPTranslator {
	h.CustomStatusMap[id] = status
	return h
}

// Functional options
type HTTPTranslatorOption func(*HTTPTranslator)

func WithErrorID(include bool) HTTPTranslatorOption {
	return func(h *HTTPTranslator) { h.IncludeErrorID = include }
}
func WithTraces(include bool) HTTPTranslatorOption {
	return func(h *HTTPTranslator) { h.IncludeTraces = include }
}
func WithDebug(include bool) HTTPTranslatorOption {
	return func(h *HTTPTranslator) { h.IncludeDebug = include }
}
func WithMeta(include bool) HTTPTranslatorOption {
	return func(h *HTTPTranslator) { h.IncludeMeta = include }
}

// NewHTTPTranslator creates a customized translator
func NewHTTPTranslator(opts ...HTTPTranslatorOption) *HTTPTranslator {
	t := HTTPResponseTranslator()
	for _, opt := range opts {
		opt(t)
	}
	return t
}
