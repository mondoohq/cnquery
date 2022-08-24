package execruntime

const TRAVIS = "travis"

var travisEnv = &RuntimeEnv{
	Id:        TRAVIS,
	Name:      "Travis CI",
	Namespace: "travis-ci.com",
	Prefix:    "TRAVIS",
	Identify: []Variable{
		{
			Name: "TRAVIS",
		},
	},
	Variables: []Variable{
		{
			Name: "TRAVIS_BUILD_ID",
			Desc: "The id of the current build that Travis CI uses internally.",
		},
		{
			Name: "TRAVIS_BUILD_NUMBER",
			Desc: "The number of the current build (for example, “4”).",
		},
		{
			Name: "TRAVIS_BUILD_WEB_URL",
			Desc: "URL to the build log.",
		},
		{
			Name: "TRAVIS_COMMIT",
			Desc: "The commit that the current build is testing.",
		},
		{
			Name: "TRAVIS_COMMIT_MESSAGE",
			Desc: "The commit subject and body, unwrapped.",
		},
		{
			Name: "TRAVIS_JOB_ID",
			Desc: "The id of the current job that Travis CI uses internally.",
		},
		{
			Name: "TRAVIS_JOB_NAME",
			Desc: "The job name if it was specified, or ''.",
		},
		{
			Name: "TRAVIS_JOB_NUMBER",
			Desc: "The number of the current job (for example, “4.1”)",
		},
		{
			Name: "TRAVIS_JOB_WEB_URL",
			Desc: "URL to the job log.",
		},
		{
			Name: "TRAVIS_PULL_REQUEST",
			Desc: "The pull request number if the current job is a pull request, “false” if it’s not a pull request.",
		},
		{
			Name: "TRAVIS_PULL_REQUEST_BRANCH",
			Desc: "The name of the branch from which the PR originated.",
		},
		{
			Name: "TRAVIS_EVENT_TYPE",
			Desc: "Indicates how the build was triggered. One of push, pull_request, api, or cron",
		},
		{
			Name: "TRAVIS_BRANCH",
			Desc: "The name of the build for branch builds, the name of the target branch for pull request builds or the name of the tag for tag builds.",
		},
	},
}
