package observability

import (
	"context"
	"log/slog"

	"fail"
)

// SlogLogger integrates with structured logging
type SlogLogger struct {
	Logger          *slog.Logger
	LogDomainErrors bool // If true, log domain errors at Info level
	LogSystemErrors bool // If true, log system errors at Error level
	IncludeMetadata bool // If true, include all metadata in logs
	IncludeInternal bool // If true, include internal message in logs
}

// DefaultSlogLogger returns a sensible default configuration
func DefaultSlogLogger(logger *slog.Logger) *SlogLogger {
	return &SlogLogger{
		Logger:          logger,
		LogDomainErrors: true,
		LogSystemErrors: true,
		IncludeMetadata: true,
		IncludeInternal: true,
	}
}

// Log implements Logger without context
func (s *SlogLogger) Log(e *fail.Error) {
	s.log(nil, e)
}

// LogCtx implements Logger with context support
func (s *SlogLogger) LogCtx(ctx context.Context, e *fail.Error) {
	s.log(ctx, e)
}

func (s *SlogLogger) log(ctx context.Context, e *fail.Error) {
	if s.Logger == nil || e == nil {
		return
	}

	attrs := []any{
		"error_id", e.ID.String(),
		"is_system", e.IsSystem,
	}

	if s.IncludeInternal && e.InternalMessage != "" {
		attrs = append(attrs, "internal_message", e.InternalMessage)
	}

	if e.Cause != nil {
		attrs = append(attrs, "cause", e.Cause.Error())
	}

	if s.IncludeMetadata && e.Meta != nil {
		for k, v := range e.Meta {
			attrs = append(attrs, "meta."+k, v)
		}
	}

	logger := s.Logger
	if ctx != nil {
		logger = logger.With(slog.Any("context", ctx)) // optional: attach ctx
	}

	// Choose log level based on error type
	if e.IsSystem && s.LogSystemErrors {
		logger.Error(e.Message, attrs...)
		return
	}

	if !e.IsSystem && s.LogDomainErrors {
		logger.Info(e.Message, attrs...)
	}
}

var AdminUsernameEmpty = fail.ID("AdminUsernameEmpty", "ADMIN", true)
var ErrAdminUsernameEmpty = fail.Form(AdminUsernameEmpty, "username cannot be empty", false, nil)

var AdminIDNotFound = fail.ID("AdminIDNotFound", "ADMIN", true)
var ErrAdminNotFound = fail.Form(AdminIDNotFound, "admin not found", false, nil)

func CreateAdmin(logger fail.Logger, name string) error {
	if name == "" {
		logger.Log(ErrAdminUsernameEmpty)
		return ErrAdminUsernameEmpty
	}

	// pretend DB insert here
	return nil
}

func CreateAdminCtx(ctx context.Context, logger fail.Logger, name string) error {
	if name == "" {
		logger.LogCtx(ctx, ErrAdminUsernameEmpty)
		return ErrAdminUsernameEmpty
	}

	return nil
}

type AdminService struct {
	logger fail.Logger
}

func NewAdminService(logger fail.Logger) *AdminService {
	return &AdminService{logger: logger}
}

func (s *AdminService) CreateAdmin(name string) error {
	if name == "" {
		s.logger.Log(ErrAdminNotFound)
		return ErrAdminNotFound
	}

	// pretend DB insert
	return nil
}

func (s *AdminService) DeleteUser(ctx context.Context, id string) error {
	if id == "" {
		s.logger.LogCtx(ctx, ErrAdminNotFound)
		return ErrAdminNotFound
	}

	// pretend DB delete
	return nil
}
