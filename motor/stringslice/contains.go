package stringslice

func Contains(slice []string, entry string) bool {
	for i := range slice {
		if slice[i] == entry {
			return true
		}
	}
	return false
}

func RemoveEmpty(a []string) []string {
	b := a[:0]
	for _, x := range a {
		if x != "" {
			b = append(b, x)
		}
	}
	for i := len(b); i < len(a); i++ {
		a[i] = "" // or the zero value of T
	}
	return b
}
