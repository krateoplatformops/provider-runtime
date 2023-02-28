package helpers

// Int32OrDefault sets eventually a default value for int32 type.
func Int32OrDefault(i *int32, def int32) *int32 {
	if i != nil {
		return i
	}
	return &def
}

// Int32 converts the supplied int32 pointer to an int, returning zero if
// the pointer is nil.
func Int32(v *int32) int32 {
	return *Int32OrDefault(v, 0)
}

// Int32Ptr converts the supplied int32 to a pointer to that int32.
func Int32Ptr(v int64) *int64 { return &v }

// Int64OrDefault sets eventually a default value for int64 type.
func Int64OrDefault(i *int64, def int64) *int64 {
	if i != nil {
		return i
	}
	return &def
}

// Int64 converts the supplied int64 pointer to an int, returning zero if
// the pointer is nil.
func Int64(v *int64) int64 {
	return *Int64OrDefault(v, 0)
}

// Int64Ptr converts the supplied int64 to a pointer to that int64.
func Int64Ptr(v int64) *int64 { return &v }

// Int6rDefault sets eventually a default value for int type.
func IntOrDefault(i *int, def int) *int {
	if i != nil {
		return i
	}
	return &def
}

// Int converts the supplied int pointer to an int, returning zero if
// the pointer is nil.
func Int(v *int) int {
	return *IntOrDefault(v, 0)
}

// IntPtr converts the supplied int to a pointer to that int.
func IntPtr(v int) *int { return &v }
