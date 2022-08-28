package execruntime

const AWS_CODEBUILD = "codebuild"

var awscodebuildEnv = &RuntimeEnv{
	Id:        AWS_CODEBUILD,
	Name:      "AWS CodeBuild",
	Prefix:    "CODEBUILD",
	Namespace: "codebuild.aws.amazon.com",
	Identify: []Variable{
		{
			Name: "CODEBUILD_BUILD_ID",
			Desc: "The CodeBuild ID of the build",
		},
	},
	Variables: []Variable{
		{
			Name: "CODEBUILD_BUILD_ID",
			Desc: "The CodeBuild ID of the build (for example, codebuild-demo-project:b1e6661e-e4f2-4156-9ab9-82a19EXAMPLE).",
		},
		{
			Name: "CODEBUILD_BUILD_ARN",
			Desc: "The Amazon Resource Name (ARN) of the build",
		},
		{
			Name: "CODEBUILD_RESOLVED_SOURCE_VERSION",
			Desc: "An identifier for the version of a build's source code.",
		},
		{
			Name: "CODEBUILD_SOURCE_REPO_URL",
			Desc: "The URL to the input artifact or source code repository.",
		},
		{
			Name: "AWS_REGION",
			Desc: "The AWS Region where the build is running",
		},
	},
}
