package helpers

// String converts the supplied string pointer to a string, returning the
// empty string if the pointer is nil.
func String(v *string) string {
	return StringOrDefault(v, "")
}

// StringOrDefault converts the supplied string pointer to a string, returning a
// default string if the pointer is nil.
func StringOrDefault(v *string, def string) string {
	if v == nil {
		return def
	}
	return *v
}

// StringPtr converts the supplied string to a pointer to that string.
func StringPtr(p string) *string {
	return &p
}
