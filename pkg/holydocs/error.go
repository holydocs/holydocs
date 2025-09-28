package holydocs

import (
	"fmt"
)

// UnsupportedFormatModeError represents an error when an unsupported format mode is used.
type UnsupportedFormatModeError struct {
	Mode     FormatMode
	Expected []FormatMode
}

// Error returns the error message for UnsupportedFormatModeError.
func (e UnsupportedFormatModeError) Error() string {
	return fmt.Sprintf("unsupported format mode: %s, expected one of: %v", e.Mode, e.Expected)
}

// NewUnsupportedFormatModeError creates a new UnsupportedFormatModeError.
func NewUnsupportedFormatModeError(mode FormatMode, expected []FormatMode) UnsupportedFormatModeError {
	return UnsupportedFormatModeError{
		Mode:     mode,
		Expected: expected,
	}
}

// UnsupportedFormatError represents an error when an unsupported format type is used.
type UnsupportedFormatError struct {
	Format   TargetType
	Expected TargetType
}

// Error returns the error message for UnsupportedFormatError.
func (e UnsupportedFormatError) Error() string {
	return fmt.Sprintf("unsupported format type: %s, expected: %s", e.Format, e.Expected)
}

// NewUnsupportedFormatError creates a new UnsupportedFormatError.
func NewUnsupportedFormatError(format, expected TargetType) UnsupportedFormatError {
	return UnsupportedFormatError{
		Format:   format,
		Expected: expected,
	}
}
