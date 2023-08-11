package utils

func ToString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
