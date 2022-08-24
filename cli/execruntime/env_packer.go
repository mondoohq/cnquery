package execruntime

const PACKER = "packer"

var packerEnv = &RuntimeEnv{
	Id:        PACKER,
	Name:      "Packer",
	Namespace: "packer.io",
	Prefix:    "PACKER",
	Identify: []Variable{
		{
			Name: "PACKER_PIPELINE",
		},
	},
	Variables: []Variable{},
}
