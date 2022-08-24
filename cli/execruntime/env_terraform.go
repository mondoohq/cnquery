package execruntime

const TERRAFORM = "terraform"

var terraformEnv = &RuntimeEnv{
	Id:        TERRAFORM,
	Name:      "Terraform",
	Namespace: "terraform.io",
	Prefix:    "TERRAFORM",
	Identify: []Variable{
		{
			Name: "TERRAFORM_PIPELINE",
		},
	},
	Variables: []Variable{},
}
