package helpers

// BoolOrDefault converts the supplied bool pointer to a bool, returning a
// default bool if the pointer is nil.
func BoolOrDefault(v *bool, def bool) bool {
	if v == nil {
		return def
	}
	return *v
}

// BoolPtr converts the supplied bool to a pointer to that bool.
func BoolPtr(v bool) *bool {
	return &v
}

// Bool converts the supplied bool pointer to an bool, returning false if
// the pointer is nil.
func Bool(v *bool) bool {
	return BoolOrDefault(v, false)
}
