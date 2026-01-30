package fail

import "fmt"

// Translator converts Error to another format (HTTP response, gRPC status, etc.)
type Translator interface {
	Name() string
	Translate(*Error) any
}

// RegisterTranslator adds a translator for converting errors to other formats
func RegisterTranslator(t Translator) {
	global.RegisterTranslator(t)
}

func (r *Registry) RegisterTranslator(t Translator) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.translators[t.Name()] = t
}

// Translate converts an error using the named translator
func Translate(err *Error, translatorName string) (any, error) {
	return global.Translate(err, translatorName)
}

func (r *Registry) Translate(err *Error, translatorName string) (any, error) {
	r.mu.RLock()
	translator, exists := r.translators[translatorName]
	r.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("translator %s not found", translatorName)
	}

	return translator.Translate(err), nil
}
