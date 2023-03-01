package sortx

import "sort"

func Keys[X any](m map[string]X) []string {
	res := make([]string, len(m))
	i := 0
	for k := range m {
		res[i] = k
		i++
	}
	sort.Strings(res)
	return res
}
