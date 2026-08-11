package common

func Must[T any](val T, err any, messageArgs ...any) T {
	if err == nil {
		return val
	} else {
		panic(err)
	}
}

func ToPtr[T any](val T) *T {
	return &val
}

func SafeDeref[T any](ptr *T, fallback T) T {
	if ptr == nil {
		return fallback
	}

	return *ptr
}
