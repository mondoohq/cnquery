package inspector

import "go.mondoo.io/mondoo/lumi"

// AnalyzeRuntime observers and their connections
func AnalyzeRuntime(runtime *lumi.Runtime) string {
	res := ""
	fwd, bck := runtime.Observers.List()
	for k, vs := range fwd {
		for _, v := range vs {
			res += k + " -> " + v + "\n"
		}
	}

	res += "\n----\n\n"
	for k, vs := range bck {
		for _, v := range vs {
			res += k + " -> " + v + "\n"
		}
	}

	return res
}
