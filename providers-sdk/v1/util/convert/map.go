package convert

// MapToInterfaceMap converts a map[string]T to map[string]interface{}
func MapToInterfaceMap[T any](m map[string]T) map[string]interface{} {
	res := make(map[string]interface{})
	for k, v := range m {
		res[k] = v
	}
	return res
}
