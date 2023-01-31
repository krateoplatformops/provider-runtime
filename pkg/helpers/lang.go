package helpers

// IsBoolPtrEqualToBool compares a *bool with bool
func IsBoolPtrEqualToBool(bp *bool, b bool) bool {
	if bp == nil {
		return false
	}

	return (*bp == b)
}

// IsIntEqualToIntPtr compares an *int with int
func IsIntEqualToIntPtr(ip *int, i int) bool {
	if ip == nil {
		return false
	}
	return (*ip == i)
}

// IntPtrValue return the *int value or a default
func IntPtrValue(ip *int, i int) int {
	if ip == nil {
		return i
	}
	return *ip
}

// StringValue converts the supplied string pointer to a string, returning the
// empty string if the pointer is nil.
func StringValue(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

// Int64Value converts the supplied int64 pointer to an int, returning zero if
// the pointer is nil.
func Int64Value(v *int64) int64 {
	if v == nil {
		return 0
	}
	return *v
}

// Int32Value converts the supplied int32 pointer to an int, returning zero if
// the pointer is nil.
func Int32Value(v *int32) int32 {
	if v == nil {
		return 0
	}
	return *v
}

// BoolValue converts the supplied bool pointer to an bool, returning false if
// the pointer is nil.
func BoolValue(v *bool) bool {
	if v == nil {
		return false
	}
	return *v
}

// StringPtr converts the supplied string to a pointer to that string.
func StringPtr(p string) *string { return &p }

// Int64Ptr converts the supplied int64 to a pointer to that int64.
func Int64Ptr(p int64) *int64 { return &p }

// Int32Ptr converts the supplied int32 to a pointer to that int32.
func Int32Ptr(p int32) *int32 { return &p }

// BoolPtr converts the supplied bool to a pointer to that bool
func BoolPtr(p bool) *bool { return &p }

// LateInitialize functions initialize s(first argument), presumed to be an
// optional field of a Kubernetes API object's spec per Kubernetes
// "late initialization" semantics. s is returned unchanged if it is non-nil
// or from(second argument) is the empty string, otherwise a pointer to from
// is returned.
// https://github.com/kubernetes/community/blob/db7f270f/contributors/devel/sig-architecture/api-conventions.md#optional-vs-required
// https://github.com/kubernetes/community/blob/db7f270f/contributors/devel/sig-architecture/api-conventions.md#late-initialization

// LateInitializeString implements late initialization for string type.
func LateInitializeString(s *string, from string) *string {
	if s != nil || from == "" {
		return s
	}
	return &from
}

// LateInitializeInt64 implements late initialization for int64 type.
func LateInitializeInt64(i *int64, from int64) *int64 {
	if i != nil || from == 0 {
		return i
	}
	return &from
}

// LateInitializeInt32 implements late initialization for int32 type.
func LateInitializeInt32(i *int32, from int32) *int32 {
	if i != nil || from == 0 {
		return i
	}
	return &from
}

// LateInitializeBool implements late initialization for bool type.
func LateInitializeBool(b *bool, from bool) *bool {
	if b != nil || !from {
		return b
	}
	return &from
}

// BoolOrDefault sets eventually a default value for bool type.
func BoolOrDefault(b *bool, def bool) *bool {
	if b != nil {
		return b
	}
	return &def
}

// Int32OrDefault sets eventually a default value for int32 type.
func Int32OrDefault(i *int32, def int32) *int32 {
	if i != nil {
		return i
	}
	return &def
}

// Int64OrDefault sets eventually a default value for int64 type.
func Int64OrDefault(i *int64, def int64) *int64 {
	if i != nil {
		return i
	}
	return &def
}

// StringOrDefault sets eventually a default value for string type.
func StringOrDefault(s *string, def string) *string {
	if s != nil {
		return s
	}
	return &def
}

// StringSliceContains contains checks if a string is present in a slice.
func StringSliceContains(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}

	return false
}
