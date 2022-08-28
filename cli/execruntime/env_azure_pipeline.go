package execruntime

const AZUREPIPELINE = "azure"

var azurePipelineEnv = &RuntimeEnv{
	Id:        AZUREPIPELINE,
	Name:      "Azure Pipelines",
	Namespace: "devops.azure.com",
	Prefix:    "BUILD",
	Identify: []Variable{
		{
			Name: "TF_BUILD",
		},
	},
	Variables: []Variable{
		{
			Name: "BUILD_BUILDID",
			Desc: "The ID of the record for the completed build.",
		},
		{
			Name: "BUILD_BUILDNUMBER",
			Desc: "The name of the completed build.",
		},
		{
			Name: "BUILD_DEFINITIONNAME",
			Desc: "The name of the build pipeline.",
		},
		{
			Name: "BUILD_REASON",
			Desc: "The event that caused the build to run. Manual, IndividualCI, BatchedCI, Schedule, ValidateShelveset, CheckInShelveset, PullRequest, or ResourceTrigger.",
		},
		{
			Name: "BUILD_REPOSITORY_ID",
			Desc: "The unique identifier of the repository.",
		},
		{
			Name: "BUILD_REPOSITORY_URI",
			Desc: "The URL for the repository.",
		},
		{
			Name: "BUILD_REPOSITORY_NAME",
			Desc: "Name of the repository being built.",
		},
		{
			Name: "SYSTEM_PULLREQUEST_PULLREQUESTNUMBER",
			Desc: "Number of the pull request.",
		},
		{
			Name: "SYSTEM_PULLREQUEST_SOURCEBRANCH",
			Desc: "Branch name of the pull request.",
		},
		{
			Name: "SYSTEM_PULLREQUEST_SOURCEREPOSITORYURI",
			Desc: "URL of the pull request.",
		},
		{
			Name: "SYSTEM_JOBNAME",
			Desc: "Name of the job.",
		},
		{
			Name: "BUILD_REQUESTEDFOR",
			Desc: "The person who checked in the change that triggered the build.",
		},
		{
			Name: "BUILD_REQUESTEDFOREMAIL",
			Desc: "The e-mail of the person who checked in the change that triggered the build.",
		},
		{
			Name: "BUILD_SOURCEBRANCHNAME",
			Desc: "The name of the branch the build was queued for.",
		},
		{
			Name: "BUILD_SOURCEVERSION",
			Desc: "The latest version control change that is included in this build.",
		},
		{
			Name: "BUILD_SOURCEVERSIONAUTHOR",
			Desc: "The author of the change that is included in this build.",
		},
		{
			Name: "BUILD_SOURCEVERSIONMESSAGE",
			Desc: "The comment of the commit or changeset.",
		},
		{
			Name: "BUILD_PULLREQUEST_PULLREQUESTID",
			Desc: "The ID of the pull request that caused this build.",
		},
		{
			Name: "BUILD_PULLREQUEST_PULLREQUESTNUMBER",
			Desc: "The number of the pull request that caused this build.",
		},
		{
			Name: "BUILD_TEAMFOUNDATIONCOLLECTIONURI",
			Desc: "The URI of the team foundation collection.",
		},
		{
			Name: "BUILD_TEAMPROJECT",
			Desc: "The name of the project that contains this build.",
		},
		{
			Name: "BUILD_TEAMPROJECTID",
			Desc: "The ID of the project that this build belongs to.",
		},
	},
}
