package fail

// Translator converts a trusted registry Error into another format
// (HTTP response, gRPC status, CLI output, etc.)
type Translator interface {
	Name() string

	// Supports reports whether this translator is allowed to translate this error.
	// This is for capability, boundary, or policy checks — not mapping.
	Supports(*Error) error

	// Translate converts the error into an external representation.
	// It may still fail even if Supports returned true (e.g. missing metadata).
	Translate(*Error) (any, error)
}

func MustRegisterTranslator(t Translator) {
	if err := RegisterTranslator(t); err != nil {
		panic(err)
	}
}

// RegisterTranslator adds a translator for converting errors to other formats
func RegisterTranslator(t Translator) error {
	return global.RegisterTranslator(t)
}

func (r *Registry) RegisterTranslator(t Translator) error {
	if t == nil {
		return New(TranslatorNil)
	}

	name := t.Name()
	if name == "" {
		return New(TranslatorNameEmpty)
	}

	r.mu.Lock()
	if _, exists := r.translators[name]; exists {
		r.mu.Unlock()
		return New(TranslatorAlreadyRegistered).AddMeta("name", name)
	}

	r.translators[name] = t
	r.mu.Unlock()
	return nil
}

// To converts a fail.Error to an external format using the named translator
// Only registered errors can be translated (safety guarantee)
func To(err *Error, translatorName string) (any, error) {
	return global.To(err, translatorName)
}

// To converts a fail.Error to an external format using the named translator
// Only registered errors can be translated (safety guarantee)
func (r *Registry) To(err *Error, translatorName string) (zero any, retErr error) {
	if err == nil {
		return nil, nil
	}

	// Safety check: only translate registered errors
	if !err.IsRegistered() {
		return nil, New(TranslateUnregisteredError).AddMeta("translator", translatorName).With(err)
	}

	r.mu.RLock()
	translator, exists := r.translators[translatorName]
	r.mu.RUnlock()

	if !exists {
		return nil, New(TranslatorNotFound).WithArgs(translatorName).Render()
	}

	// Translator's Supports() check
	if spErr := translator.Supports(err); spErr != nil {
		return nil, New(TranslateUnsupportedError).WithArgs(translatorName).With(spErr).Render()
	}

	r.hooks.runTranslate(err, map[string]any{
		"translator": translatorName,
	})

	defer func() {
		if rec := recover(); rec != nil {
			retErr = New(TranslatePanicked).
				With(err).
				WithArgs(translatorName).
				AddMeta("panic", rec).
				Render()
		}
	}()

	return translator.Translate(err)
}

// ToAs is the generic version for global registry
func ToAs[T any](err *Error, translatorName string) (T, error) {
	var zero T

	out, trErr := global.To(err, translatorName)
	if trErr != nil {
		return zero, trErr
	}

	if out == nil {
		return zero, nil
	}

	typed, ok := out.(T)
	if !ok {
		return zero, New(TranslateWrongType).WithArgs(translatorName, zero, out).Render()
	}

	return typed, nil
}

// ToAsFrom is the generic version for a specific registry
func ToAsFrom[T any](r *Registry, err *Error, translatorName string) (T, error) {
	var zero T

	out, trErr := r.To(err, translatorName)
	if trErr != nil {
		return zero, trErr
	}

	if out == nil {
		return zero, nil
	}

	typed, ok := out.(T)
	if !ok {
		return zero, New(TranslateWrongType).WithArgs(translatorName, zero, out).Render()
	}

	return typed, nil
}
