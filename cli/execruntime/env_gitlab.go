package execruntime

const GITLAB = "gitlab"

var gitlabEnv = &RuntimeEnv{
	Id:        GITLAB,
	Name:      "GitLab CI",
	Namespace: "gitlab.com",
	Prefix:    "GITLAB",
	Identify: []Variable{
		{
			Name: "GITLAB_CI",
			Desc: "Mark that job is executed in GitLab CI environment",
		},
	},
	Variables: []Variable{
		{
			Name: "CI_PROJECT_URL",
			Desc: "The HTTP(S) address to access project",
		},
		{
			Name: "CI_PROJECT_TITLE",
			Desc: "The human-readable project name as displayed in the GitLab web interface",
		},
		{
			Name: "CI_PROJECT_ID",
			Desc: "The unique id of the current project that GitLab CI uses internally",
		},
		{
			Name: "CI_PROJECT_NAME",
			Desc: "The project name that is currently being built",
		},
		{
			Name: "CI_PROJECT_PATH",
			Desc: "The project namespace with the project name included",
		},
		{
			Name: "CI_PIPELINE_URL",
			Desc: "Pipeline details URL",
		},
		{
			Name: "CI_DEFAULT_BRANCH",
			Desc: "The name of the project's default branch.",
		},
		{
			Name: "CI_JOB_ID",
			Desc: "The unique id of the current job that GitLab CI uses internally",
		},
		{
			Name: "CI_JOB_URL",
			Desc: "Job details URL",
		},
		{
			Name: "CI_JOB_NAME",
			Desc: "The name of the job as defined in .gitlab-ci.yml",
		},
		{
			Name: "CI_JOB_STAGE",
			Desc: "The name of the job's stage",
		},
		{
			Name: "CI_COMMIT_SHA",
			Desc: "The commit revision for which project is built",
		},
		{
			Name: "CI_COMMIT_DESCRIPTION",
			Desc: "The description of the commit",
		},
		{
			Name: "CI_COMMIT_REF_NAME",
			Desc: "The branch or tag name for which project is built",
		},
		{
			Name: "CI_COMMIT_TAG",
			Desc: "The commit tag name",
		},
		{
			Name: "CI_COMMIT_BRANCH",
			Desc: "The commit branch name",
		},
		{
			Name: "CI_MERGE_REQUEST_ID",
			Desc: "The ID of the merge request if it's pipelines for merge requests",
		},
		{
			Name: "CI_MERGE_REQUEST_IID",
			Desc: "The project-level IID (internal ID) of the merge request. This ID is unique for the current project.",
		},
		{
			Name: "CI_MERGE_REQUEST_PROJECT_URL",
			Desc: "The URL of the project of the merge request if it's pipelines for merge requests",
		},
		{
			Name: "CI_MERGE_REQUEST_TITLE",
			Desc: "The title of the merge request",
		},
		{
			Name: "GITLAB_USER_NAME",
			Desc: "The real name of the user who started the job",
		},
		{
			Name: "GITLAB_USER_ID",
			Desc: "The id of the user who started the job",
		},
		{
			Name: "GITLAB_USER_EMAIL",
			Desc: "The email of the user who started the job",
		},
	},
}
