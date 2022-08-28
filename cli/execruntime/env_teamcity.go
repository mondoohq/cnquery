package execruntime

const TEAMCITY = "teamcity"

// see https://www.jetbrains.com/help/teamcity/predefined-build-parameters.html#Predefined+Server+Build+Parameters
var teamcityEnv = &RuntimeEnv{
	Id:        TEAMCITY,
	Name:      "TeamCity",
	Prefix:    "TEAMCITY",
	Namespace: "jetbrains.com",
	Identify: []Variable{
		{
			Name: "TEAMCITY_PROJECT_NAME",
			Desc: "The name of the project the current build belongs to.",
		},
	},
	Variables: []Variable{
		{
			Name: "TEAMCITY_PROJECT_NAME",
			Desc: "The name of the project the current build belongs to.",
		},
		{
			Name: "BUILD_NUMBER",
			Desc: "The build number assigned to the build by TeamCity.",
		},
	},
}
