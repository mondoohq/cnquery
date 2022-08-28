package execruntime

const MONDOO_AWS_OPERATOR = "mondoo-aws-operator"

var mondooAwsOperatorEnv = &RuntimeEnv{
	Id:        MONDOO_AWS_OPERATOR,
	Name:      "Mondoo AWS Operator",
	Namespace: "aws-ops.mondoo.com",
	Identify: []Variable{
		{
			Name: "AWS_LAMBDA_RUNTIME_API",
		},
	},
	Variables: []Variable{},
}
