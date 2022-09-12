package execruntime

import (
	"path"
	"strings"

	"github.com/rs/zerolog/log"
)

var (
	environmentProvider envProvider
	environmentDef      CiConfig
)

func init() {
	environmentProvider = &osEnvProvider{}
	environmentDef = CiConfig{
		GITHUB:        githubEnv,
		GITLAB:        gitlabEnv,
		K8S_OPERATOR:  kubernetesEnv,
		CIRCLE:        circleciEnv,
		AZUREPIPELINE: azurePipelineEnv,
		JENKINS:       jenkinsEnv,
		// The following detections have been deactivated in scope of the v6 release
		// to only support the CI/CD detections that we can show in the UI.
		// At the time of writing, the ones supported are listed above.
		// We will be adding CI/CDs back throughout
		// TRAVIS:              travisEnv,
		// AWS_CODEBUILD:       awscodebuildEnv,
		// AWS_RUN_COMMAND:     awsruncommandEnv,
		// GOOGLE_CLOUD_BUILD:  googleCloudBuildEnv,
		// TERRAFORM:           terraformEnv,
		// PACKER:              packerEnv,
		// TEAMCITY:            teamcityEnv,
		// MONDOO_CI:           mondooCIEnv,
		// MONDOO_AWS_OPERATOR: mondooAwsOperatorEnv,
	}
}

type CiConfig map[string]*RuntimeEnv

type RuntimeEnv struct {
	Id        string
	Name      string `json:"name"`
	Prefix    string `json:"prefix"`
	Namespace string `json:"slug"`
	// Identify holds all env variables that must be set to
	// identify the CI environment
	Identify  []Variable `json:"identify"`
	Variables []Variable `json:"vars"`
}

type Variable struct {
	Name  string `json:"name"`
	Desc  string `json:"desc"`
	Value string `json:"value"`
	Type  string `json:"type"`
}

// Detect determines if we are running within the CI environment or not
func (c *RuntimeEnv) Detect() bool {
	for i := range c.Identify {
		id := c.Identify[i].Name
		if len(environmentProvider.Getenv(id)) != 0 {
			return true
		}
	}

	return false
}

func (c *RuntimeEnv) IsAutomatedEnv() bool {
	return c.Id != CLIENT_ENV
}

// Annotations returns env variables as key value pairs
func (c *RuntimeEnv) Labels() map[string]string {
	labels := map[string]string{}

	// store used ci environment in labels
	labels["mondoo.com/exec-environment"] = c.Namespace
	// iterate over all known ENV variables and fetch the data
	for i := range c.Variables {
		key := c.Variables[i].Name
		val := environmentProvider.Getenv(key)
		log.Debug().Msgf("cicd asset env var> %s : %s", key, val)

		// only store data if a value is set
		if len(val) > 0 {
			// replace prefix of variable name and make it lowercase
			valkey := strings.Replace(key, c.Prefix+"_", "", 1)
			slug := strings.ToLower(valkey)
			slug = strings.Replace(slug, "_", "-", -1)

			// check if the env var has the generic prefix
			if strings.HasPrefix(slug, "ci-") {
				slug = strings.Replace(slug, "ci-", "", 1)
			}

			// check if the env var has the env prefix already
			// if !strings.HasPrefix(slug, c.Id+"-") {
			// 	slug = fmt.Sprintf("%s-%s", c.Id, slug)
			// }

			log.Debug().Str("namespace", c.Namespace).Str("slug", slug).Msg("labels")
			labels[path.Join(c.Namespace, slug)] = val
		}
	}
	return labels
}
