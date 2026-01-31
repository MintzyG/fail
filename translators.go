package fail

// Translator converts a trusted registry Error into another format
// (HTTP response, gRPC status, CLI output, etc.)
type Translator interface {
	Name() string

	// Supports reports whether this translator is allowed to translate this error.
	// This is for capability, boundary, or policy checks — not mapping.
	Supports(*Error) bool

	// Translate converts the error into an external representation.
	// It may still fail even if Supports returned true (e.g. missing metadata).
	Translate(*Error) (any, error)
}

// RegisterTranslator adds a translator for converting errors to other formats
func RegisterTranslator(t Translator) error {
	return global.RegisterTranslator(t)
}

var TranslatorAlreadyRegistered = internalID(0, 0, false, "FailTranslatorAlreadyRegistered")
var ErrTranslatorAlreadyRegistered = Form(TranslatorAlreadyRegistered, "translator already registered", true, nil)

var TranslatorNil = internalID(0, 0, true, "FailTranslatorNil")
var ErrTranslatorNil = Form(TranslatorNil, "cannot register nil translator", true, nil)

var TranslatorNameEmpty = internalID(0, 1, true, "FailTranslatorNameEmpty")
var ErrTranslatorNameEmpty = Form(TranslatorNameEmpty, "translator must have a non-empty name", true, nil)

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

func MustRegisterTranslator(t Translator) {
	if err := RegisterTranslator(t); err != nil {
		panic(err)
	}
}

func Translate(err *Error, translatorName string) (any, error) {
	return global.Translate(err, translatorName)
}

var TranslateUntrustedError = internalID(0, 1, false, "FailTranslateUntrustedError")
var ErrTranslateUntrustedError = Form(TranslateUntrustedError, "tried translating unregistered error", true, nil)

var TranslateNotFound = internalID(0, 2, false, "FailTranslatorNotFound")
var ErrTranslateNotFound = Form(TranslateNotFound, "couldn't find translator", true, nil)

var TranslateUnsupportedError = internalID(0, 3, false, "FailTranslateUnsupportedError")
var ErrTranslateUnsupportedError = Form(TranslateUnsupportedError, "can't translate unsupported error", true, nil)

var TranslatePanic = internalID(0, 4, false, "FailTranslatorPanic")
var ErrTranslatePanic = Form(TranslatePanic, "translator panicked during translation", true, nil)

// Translate converts an error using the named translator
func (r *Registry) Translate(err *Error, translatorName string) (out any, retErr error) {
	if err == nil {
		return nil, nil
	}

	if !err.trusted {
		return nil, New(TranslateUntrustedError).With(err)
	}

	r.mu.RLock()
	translator, exists := r.translators[translatorName]
	r.mu.RUnlock()

	if !exists {
		return nil, New(TranslateNotFound).AddMeta("name", translatorName)
	}

	if !translator.Supports(err) {
		return nil, New(TranslateUnsupportedError).With(err)
	}

	r.hooks.runTranslate(err, map[string]any{
		"translator": translatorName,
	})

	defer func() {
		if rec := recover(); rec != nil {
			retErr = New(TranslatePanic).With(err). // original error being translated
								AddMeta("translator", translatorName).
								AddMeta("panic", rec)
		}
	}()

	return translator.Translate(err)
}

var TranslateWrongType = internalID(0, 5, false, "FailTranslateWrongType")
var ErrTranslateWrongType = Form(TranslateWrongType, "translator returned unexpected type", true, nil)

func TranslateAs[T any](err *Error, translatorName string) (T, error) {
	return TranslateAsFrom[T](global, err, translatorName)
}

func TranslateAsFrom[T any](r *Registry, err *Error, translatorName string) (T, error) {
	var zero T

	out, trErr := r.Translate(err, translatorName)
	if trErr != nil {
		return zero, trErr
	}

	if out == nil {
		return zero, nil
	}

	typed, ok := out.(T)
	if !ok {
		return zero, New(TranslateWrongType).
			With(err).
			AddMeta("translator", translatorName)
	}

	return typed, nil
}
