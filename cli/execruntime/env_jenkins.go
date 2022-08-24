package execruntime

const JENKINS = "jenkins"

var jenkinsEnv = &RuntimeEnv{
	Id:        JENKINS,
	Name:      "Jenkins CI",
	Namespace: "jenkins.io",
	Identify: []Variable{
		{
			Name: "JENKINS_URL",
		},
	},
	Variables: []Variable{
		{
			Name: "GIT_COMMIT",
			Desc: "Git hash of the commit checked out for the build",
		},
		{
			Name: "JENKINS_URL",
			Desc: "Set to the URL of the Jenkins master that's running the build.",
		},
		{
			Name: "JOB_NAME",
			Desc: "Name of the project of this build.",
		},
		{
			Name: "BUILD_URL",
			Desc: "The URL where the results of this build can be found (e.g. http://buildserver/jenkins/job/MyJobName/666/)",
		},
		{
			Name: "BUILD_NUMBER",
			Desc: "The current build number, such as '153'",
		},
		{
			Name: "BUILD_ID",
			Desc: "The current build id, such as '2005-08-22_23-59-59' (YYYY-MM-DD_hh-mm-ss)",
		},
		{
			Name: "CHANGE_AUTHOR",
			Desc: "For a multibranch project corresponding to some kind of change request, this will be set to the username of the author of the proposed change.",
		},
		{
			Name: "CHANGE_AUTHOR_DISPLAY_NAME",
			Desc: "For a multibranch project corresponding to some kind of change request, this will be set to the human name of the author.",
		},
		{
			Name: "CHANGE_AUTHOR_EMAIL",
			Desc: "For a multibranch project corresponding to some kind of change request, this will be set to the email address of the author.",
		},
		{
			Name: "GIT_URL",
			Desc: "The remote URL. If there are multiple, will be GIT_URL_1, GIT_URL_2, etc.",
		},
		{
			Name: "GIT_BRANCH",
			Desc: "For a multibranch project, this will be set to the name of the branch being built, for example in case you wish to deploy to production from master but not from feature branches; if corresponding to some kind of change request, the name is generally arbitrary (refer to CHANGE_ID and CHANGE_TARGET).",
		},
		{
			Name: "BRANCH_NAME",
			Desc: "For a multibranch project, this will be set to the name of the branch being built, for example in case you wish to deploy to production from master but not from feature branches; if corresponding to some kind of change request, the name is generally arbitrary (refer to CHANGE_ID and CHANGE_TARGET).",
		},
		{
			Name: "CHANGE_BRANCH",
			Desc: "For a multibranch project corresponding to some kind of change request, this will be set to the name of the actual head on the source control system which may or may not be different from BRANCH_NAME. For example in GitHub or Bitbucket this would have the name of the origin branch whereas BRANCH_NAME would be something like PR-24.",
		},
		{
			Name: "CHANGE_TARGET",
			Desc: "For a multibranch project corresponding to some kind of change request, this will be set to the target or base branch to which the change could be merged, if supported; else unset.",
		},
		{
			Name: "BRANCH_IS_PRIMARY",
			Desc: "For a multibranch project, if the SCM source reports that the branch being built is a primary branch, this will be set to true; else unset. Some SCM sources may report more than one branch as a primary branch while others may not supply this information.",
		},
		{
			Name: "CHANGE_TITLE",
			Desc: "For a multibranch project corresponding to some kind of change request, this will be set to the change, if supported; else unset.",
		},
		{
			Name: "TAG_NAME",
			Desc: "For a multibranch project corresponding to some kind of tag, this will be set to the name of the tag being built, if supported; else unset.",
		},
	},
}
