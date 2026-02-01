package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/MintzyG/fail"
)

var (
	AdminUsernameEmpty = fail.ID(0, "ADMIN", 0, true, "AdminUsernameEmpty")
	_                  = fail.Form(AdminUsernameEmpty, "username cannot be empty", false, nil)
)

type SlogLogger struct {
	Logger          *slog.Logger
	LogDomainErrors bool
	LogSystemErrors bool
	IncludeMetadata bool
	IncludeInternal bool
}

func DefaultSlogLogger(logger *slog.Logger) *SlogLogger {
	return &SlogLogger{
		Logger:          logger,
		LogDomainErrors: true,
		LogSystemErrors: true,
		IncludeMetadata: true,
		IncludeInternal: true,
	}
}

func (s *SlogLogger) Log(e *fail.Error) {
	s.log(context.Background(), e)
}

func (s *SlogLogger) LogCtx(ctx context.Context, e *fail.Error) {
	s.log(ctx, e)
}

func (s *SlogLogger) log(ctx context.Context, e *fail.Error) {
	attrs := []any{
		"error_id", e.ID.String(),
		"is_system", e.IsSystem,
	}

	if e.IsSystem && s.LogSystemErrors {
		s.Logger.ErrorContext(ctx, e.Message, attrs...)
		return
	}

	if !e.IsSystem && s.LogDomainErrors {
		s.Logger.InfoContext(ctx, e.Message, attrs...)
	}
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	fail.SetLogger(DefaultSlogLogger(logger))

	fmt.Println("=== Slog Logger Example ===")

	// Use fail.New(ID) to avoid mutating the sentinel with AddMeta
	_ = fail.New(AdminUsernameEmpty).
		AddMeta("user_id", 123).
		Log()
}
