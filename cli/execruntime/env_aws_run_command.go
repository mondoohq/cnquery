package execruntime

const AWS_RUN_COMMAND = "aws_ssm_runcommand"

// see https://github.com/aws/amazon-ssm-agent/blob/master/agent/executers/executers.go
var awsruncommandEnv = &RuntimeEnv{
	Id:        AWS_RUN_COMMAND,
	Name:      "AWS SSM Run Command",
	Namespace: "ssm.aws.amazon.com",
	Prefix:    "AWS_SSM",
	Identify: []Variable{
		{
			Name: "AWS_SSM_INSTANCE_ID",
			Desc: "The AWS instance where ssm command is executed",
		},

		{
			Name: "AWS_SSM_REGION_NAME",
			Desc: "The AWS Region where ssm command is executed",
		},
	},
	Variables: []Variable{},
}
