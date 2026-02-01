package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/MintzyG/fail"
)

var (
	UserValidationFailed = fail.ID(0, "USER", 9, false, "UserValidationFailed")
	_                    = fail.Form(UserValidationFailed, "validation failed", false, nil)

	CannotTranslateToHTTP = fail.ID(5, "TR", 0, false, "TRCannotTranslateToHTTP")
	_                     = fail.Form(CannotTranslateToHTTP, "cannot translate error to http", true, nil)
)

type HTTPResponse struct {
	StatusCode int            `json:"-"`
	ErrorID    string         `json:"error_id,omitempty"`
	Message    string         `json:"message"`
	Meta       map[string]any `json:"meta,omitempty"`
}

type HTTPTranslator struct {
	IncludeErrorID bool
}

func HTTPResponseTranslator() *HTTPTranslator {
	return &HTTPTranslator{IncludeErrorID: true}
}

func (h *HTTPTranslator) Name() string { return "http" }
func (h *HTTPTranslator) Supports(err *fail.Error) error {
	if err != nil && err.IsTrusted() {
		return nil
	}
	return errors.New("unsupported")
}

func (h *HTTPTranslator) Translate(err *fail.Error) (any, error) {
	if err := h.Supports(err); err != nil {
		return nil, fail.New(CannotTranslateToHTTP).With(err)
	}

	resp := HTTPResponse{
		StatusCode: h.getStatusCode(err),
		Message:    err.Message,
	}

	if h.IncludeErrorID {
		resp.ErrorID = err.ID.String()
	}

	return resp, nil
}

func (h *HTTPTranslator) getStatusCode(err *fail.Error) int {
	domain := err.ID.Domain()
	switch domain {
	case "AUTH":
		return http.StatusUnauthorized
	case "USER":
		return http.StatusBadRequest
	}
	if err.IsSystem {
		return http.StatusInternalServerError
	}
	return http.StatusBadRequest
}

func main() {
	fmt.Println("=== HTTP Translator Example ===")

	if err := fail.RegisterTranslator(HTTPResponseTranslator()); err != nil {
		log.Fatal(err)
	}

	// Use fail.New(ID) to avoid mutating sentinel
	err := fail.New(UserValidationFailed).
		Msg("validation failed").
		Validation("email", "invalid format")

	resp, _ := fail.Translate(err, "http")

	b, _ := json.MarshalIndent(resp, "", "  ")
	fmt.Printf("HTTP Response:\n%s\n", string(b))

	httpResp := resp.(HTTPResponse)
	fmt.Printf("Status Code: %d\n", httpResp.StatusCode)
}
