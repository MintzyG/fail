package fail

// IDs
var (
	UnregisteredError           = internalID(0, 0, false, "FailUnregisteredError")
	TranslateUnregisteredError  = internalID(0, 1, false, "FailTranslateUnregisteredError")
	TranslatorNotFound          = internalID(0, 2, false, "FailTranslatorNotFound")
	TranslateUnsupportedError   = internalID(0, 3, false, "FailTranslateUnsupportedError")
	TranslatePanicked           = internalID(0, 4, false, "FailTranslatorPanicked")
	TranslateWrongType          = internalID(0, 5, false, "FailTranslateWrongType")
	MultipleErrors              = internalID(0, 6, false, "FailMultipleErrors")
	UnknownError                = internalID(0, 7, false, "FailUnknownError")
	NotMatchedInAnyMapper       = internalID(0, 8, false, "FailNotMatchedInAnyMapper")
	NoMapperRegistered          = internalID(0, 9, false, "FailNoMapperRegistered")
	TranslatorAlreadyRegistered = internalID(0, 10, false, "FailTranslatorAlreadyRegistered")
	RuntimeIDInvalid            = internalID(9, 11, false, "FailRuntimeIDInvalid")
	UnregisteredIDError         = internalID(9, 12, false, "FailIDNotRegisteredError")
	RegisterManyError           = internalID(9, 13, false, "FailRegisterManyError")
	RegistryAlreadyRegistered   = internalID(9, 14, false, "FailRegistryAlreadyRegistered")

	TranslatorNil       = internalID(0, 0, true, "FailTranslatorNil")
	TranslatorNameEmpty = internalID(0, 1, true, "FailTranslatorNameEmpty")
)

type UNSET struct{}

// Sentinels
var (
	errUnregisteredError           = Form(UnregisteredError, "error with ID(%s) is not registered in the registry", true, nil, "ID NOT SET")
	errTranslateWrongType          = Form(TranslateWrongType, "%s translator returned unexpected type: expected(%T) got(T)", true, nil, "UNSET TRANSLATOR NAME", UNSET{}, UNSET{})
	errTranslateUnregisteredError  = Form(TranslateUnregisteredError, "tried translating an unregistered error", true, nil)
	errTranslateNotFound           = Form(TranslatorNotFound, "couldn't find translator: %s", true, nil, "UNSET TRANSLATOR NAME")
	errTranslateUnsupportedError   = Form(TranslateUnsupportedError, "error not supported by %s translator", true, nil, "UNSET TRANSLATOR NAME")
	errTranslatePanicked           = Form(TranslatePanicked, "%s translator panicked during translation", true, nil, "UNSET TRANSLATOR NAME")
	errTranslatorAlreadyRegistered = Form(TranslatorAlreadyRegistered, "translator already registered", true, nil)
	errTranslatorNil               = Form(TranslatorNil, "cannot register nil translator", true, nil)
	errTranslatorNameEmpty         = Form(TranslatorNameEmpty, "translator must have a non-empty name", true, nil)
	errNotMatchedInAnyMapper       = Form(NotMatchedInAnyMapper, "error wasn't matched/mapped by any mapper", true, nil)
	errNoMapperRegistered          = Form(NoMapperRegistered, "no mapper is registered in the registry", true, nil)
	errMultipleErrors              = Form(MultipleErrors, "multiple errors occurred", false, nil)
	errUnknownError                = Form(UnknownError, "unknown error", true, nil)
	errRuntimeInvalidID            = Form(RuntimeIDInvalid, "all error IDs must be defined at package initialization time and not runtime", true, nil)
	errUnregisteredIDError         = Form(UnregisteredIDError, "ID(%s) is not registered in the ID registry", true, nil, "UNSET ID")
	errRegisterManyError           = Form(RegisterManyError, "one or more errors occurred during error registering", true, nil)
	errRegistryAlreadyRegistered   = Form(RegistryAlreadyRegistered, "%s registry already registered", true, nil, "UNSET REGISTRY NAME")
)
