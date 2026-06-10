package api

// Ptr returns a pointer to the given value. Useful for assigning literals
// to *T fields in generated structs.
//
//	resp := TopicsResponse{Topics: t}
//	if id != "" { resp.ChatroomID = Ptr(id) }
func Ptr[T any](v T) *T {
	return &v
}

// Deref returns the value pointed to by p, or the zero value of T if p is nil.
func Deref[T any](p *T) T {
	if p == nil {
		var zero T
		return zero
	}
	return *p
}

// DerefOr returns the value pointed to by p, or the given default if p is nil.
func DerefOr[T any](p *T, def T) T {
	if p == nil {
		return def
	}
	return *p
}
