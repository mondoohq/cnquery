package stringslice

func Contains(slice []string, entry string) bool {
	for i := range slice {
		if slice[i] == entry {
			return true
		}
	}
	return false
}
