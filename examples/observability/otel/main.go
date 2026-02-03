package main

import (
	"context"
	"fmt"

	"github.com/MintzyG/fail/v3"
	failotel "github.com/MintzyG/fail/v3/plugins/tracing/otel"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

var (
	// We define our errors as sentinels using fail.Form.
	// We set static=false in fail.ID so we can use builder methods later.

	// Domain Error: A business logic error (e.g. Validation)

	UserEmpty    = fail.ID(0, "USER", 0, false, "USERUserEmpty")
	ErrUserEmpty = fail.Form(UserEmpty, "username is required", false, nil)

	// System Error: A technical failure (e.g. Database)

	DBConnection    = fail.ID(0, "DB", 0, false, "DBConnectionError")
	ErrDBConnection = fail.Form(DBConnection, "database connection failed", true, nil)
)

func main() {
	// 1. Setup OpenTelemetry (InMemory Exporter for demonstration)
	exporter := tracetest.NewInMemoryExporter()
	tp := trace.NewTracerProvider(trace.WithSyncer(exporter))
	otel.SetTracerProvider(tp)

	// 2. Configure the fail library to use the OTel plugin

	tracerPlugin := failotel.New(
		failotel.WithTracerName("example-service"),
		// RecordSmart:
		// - System Errors -> Set Span Status to Error (for alerting)
		// - Domain Errors -> Add as Span Event (for debugging)
		failotel.WithMode(failotel.RecordSmart),
		failotel.WithStackTrace(),
		failotel.WithAttributePrefix("fail"),
	)
	fail.SetTracer(tracerPlugin)

	fmt.Println("=== OpenTelemetry Plugin Example ===")

	// 3. Run a simulated operation
	ctx := context.Background()
	tracer := otel.Tracer("main-logic")
	ctx, span := tracer.Start(ctx, "ProcessUserRegistration")
	defer span.End()

	fmt.Println("Running operation...")

	// Scenario A: A validation error (Domain Error)
	// We use fail.New() to spawn a fresh instance from the sentinel.
	err1 := fail.New(UserEmpty).
		AddMeta("field", "username").
		RecordCtx(ctx)

	fmt.Printf("Recorded Domain Error: %s\n", err1.Message)

	// Scenario B: A critical infrastructure error (System Error)
	// Here we might modify it further, e.g., adding an internal message.
	err2 := fail.New(DBConnection).
		Internal("connection timeout to 10.0.0.5:5432").
		RecordCtx(ctx)

	fmt.Printf("Recorded System Error: %s\n", err2.Message)

	span.End()

	// 4. Inspect the recorded telemetry
	spans := exporter.GetSpans()
	fmt.Printf("\n✅ Spans recorded: %d\n", len(spans))

	for _, s := range spans {
		fmt.Printf("--------------------------------------------------\n")
		fmt.Printf("Span: %s\n", s.Name)
		fmt.Printf("Status: %s\n", s.Status.Code)

		if s.Status.Code == codes.Error {
			fmt.Println("  -> Span is marked as ERROR (caused by a System Error)")
		}

		fmt.Println("Attributes:")
		for _, attr := range s.Attributes {
			if attr.Key == "fail.message" || attr.Key == "fail.is_system" || attr.Key == "fail.internal_message" {
				fmt.Printf("  - %s: %v\n", attr.Key, attr.Value.Emit())
			}
		}

		fmt.Println("Events:")
		for _, event := range s.Events {
			fmt.Printf("  - Event: %s\n", event.Name)
			for _, attr := range event.Attributes {
				fmt.Printf("    - %s: %v\n", attr.Key, attr.Value.Emit())
			}
		}
	}
}
