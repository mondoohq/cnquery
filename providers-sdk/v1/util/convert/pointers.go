package convert

func ToBool(ptr *bool) bool {
	if ptr == nil {
		return false
	}
	return *ptr
}

func ToIntFrom32(ptr *int32) int {
	if ptr == nil {
		return 0
	}
	return int(*ptr)
}

func ToString(ptr *string) string {
	if ptr == nil {
		return ""
	}
	return *ptr
}

func ToInt64(ptr *int64) int64 {
	if ptr == nil {
		return 0
	}
	return *ptr
}

func ToInt64From32(ptr *int32) int64 {
	if ptr == nil {
		return 0
	}
	return int64(*ptr)
}

func ToFloat64(ptr *float64) float64 {
	if ptr == nil {
		return 0
	}
	return *ptr
}

func ToListFromPtrs(ptrs []*string) []string {
	if ptrs == nil {
		return nil
	}
	list := make([]string, len(ptrs))
	for i, ptr := range ptrs {
		list[i] = ToString(ptr)
	}
	return list
}
