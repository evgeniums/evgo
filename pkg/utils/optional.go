package utils

// Optional mimics std::optional to handle "Not Set" values without pointers.
type Optional[T any] struct {
	Value T
	IsSet bool
}

func Opt[T any](v T) Optional[T]  { return Optional[T]{Value: v, IsSet: true} }
func Nullopt[T any]() Optional[T] { return Optional[T]{IsSet: false} }

type OptString = Optional[string]

func NullString() OptString { return Nullopt[string]() }
