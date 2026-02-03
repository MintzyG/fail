package fail

import (
	"fmt"
	"strings"
)

type Localizer interface {
	// Localize returns the localized template for the given error ID and locale
	// Returns empty string if no translation exists
	Localize(id ErrorID, locale string) string

	// RegisterLocalization adds a single translation (for AddLocalization)
	RegisterLocalization(id ErrorID, locale string, template string)

	// RegisterLocalizations adds multiple translations for a locale (for bulk operations)
	RegisterLocalizations(locale string, translations map[ErrorID]string)
}

// RegisterLocalizations adds translations for a locale on the global registry
func RegisterLocalizations(locale string, msgs map[ErrorID]string) {
	global.RegisterLocalizations(locale, msgs)
}

// RegisterLocalizations adds translations for a locale in a specific registry
func (r *Registry) RegisterLocalizations(locale string, msgs map[ErrorID]string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.localization != nil {
		r.localization.RegisterLocalizations(locale, msgs)
	} else {
		for id, msg := range msgs {
			if r.pendingLocalizations[id] == nil {
				r.pendingLocalizations[id] = make(map[string]string)
			}
			r.pendingLocalizations[id][locale] = msg
		}
	}
}

// SetDefaultLocale sets the fallback locale for the global registry
func SetDefaultLocale(locale string) {
	global.SetDefaultLocale(locale)
}

// SetDefaultLocale sets the fallback locale for the specific registry
func (r *Registry) SetDefaultLocale(locale string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.defaultLocale = locale
}

// Localize resolves the translated message template for this error's locale
// Stores result in e.Message, returns *Error for chaining
func (e *Error) Localize() *Error {
	locale := e.resolveLocale()
	e.Message = resolveTemplate(e, locale)
	return e
}

// Render formats the error's message template with its arguments
// Stores result in e.Message, returns *Error for chaining
func (e *Error) Render() *Error {
	// Ensure we have localized template first
	if e.Message == "" {
		_ = e.Localize()
	}

	template := e.Message

	// No formatting needed
	if !strings.Contains(template, "%") {
		return e
	}

	args := e.effectiveArgs()

	if len(args) == 0 {
		_ = e.AddMeta("fail.render_warning", "template has placeholders but no args provided")
		return e
	}

	e.Message = safeSprintf(e, template, args...)
	return e
}

// GetLocalized returns the localized message template as string (read-only, no modification)
func (e *Error) GetLocalized() string {
	locale := e.resolveLocale()
	return resolveTemplate(e, locale)
}

// GetRendered returns the fully rendered message as string (read-only, no modification)
func (e *Error) GetRendered() string {
	template := e.GetLocalized()

	if !strings.Contains(template, "%") {
		return template
	}

	args := e.effectiveArgs()
	if len(args) == 0 {
		_ = e.AddMeta("fail.render_warning", "template has placeholders but no args provided")
		return template
	}

	return safeSprintf(e, template, args...)
}

func (e *Error) resolveLocale() string {
	if e.Locale != "" {
		return e.Locale
	}

	reg := e.registry
	if reg == nil {
		reg = global
	}

	reg.mu.RLock()
	def := reg.defaultLocale
	reg.mu.RUnlock()

	if def != "" {
		return def
	}
	return "en-US"
}

func resolveTemplate(e *Error, locale string) string {
	reg := e.registry
	if reg == nil {
		reg = global
	}

	reg.mu.RLock()
	loc := reg.localization
	reg.mu.RUnlock()

	if loc == nil {
		return e.Message
	}

	if msg := loc.Localize(e.ID, locale); msg != "" {
		return msg
	}

	// Fallback to error's default message
	return e.Message
}

func (e *Error) effectiveArgs() []any {
	if len(e.Args) > 0 {
		return e.Args
	}

	// Try to get default args from definition
	reg := e.registry
	if reg == nil {
		reg = global
	}

	reg.mu.RLock()
	def, exists := reg.definitions[e.ID]
	reg.mu.RUnlock()

	if exists && len(def.DefaultArgs) > 0 {
		return def.DefaultArgs
	}

	return nil
}

func safeSprintf(e *Error, format string, args ...any) (result string) {
	expectedArgs := strings.Count(format, "%") - strings.Count(format, "%%")*2
	if len(args) != expectedArgs {
		_ = e.AddMeta("fail.render_error",
			fmt.Sprintf("arg mismatch: expected %d, got %d", expectedArgs, len(args)))
		return format
	}

	defer func() {
		if r := recover(); r != nil {
			_ = e.AddMeta("fail.render_error", fmt.Sprintf("panic during sprintf: %v", r))
			result = format
		}
	}()

	return fmt.Sprintf(format, args...)
}

// AddLocalization adds a translation for this error's ID to its registry
// If localization already exists for this locale+ID, does nothing (idempotent)
// Returns the original error unmodified for chaining
func (e *Error) AddLocalization(locale string, msg string) *Error {
	reg := e.registry
	if reg == nil {
		reg = global
	}

	reg.mu.RLock()
	localizer := reg.localization
	reg.mu.RUnlock()

	if localizer != nil {
		// Skip if already exists (first registration wins)
		if existing := localizer.Localize(e.ID, locale); existing != "" {
			return e
		}
		localizer.RegisterLocalization(e.ID, locale, msg)
	} else {
		if reg.pendingLocalizations[e.ID] == nil {
			reg.pendingLocalizations[e.ID] = make(map[string]string)
		}
		reg.pendingLocalizations[e.ID][locale] = msg
	}

	return e
}

// AddLocalizations adds multiple translations at once
func (e *Error) AddLocalizations(msgs map[string]string) *Error {
	for locale, msg := range msgs {
		_ = e.AddLocalization(locale, msg)
	}
	return e
}
