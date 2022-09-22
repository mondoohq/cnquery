package builder

import (
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"go.mondoo.com/cnquery/motor/asset"
	"go.mondoo.com/cnquery/motor/motorid/awsec2"
	"go.mondoo.com/cnquery/motor/providers"
	"go.mondoo.com/cnquery/motor/providers/awsec2ebs"
	"go.mondoo.com/cnquery/motor/vault"
)

type target struct {
	Hostname string
	Username string
	Path     string
	Port     int32
}

// parseTarget parses the specified target, which may be specified as either:
// - [user@]hostname or
// - a URI of the form ssh://[user@]hostname[:port]
func parseTarget(uri string) (target, error) {
	var target target
	if !strings.Contains(uri, "://") {
		uri = "//" + uri
	}

	u, err := url.Parse(uri)
	if err != nil {
		return target, err
	}
	target.Hostname = u.Hostname()
	target.Path = u.Path
	target.Username = u.User.Username()

	if u.Port() != "" {
		port, err := strconv.Atoi(u.Port())
		if err != nil {
			return target, err
		}
		target.Port = int32(port)
	}
	return target, nil
}

func ParseTargetAsset(cmd *cobra.Command, args []string, providerType providers.ProviderType, assetType AssetType) *asset.Asset {
	username := ""
	password, _ := cmd.Flags().GetString("password")
	identityFile, _ := cmd.Flags().GetString("identity-file")

	// parse options
	optionData, err := cmd.Flags().GetStringToString("option")
	if err != nil {
		log.Error().Err(err).Msg("cannot parse --option values")
	}
	if optionData == nil {
		optionData = map[string]string{}
	}

	filepath, _ := cmd.Flags().GetString("path")

	discoverTargets := []string{}
	discover, err := cmd.Flags().GetString("discover")
	if err == nil && discover != "" {
		discoverTargets = strings.Split(discover, ",")

		// sanitization, remove whitespace
		for i := range discoverTargets {
			discoverTargets[i] = strings.TrimSpace(discoverTargets[i])
		}
	}

	// parse discovery filter option
	discoveryFilter, err := cmd.Flags().GetStringToString("discover-filter")
	if err != nil {
		log.Error().Err(err).Msg("cannot parse --discover-filter values")
	}

	// label and annotation are not supported for all commands therefore we ignore the error
	labels, _ := cmd.Flags().GetStringToString("label")
	annotations, err := cmd.Flags().GetStringToString("annotation")

	parsedAsset := &asset.Asset{
		Options:     map[string]string{},
		Labels:      labels,
		Annotations: annotations,
		Connections: []*providers.Config{},
	}

	connection := &providers.Config{
		Backend: providers.ProviderType_LOCAL_OS, // TODO: need to be set to unknown at this point
		Discover: &providers.Discovery{
			Targets: discoverTargets,
			Filter:  discoveryFilter,
		},
		Credentials: []*vault.Credential{},
		Options:     optionData,
	}

	log.Debug().Str("provider", providerType.String()).Int64("asset-type", int64(assetType)).Msg("parsing asset")
	switch providerType {
	case providers.ProviderType_LOCAL_OS:
		if assetType == UnknownAssetType {
			return nil
		} else {
			connection.Backend = providerType
		}
	case providers.ProviderType_MOCK:
		connection.Backend = providerType
		connection.Options["path"] = filepath
	case providers.ProviderType_VAGRANT:
		connection.Backend = providerType
		connection.Host = args[0]
	case providers.ProviderType_TERRAFORM:
		connection.Backend = providerType
		connection.Options["path"] = filepath
		// if the asset type is set, we scan either plan or state
		switch assetType {
		case TerraformHclAssetType:
			connection.Options["asset-type"] = "hcl"
		case TerraformPlanAssetType:
			connection.Options["asset-type"] = "plan"
		case TerraformStateAssetType:
			connection.Options["asset-type"] = "state"
		default:
			log.Fatal().Msg("asset type must be set for terraform")
		}
	case providers.ProviderType_SSH:
		connection.Backend = providerType
		target, err := parseTarget(args[0])
		if err != nil {
			log.Error().Err(err).Msg("cannot parse target")
		}
		connection.Host = target.Hostname
		connection.Port = target.Port
		connection.Path = target.Path
		username = target.Username

		switch assetType {
		case Ec2InstanceConnectAssetType:
			connection.Credentials = append(connection.Credentials, &vault.Credential{
				Type: vault.CredentialType_aws_ec2_instance_connect,
				User: username,
			})
		case DefaultAssetType:
			if password != "" {
				connection.Credentials = append(connection.Credentials, &vault.Credential{
					Type:     vault.CredentialType_password,
					User:     username,
					Password: password,
				})
			}
			if identityFile != "" {
				connection.Credentials = append(connection.Credentials, &vault.Credential{
					Type:           vault.CredentialType_private_key,
					User:           username,
					PrivateKeyPath: identityFile,
				})
			}
		}
	case providers.ProviderType_WINRM:
		connection.Backend = providerType
		target, err := parseTarget(args[0])
		if err != nil {
			log.Error().Err(err).Msg("cannot parse target")
		}
		connection.Host = target.Hostname
		connection.Port = target.Port
		connection.Path = target.Path
		username = target.Username
		connection.Credentials = append(connection.Credentials, &vault.Credential{
			Type:     vault.CredentialType_password,
			User:     username,
			Password: password,
		})
	case providers.ProviderType_DOCKER_ENGINE_CONTAINER:
		connection.Backend = providerType
		connection.Host = args[0]
	case providers.ProviderType_DOCKER_ENGINE_IMAGE:
		connection.Backend = providerType
		connection.Host = args[0]
	case providers.ProviderType_CONTAINER_REGISTRY:
		switch assetType {
		case GcrContainerRegistryAssetType:
			url := args[0]

			if x, err := cmd.Flags().GetString("repository"); err != nil {
				log.Fatal().Err(err).Msg("cannot parse --repository value")
			} else if x != "" {
				if url == "" {
					log.Fatal().Msg("please provide a GCR project for your scan")
				}
				url += "/" + x
			}

			connection.Backend = providerType
			connection.Host = "gcr.io" + url
		default:
			connection.Backend = providerType
			connection.Host = args[0]
		}
	case providers.ProviderType_DOCKER:
		connection.Backend = providerType
		connection.Host = args[0]
	case providers.ProviderType_K8S:
		connection.Backend = providerType

		if filepath != "" {
			if _, err := os.Stat(filepath); os.IsNotExist(err) {
				log.Fatal().Str("file", filepath).Msg("Could not find the Kubernetes manifest file. Please specify the correct path.")
			}
			connection.Options["path"] = filepath
		}

		if namespace, err := cmd.Flags().GetString("namespace"); err != nil {
			log.Fatal().Err(err).Msg("cannot parse --namespace values")
		} else if namespace != "" {
			connection.Options["namespace"] = namespace
		}

		if allNamespaces, err := cmd.Flags().GetBool("all-namespaces"); err != nil {
			log.Fatal().Err(err).Msg("cannot parse --all-namespaces values")
		} else if allNamespaces == true {
			connection.Options["all-namespaces"] = "true"
		}
	case providers.ProviderType_AWS:
		connection.Backend = providerType
		if profile, err := cmd.Flags().GetString("profile"); err != nil {
			log.Fatal().Err(err).Msg("cannot parse --profile value")
		} else if profile != "" {
			connection.Options["profile"] = profile
		}

		if region, err := cmd.Flags().GetString("region"); err != nil {
			log.Fatal().Err(err).Msg("cannot parse --region values")
		} else if region != "" {
			connection.Options["region"] = region
		}
	case providers.ProviderType_AWS_EC2_EBS:
		assembleAwsEc2EbsConnectionUrl := func(flagAsset *asset.Asset, arg string, targetType string) string {
			instanceInfo, err := awsec2ebs.GetRawInstanceInfo(flagAsset.Options["profile"])
			if err != nil {
				log.Fatal().Err(err).Msg("cannot detect instance info")
			}
			targetRegion := instanceInfo.Region
			if flagAsset.Options["region"] != "" {
				targetRegion = flagAsset.Options["region"]
			}
			return "account/" + instanceInfo.AccountID + "/region/" + targetRegion + "/" + targetType + "/" + arg
		}

		targetDestination := ""
		switch assetType {
		case Ec2ebsInstanceAssetType:
			targetDestination = assembleAwsEc2EbsConnectionUrl(parsedAsset, args[0], awsec2ebs.EBSTargetInstance)
		case Ec2ebsVolumeAssetType:
			targetDestination = assembleAwsEc2EbsConnectionUrl(parsedAsset, args[0], awsec2ebs.EBSTargetVolume)
		case Ec2ebsSnapshotAssetType:
			targetDestination = assembleAwsEc2EbsConnectionUrl(parsedAsset, args[0], awsec2ebs.EBSTargetSnapshot)
		default:
			log.Fatal().Msg("asset type must be set for aws-ec2-ebs")
		}

		target, err := awsec2ebs.ParseEbsTransportUrl(targetDestination)
		if err != nil {
			log.Error().Err(err).Msg("cannot parse target")
		}
		var platformId string
		switch target.Type {
		case awsec2ebs.EBSTargetInstance:
			platformId = awsec2.MondooInstanceID(target.Account, target.Region, target.Id)
		case awsec2ebs.EBSTargetVolume:
			platformId = awsec2.MondooVolumeID(target.Account, target.Region, target.Id)
		case awsec2ebs.EBSTargetSnapshot:
			platformId = awsec2.MondooSnapshotID(target.Account, target.Region, target.Id)
		}

		connection.Backend = providerType
		connection.PlatformId = platformId
		connection.Options = map[string]string{
			"account": target.Account,
			"region":  target.Region,
			"id":      target.Id,
			"type":    target.Type,
		}
	case providers.ProviderType_AWS_SSM_RUN_COMMAND:
		connection.Backend = providers.ProviderType_SSH // TODO: allow the usage the provider type here
		connection.Credentials = append(connection.Credentials, &vault.Credential{
			Type: vault.CredentialType_aws_ec2_ssm_session,
			User: username,
		})
	case providers.ProviderType_AZURE:
		connection.Backend = providerType
		if subscription, err := cmd.Flags().GetString("subscription"); err != nil {
			log.Fatal().Err(err).Msg("cannot parse --subscription value")
		} else if subscription != "" {
			connection.Options["subscriptionID"] = subscription
		}
	case providers.ProviderType_GCP:
		connection.Backend = providerType

		if project, err := cmd.Flags().GetString("project"); err != nil {
			log.Fatal().Err(err).Msg("cannot parse --project value")
		} else if project != "" {
			connection.Options["project"] = project
		}

		if organization, err := cmd.Flags().GetString("organization"); err != nil {
			log.Fatal().Err(err).Msg("cannot parse --organization value")
		} else if organization != "" {
			connection.Options["organization"] = organization
		}
	case providers.ProviderType_VSPHERE:
		connection.Backend = providerType
		target, err := parseTarget(args[0])
		if err != nil {
			log.Error().Err(err).Msg("cannot parse target")
		}
		connection.Host = target.Hostname
		connection.Port = target.Port
		connection.Path = target.Path
		username = target.Username
		connection.Credentials = append(connection.Credentials, &vault.Credential{
			Type:     vault.CredentialType_password,
			User:     username,
			Password: password,
		})
	case providers.ProviderType_VSPHERE_VM:
		connection.Backend = providerType
		target, err := parseTarget(args[0])
		if err != nil {
			log.Error().Err(err).Msg("cannot parse target")
		}
		connection.Host = target.Hostname
		connection.Port = target.Port
		connection.Path = target.Path
		username = target.Username
		connection.Credentials = append(connection.Credentials, &vault.Credential{
			Type:     vault.CredentialType_password,
			User:     username,
			Password: password,
		})
	case providers.ProviderType_GITHUB:
		switch assetType {
		case GithubOrganizationAssetType:
			connection.Backend = providerType
			connection.Options["organization"] = args[0]

			if x, err := cmd.Flags().GetString("token"); err != nil {
				log.Fatal().Err(err).Msg("cannot parse --token value")
			} else if x != "" {
				connection.Credentials = append(connection.Credentials, &vault.Credential{
					Type:     vault.CredentialType_password,
					Password: x,
				})
			}
		case GithubRepositoryAssetType:
			connection.Backend = providerType
			paths := strings.Split(args[0], "/")

			if len(paths) != 2 {
				log.Fatal().Msg("please provide a GitHub repository in the form of <organization>/<repository>")
			}

			owner := paths[0]
			repo := paths[1]

			connection.Options["owner"] = owner
			connection.Options["repository"] = repo

			if x, err := cmd.Flags().GetString("token"); err != nil {
				log.Fatal().Err(err).Msg("cannot parse --token value")
			} else if x != "" {
				connection.Credentials = append(connection.Credentials, &vault.Credential{
					Type:     vault.CredentialType_password,
					Password: x,
				})
			}
		case GithubUserAssetType:
			connection.Backend = providerType
			connection.Options["user"] = args[0]

			if x, err := cmd.Flags().GetString("token"); err != nil {
				log.Fatal().Err(err).Msg("cannot parse --token value")
			} else if x != "" {
				connection.Credentials = append(connection.Credentials, &vault.Credential{
					Type:     vault.CredentialType_password,
					Password: x,
				})
			}
		default:
			log.Fatal().Msg("asset type must be set for GitHub")
		}
	case providers.ProviderType_GITLAB:
		connection.Backend = providerType

		if x, err := cmd.Flags().GetString("group"); err != nil {
			log.Fatal().Err(err).Msg("cannot parse --group value")
		} else if x != "" {
			connection.Options["group"] = x
		}

		if x, err := cmd.Flags().GetString("token"); err != nil {
			log.Fatal().Err(err).Msg("cannot parse --token value")
		} else if x != "" {
			connection.Credentials = append(connection.Credentials, &vault.Credential{
				Type:     vault.CredentialType_password,
				User:     username,
				Password: password,
			})
		}
	case providers.ProviderType_MS365:
		connection.Backend = providerType

		// data report is deprecated in v6 and should be removed asap
		if x, err := cmd.Flags().GetString("datareport"); err != nil {
			log.Fatal().Err(err).Msg("cannot parse --datareport values")
		} else if x != "" {
			connection.Options["mondoo-ms365-datareport"] = x
		}

		if tenantId, err := cmd.Flags().GetString("tenant-id"); err == nil {
			connection.Options["tenantId"] = tenantId
		}
		if clientID, err := cmd.Flags().GetString("client-id"); err == nil {
			connection.Options["clientId"] = clientID
		}

		if clientSecret, err := cmd.Flags().GetString("client-secret"); err != nil {
			log.Fatal().Err(err).Msg("cannot parse --client-secret value")
		} else if clientSecret != "" {
			connection.Credentials = append(connection.Credentials, &vault.Credential{
				Type:     vault.CredentialType_password,
				User:     username,
				Password: password,
			})
		}

		if certificatepPath, err := cmd.Flags().GetString("certificate-path"); err != nil {
			log.Fatal().Err(err).Msg("cannot parse --certificate-path value")
		} else if certificatepPath != "" {
			connection.Credentials = append(connection.Credentials, &vault.Credential{
				Type:           vault.CredentialType_private_key,
				User:           username,
				PrivateKeyPath: certificatepPath,
			})
		}

		if certificateSecret, err := cmd.Flags().GetString("certificate-secret"); err != nil {
			log.Fatal().Err(err).Msg("cannot parse --certificate-secret value")
		} else if certificateSecret != "" {
			connection.Credentials = append(connection.Credentials, &vault.Credential{
				Type:     vault.CredentialType_password,
				User:     username,
				Password: password,
			})
		}
	case providers.ProviderType_HOST:
		connection.Backend = providerType
		target, err := parseTarget(args[0])
		if err != nil {
			log.Error().Err(err).Msg("cannot parse target")
		}
		connection.Host = target.Hostname
		connection.Port = target.Port
		connection.Path = target.Path
		username = target.Username

		connection.Credentials = append(connection.Credentials, &vault.Credential{
			Type:     vault.CredentialType_password,
			User:     username,
			Password: password,
		})
	case providers.ProviderType_ARISTAEOS:
		connection.Backend = providerType
		target, err := parseTarget(args[0])
		if err != nil {
			log.Error().Err(err).Msg("cannot parse target")
		}
		connection.Host = target.Hostname
		connection.Port = target.Port
		connection.Path = target.Path
		username = target.Username

		connection.Credentials = append(connection.Credentials, &vault.Credential{
			Type:     vault.CredentialType_password,
			User:     username,
			Password: password,
		})
	}

	// if username was set but not credentials
	if username != "" && len(connection.Credentials) == 0 {
		connection.Credentials = append(connection.Credentials, &vault.Credential{
			Type: vault.CredentialType_password,
			User: username,
		})
	}

	parsedAsset.Connections = []*providers.Config{connection}
	return parsedAsset
}
