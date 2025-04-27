package errorz

import "errors"

var (
	ErrLanguageUnsupported = errors.New("language not supported!")
)
