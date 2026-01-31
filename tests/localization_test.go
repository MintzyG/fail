package fail_test

import (
	"strings"
	"testing"

	"github.com/MintzyG/fail"
)

var (
	LocTestID      = fail.ID(0, "LOC", 0, true, "LocalizationTestError")
	LocTemplateID  = fail.ID(0, "LOC", 1, false, "LocTemplateError")
	LocArgID       = fail.ID(0, "LOC", 2, false, "LocArgError")
	LocBadFormatID = fail.ID(0, "LOC", 3, false, "LocBadFormatError")
)

func TestLocalization_RegisterAndLocalize(t *testing.T) {
	// Create a fresh registry to avoid polluting global state and to test initialization
	reg := fail.NewRegistry()

	reg.Form(LocTestID, "default message", false, nil)

	t.Run("RegisterTranslations should not panic", func(t *testing.T) {
		msgs := map[fail.ErrorID]string{
			LocTestID: "mensaje traducido",
		}
		reg.RegisterTranslations("es-ES", msgs)
	})

	t.Run("Localize should return translated message", func(t *testing.T) {
		msgs := map[fail.ErrorID]string{
			LocTestID: "mensaje traducido",
		}
		reg.RegisterTranslations("es-ES", msgs)

		err := reg.New(LocTestID)
		err.Locale = "es-ES"

		if err.Message != "default message" {
			t.Errorf("Expected 'default message', got '%s'", err.Message)
		}

		err.Localize()
		if err.Message != "mensaje traducido" {
			t.Errorf("Expected 'mensaje traducido', got '%s'", err.Message)
		}
	})

	t.Run("Localize should fallback to default locale if set", func(t *testing.T) {
		msgs := map[fail.ErrorID]string{
			LocTestID: "message traduit",
		}
		reg.RegisterTranslations("fr-FR", msgs)
		reg.SetDefaultLocale("fr-FR")

		err := reg.New(LocTestID)
		// No locale set on error

		err.Localize()
		if err.Message != "message traduit" {
			t.Errorf("Expected 'message traduit', got '%s'", err.Message)
		}
	})

	t.Run("Localize should fallback to original message if translation missing", func(t *testing.T) {
		err := reg.New(LocTestID)
		err.Locale = "de-DE" // No translations for German

		err.Localize()
		if err.Message != "default message" {
			t.Errorf("Expected 'default message', got '%s'", err.Message)
		}
	})
}

func TestRendering(t *testing.T) {
	reg := fail.NewRegistry()
	reg.Form(LocTemplateID, "Hello %s", false, nil)
	reg.Form(LocArgID, "Value: %d", false, nil, 42) // Default arg 42

	t.Run("Render should format message with args", func(t *testing.T) {
		// fail.Error has Args field.
		err := reg.New(LocTemplateID)
		err.Args = []any{"World"}

		err.Render()
		if err.Message != "Hello World" {
			t.Errorf("Expected 'Hello World', got '%s'", err.Message)
		}
	})

	t.Run("Render should use localized template", func(t *testing.T) {
		reg.RegisterTranslations("es-ES", map[fail.ErrorID]string{
			LocTemplateID: "Hola %s",
		})

		err := reg.New(LocTemplateID)
		err.Locale = "es-ES"
		err.Args = []any{"Mundo"}

		// Render uses e.Message. If it's already set (by New), it won't re-localize automatically unless we call Localize.
		err.Localize().Render()
		if err.Message != "Hola Mundo" {
			t.Errorf("Expected 'Hola Mundo', got '%s'", err.Message)
		}
	})

	t.Run("Render should use default args if none provided", func(t *testing.T) {
		err := reg.New(LocArgID)
		// Default arg 42 should be used

		err.Render()
		if err.Message != "Value: 42" {
			t.Errorf("Expected 'Value: 42', got '%s'", err.Message)
		}
	})

	t.Run("Render should safe guard against panic", func(t *testing.T) {
		reg.Form(LocBadFormatID, "Bad %v", false, nil)

		err := reg.New(LocBadFormatID)
		err.Args = []any{panicStringer{}}

		err.Render()
		// fmt.Sprintf handles panics in String() by printing a special message
		// e.g. "Bad %!v(PANIC=String method: oops)"
		if !strings.Contains(err.Message, "PANIC") {
			t.Errorf("Expected panic indication in message, got '%s'", err.Message)
		}

		// Note: safeSprintf's recover block is not triggered because fmt.Sprintf recovers internally.
		// We are just verifying that Render doesn't crash the whole app.
	})

	t.Run("Render should add warning if placeholders exist but no args", func(t *testing.T) {
		reg.Form(LocTemplateID, "Need args %s", false, nil)

		err := reg.New(LocTemplateID)
		err.Args = nil // explicit nil

		err.Render()
		if err.Message != "Need args %s" {
			t.Errorf("Expected original template when no args, got '%s'", err.Message)
		}

		if err.Meta == nil || err.Meta["fail.render_warning"] == nil {
			t.Error("Expected fail.render_warning meta to be set")
		}
	})

	t.Run("GetRendered should return rendered string without modifying error", func(t *testing.T) {
		reg.Form(LocTemplateID, "Hello %s", false, nil)

		err := reg.New(LocTemplateID)
		err.Args = []any{"Universe"}

		rendered := err.GetRendered()
		if rendered != "Hello Universe" {
			t.Errorf("Expected 'Hello Universe', got '%s'", rendered)
		}

		if err.Message != "Hello %s" {
			t.Errorf("Expected error message to remain template 'Hello %%s', got '%s'", err.Message)
		}
	})
}

var LocBuilderID = fail.ID(0, "LOCBLD", 0, true, "LocBldTestError")

// TestAddLocalization_Chaining verifies that AddLocalization and AddLocalizations
// can be chained on Form and correctly register translations.
func TestAddLocalization_Chaining(t *testing.T) {
	// Register error with translations using chaining
	// This simulates "at var level" usage
	_ = fail.Form(LocBuilderID, "English default", false, nil).
		AddLocalization("pt-BR", "Erro em português").
		AddLocalizations(map[string]string{
			"es-ES": "Error en español",
			"zh-CN": "中文错误",
		})

	tests := []struct {
		name     string
		locale   string
		expected string
	}{
		{
			name:     "Default Locale (English)",
			locale:   "", // or "en-US" depending on default
			expected: "English default",
		},
		{
			name:     "Portuguese (pt-BR)",
			locale:   "pt-BR",
			expected: "Erro em português",
		},
		{
			name:     "Spanish (es-ES)",
			locale:   "es-ES",
			expected: "Error en español",
		},
		{
			name:     "Chinese (zh-CN)",
			locale:   "zh-CN",
			expected: "中文错误",
		},
		{
			name:     "Unknown Locale fallback",
			locale:   "fr-FR",
			expected: "English default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a new error instance from the ID
			err := fail.New(LocBuilderID)

			// Set the locale we want to test
			if tt.locale != "" {
				err.Locale = tt.locale
			}

			// Localize explicitly
			err.Localize()

			if err.Message != tt.expected {
				t.Errorf("Locale %s: expected '%s', got '%s'", tt.locale, tt.expected, err.Message)
			}
		})
	}
}

type panicStringer struct{}

func (p panicStringer) String() string {
	panic("oops")
}
