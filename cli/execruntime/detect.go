package execruntime

const CLIENT_ENV = "client"

func Detect() *RuntimeEnv {
	// check if it is a known ci environment
	for k := range environmentDef {
		env := environmentDef[k]
		if env.Detect() {
			return env
		}
	}

	// if we reach here, no environment matches, return empty env
	return &RuntimeEnv{
		Id:        CLIENT_ENV,
		Name:      "Mondoo Client",
		Namespace: "client.mondoo.com",
	}
}
