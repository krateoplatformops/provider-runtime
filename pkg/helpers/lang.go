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

// Int64Ptr converts the supplied int64 to a pointer to that int64.
func Int64Ptr(p int64) *int64 { return &p }

// Int32Ptr converts the supplied int32 to a pointer to that int32.
func Int32Ptr(p int32) *int32 { return &p }

// BoolPtr converts the supplied bool to a pointer to that bool
func BoolPtr(p bool) *bool { return &p }

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
