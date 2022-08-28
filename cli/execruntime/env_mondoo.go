package execruntime

const MONDOO_CI = "mondoo-ci"

var mondooCIEnv = &RuntimeEnv{
	Id:        MONDOO_CI,
	Name:      "Mondoo CI",
	Namespace: "ci.mondoo.com",
	Identify: []Variable{
		{
			Name: "MONDOO_CI",
		},
	},
	Variables: []Variable{
		{
			Name: "CI_COMMIT_SHA",
			Desc: "The commit revision for which project is built",
		},
		{
			Name: "CI_COMMIT_MESSAGE",
			Desc: "The description of the commit",
		},
		{
			Name: "CI_COMMIT_REF_NAME",
			Desc: "The branch or tag name for which project is built",
		},
		{
			Name: "CI_COMMIT_URL",
			Desc: "Pull Request Url",
		},
		{
			Name: "CI_PROJECT_ID",
			Desc: "The unique id of the current project",
		},
		{
			Name: "CI_PROJECT_NAME",
			Desc: "The project name that is currently being built",
		},
		{
			Name: "CI_PROJECT_URL",
			Desc: "The HTTP(S) address to access project",
		},

		{
			Name: "CI_BUILD_ID",
			Desc: "Internal ID of the target system",
		},
		{
			Name: "CI_BUILD_NAME",
			Desc: "Build name",
		},
		{
			Name: "CI_BUILD_NUMBER",
			Desc: "Build number",
		},
		{
			Name: "CI_BUILD_URL",
			Desc: "The build URL",
		},
		{
			Name: "CI_BUILD_USER_NAME",
			Desc: "user that triggered the build",
		},
		{
			Name: "CI_BUILD_USER_ID",
			Desc: "user id that triggered the build",
		},
		{
			Name: "CI_BUILD_USER_EMAIL",
			Desc: "user email that triggered the build",
		},
	},
}
