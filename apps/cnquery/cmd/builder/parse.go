package builder

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"go.mondoo.com/cnquery/motor/asset"
	"go.mondoo.com/cnquery/motor/motorid/awsec2"
	"go.mondoo.com/cnquery/motor/providers"
	"go.mondoo.com/cnquery/motor/providers/awsec2ebs"
	"go.mondoo.com/cnquery/motor/providers/os/snapshot"
	"go.mondoo.com/cnquery/motor/vault"
)

type target struct {
	Hostname string
	Username string
	Path     string
	Port     int32
}

// match [leading_protocol:]//[user@]EVERYTHING_ELSE
var ipv6Regexp = regexp.MustCompile(`(.*//)(.*@)?(.*)`)

func addIPv6Brackets(uri string) string {
	if strings.ContainsAny(uri, "[]") {
		// already has surrounding []s
		return uri
	}

	matched := ipv6Regexp.FindStringSubmatch(uri)

	if matched == nil {
		return uri
	}

	host := matched[3]

	// if there is just one : (or less) then we're not dealing with ipv6
	if strings.Count(host, ":") <= 1 {
		return uri
	}

	// at this point we assume we're dealing with ipv6 w/o []s
	return fmt.Sprintf("%s%s[%s]", matched[1], matched[2], host)
}

// parseTarget parses the specified target, which may be specified as either:
// - [user@]hostname or
// - a URI of the form ssh://[user@]hostname[:port]
func parseTarget(uri string) (target, error) {
	var target target
	if !strings.Contains(uri, "://") {
		uri = "//" + uri
	}

	uri = addIPv6Brackets(uri)

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
	sudo, _ := cmd.Flags().GetBool("sudo")
	assetName, _ := cmd.Flags().GetString("asset-name")
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
		Name:        assetName,
		Options:     map[string]string{},
		Labels:      labels,
		Annotations: annotations,
		Connections: []*providers.Config{},
	}

	connection := &providers.Config{
		Backend: providers.ProviderType_UNKNOWN,
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
			connection.Sudo = &providers.Sudo{
				Active: sudo,
			}
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

		connection.Sudo = &providers.Sudo{
			Active: sudo,
		}

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

		if len(connection.Credentials) == 0 {
			log.Warn().Msg("no identity file or password are provided for ssh authentication, use either --identity-file, --ask-pass or --password, fall back to ssh agent")
			connection.Credentials = append(connection.Credentials, &vault.Credential{
				Type: vault.CredentialType_ssh_agent,
				User: username,
			})
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
		disableInMemoryCache := false
		disableInMemoryCache, err := cmd.Flags().GetBool("disable-cache")
		if err != nil {
			log.Error().Err(err).Msg("cannot parse --disable-cache value")
		}
		connection.Options["disable-cache"] = strconv.FormatBool(disableInMemoryCache)
		connection.Backend = providerType
		connection.Host = args[0]
	case providers.ProviderType_TAR:
		connection.Backend = providerType
		if filepath != "" {
			if _, err := os.Stat(filepath); os.IsNotExist(err) {
				log.Fatal().Str("file", filepath).Msg("Could not find the container tar file. Please specify the correct path.")
			}
			connection.Options["path"] = filepath
		}
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

		// If the command did not specify an explicit target, there will be 0 arguments.
		if len(args) > 0 {
			connection.Host = args[0]
		}
	case providers.ProviderType_K8S:
		connection.Backend = providerType

		// do pre-processing of piped manifest file
		// TODO: consider using an afero.NewMemMapFs() in-memory file system instead of base64 encoding
		if filepath == "-" {
			data, err := io.ReadAll(os.Stdin)
			if err != nil {
				log.Fatal().Err(err).Msg("cannot read Kubernetes manifest from stdin")
			}
			connection.Options["manifest-content"] = base64.StdEncoding.EncodeToString(data)
		} else if filepath != "" {
			if _, err := os.Stat(filepath); os.IsNotExist(err) {
				log.Fatal().Str("file", filepath).Msg("Could not find the Kubernetes manifest file. Please specify the correct path.")
			}
			connection.Options["path"] = filepath
		}

		if excludeNamespaces, err := cmd.Flags().GetString("namespaces-exclude"); err != nil {
			log.Fatal().Err(err).Msg("cannot parse --namespaces-exclude values")
		} else if excludeNamespaces != "" {
			connection.Options["namespaces-exclude"] = excludeNamespaces
		}

		if includeNamespaces, err := cmd.Flags().GetString("namespaces"); err != nil {
			log.Fatal().Err(err).Msg("cannot parse --namespaces values")
		} else if includeNamespaces != "" {
			connection.Options["namespaces"] = includeNamespaces
		}

		if targetContext, err := cmd.Flags().GetString("context"); err != nil {
			log.Fatal().Err(err).Msg("cannot parse --context values")
		} else if targetContext != "" {
			connection.Options["context"] = targetContext
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
		noSetup := "false"
		if connection.Options[snapshot.NoSetup] != "" {
			noSetup = connection.Options[snapshot.NoSetup]
		}
		overrideRegion := connection.Options["region"]
		assembleAwsEc2EbsConnectionUrl := func(flagAsset *asset.Asset, arg string, targetType string) string {
			instanceInfo, err := awsec2ebs.GetRawInstanceInfo(flagAsset.Options["profile"])
			if err != nil {
				log.Fatal().Err(err).Msg("cannot detect instance info")
			}
			targetRegion := instanceInfo.Region
			if overrideRegion != "" {
				targetRegion = overrideRegion
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
			"account":        target.Account,
			"region":         target.Region,
			"id":             target.Id,
			"type":           target.Type,
			snapshot.NoSetup: noSetup,
		}
	case providers.ProviderType_AWS_SSM_RUN_COMMAND:
		target, err := parseTarget(args[0])
		if err != nil {
			log.Error().Err(err).Msg("cannot parse target")
		}
		connection.Host = target.Hostname
		username = target.Username
		connection.Backend = providers.ProviderType_SSH // TODO: allow the usage the provider type here
		connection.Credentials = append(connection.Credentials, &vault.Credential{
			Type: vault.CredentialType_aws_ec2_ssm_session,
			User: username,
		})

	case providers.ProviderType_AZURE:
		connection.Backend = providerType
		subscription, err := cmd.Flags().GetString("subscription")
		tenantid, _ := cmd.Flags().GetString("tenant-id")
		clientid, _ := cmd.Flags().GetString("client-id")
		clientSecret, _ := cmd.Flags().GetString("client-secret")
		certificatePath, _ := cmd.Flags().GetString("certificate-path")
		certificateSecret, _ := cmd.Flags().GetString("certificate-secret")
		subsToInclude, _ := cmd.Flags().GetString("subscriptions")
		subsToExclude, _ := cmd.Flags().GetString("subscriptions-exclude")

		if clientid == "" && (clientSecret == "" || certificatePath == "") {
			if err != nil {
				log.Fatal().Err(err).Msg("cannot parse --subscription value")
			}
		}
		if subscription != "" {
			connection.Options["subscription-id"] = subscription
		}
		if tenantid != "" {
			connection.Options["tenant-id"] = tenantid
		}
		if clientid != "" {
			connection.Options["client-id"] = clientid
		}
		if subsToExclude != "" && subsToInclude != "" {
			log.Fatal().Msg("cannot provide both --subscriptions and --subscriptions-exclude, provide only one of them")
		}
		if subsToExclude != "" {
			connection.Options["subscriptions-exclude"] = subsToExclude
		}
		if subsToInclude != "" {
			connection.Options["subscriptions"] = subsToInclude
		}

		if clientSecret != "" {
			connection.Credentials = append(connection.Credentials, &vault.Credential{
				Type:     vault.CredentialType_password,
				Password: clientSecret,
				User:     clientid,
			})
		}
		if certificatePath != "" {
			connection.Credentials = append(connection.Credentials, &vault.Credential{
				Type:           vault.CredentialType_pkcs12,
				PrivateKeyPath: certificatePath,
				Password:       certificateSecret,
			})
		}
	case providers.ProviderType_GCP:
		connection.Backend = providerType

		envVars := []string{
			"GOOGLE_APPLICATION_CREDENTIALS",
			"GOOGLE_CREDENTIALS",
			"GOOGLE_CLOUD_KEYFILE_JSON",
			"GCLOUD_KEYFILE_JSON",
		}
		serviceAccount := getGoogleCreds(cmd, envVars...)
		if serviceAccount != nil {
			connection.Credentials = append(connection.Credentials, &vault.Credential{
				Type:   vault.CredentialType_json,
				Secret: serviceAccount,
			})
		}
		switch assetType {
		case DefaultAssetType:
			// deprecated, remove in v9
			if project, err := cmd.Flags().GetString("project-id"); err != nil {
				log.Fatal().Err(err).Msg("cannot parse --project-id value")
			} else if project != "" {
				connection.Options["project-id"] = project
			}

			// deprecated, remove in v9
			if organization, err := cmd.Flags().GetString("organization-id"); err != nil {
				log.Fatal().Err(err).Msg("cannot parse --organization value")
			} else if organization != "" {
				connection.Options["organization-id"] = organization
			}
		case GcpOrganizationAssetType:
			connection.Options["organization-id"] = args[0]
		case GcpProjectAssetType:
			connection.Options["project-id"] = args[0]
		case GcpFolderAssetType:
			connection.Options["folder-id"] = args[0]
		}
	case providers.ProviderType_GCP_COMPUTE_INSTANCE_SNAPSHOT:
		connection.Backend = providerType
		connection.Options = map[string]string{}

		switch assetType {
		case GcpComputeInstanceAssetType:
			connection.Options["type"] = "instance"
			connection.Options["instance-name"] = args[0]

			if createNewSnapshot, err := cmd.Flags().GetBool("create-snapshot"); err != nil {
				log.Fatal().Err(err).Msg("cannot parse --create-snapshot")
			} else if createNewSnapshot {
				connection.Options["create-snapshot"] = "true"
			}
		case GcpComputeInstanceSnapshotAssetType:
			connection.Options["type"] = "snapshot"
			connection.Options["snapshot-name"] = args[0]
		}

		if projectId, err := cmd.Flags().GetString("project-id"); err != nil {
			log.Fatal().Err(err).Msg("cannot parse --project-id value")
		} else if projectId != "" {
			connection.Options["project-id"] = projectId
		}

		if zone, err := cmd.Flags().GetString("zone"); err != nil {
			log.Fatal().Err(err).Msg("cannot parse --zone value")
		} else if zone != "" {
			connection.Options["zone"] = zone
		}

	case providers.ProviderType_OCI:
		connection.Backend = providerType
		tenancy, _ := cmd.Flags().GetString("tenancy")
		fingerprint, _ := cmd.Flags().GetString("fingerprint")
		user, _ := cmd.Flags().GetString("user")
		keyPath, _ := cmd.Flags().GetString("key-path")
		keyPassphrase, _ := cmd.Flags().GetString("key-passphrase")
		region, _ := cmd.Flags().GetString("region")

		if tenancy != "" {
			connection.Options["tenancy"] = tenancy
		}
		if fingerprint != "" {
			connection.Options["fingerprint"] = fingerprint
		}
		if region != "" {
			connection.Options["region"] = region
		}
		if user != "" {
			connection.Options["user"] = user
		}

		if keyPath != "" {
			connection.Credentials = append(connection.Credentials, &vault.Credential{
				Type:           vault.CredentialType_private_key,
				PrivateKeyPath: keyPath,
				Password:       keyPassphrase,
			})
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
				Password: x,
			})
		}
	case providers.ProviderType_MS365:
		connection.Backend = providerType

		// FIXME: DEPRECATED in v6 vv
		// remove datareport from the list of supported options
		if x, err := cmd.Flags().GetString("datareport"); err != nil {
			log.Fatal().Err(err).Msg("cannot parse --datareport values")
		} else if x != "" {
			connection.Options["mondoo-ms365-datareport"] = x
		}
		// ^^

		tenantId, err := cmd.Flags().GetString("tenant-id")
		if err != nil {
			log.Fatal().Err(err).Msg("cannot parse --tenant-id value")
		}
		clientId, err := cmd.Flags().GetString("client-id")
		if err != nil {
			log.Fatal().Err(err).Msg("cannot parse --client-id value")
		}
		connection.Options["client-id"] = clientId
		connection.Options["tenant-id"] = tenantId

		if clientSecret, err := cmd.Flags().GetString("client-secret"); err != nil {
			log.Fatal().Err(err).Msg("cannot parse --client-secret value")
		} else if clientSecret != "" {
			connection.Credentials = append(connection.Credentials, &vault.Credential{
				Type:     vault.CredentialType_password,
				User:     username,
				Password: clientSecret,
			})
		}

		if certificatepPath, err := cmd.Flags().GetString("certificate-path"); err != nil {
			log.Fatal().Err(err).Msg("cannot parse --certificate-path value")
		} else if certificatepPath != "" {
			if password == "" {
				certificateSecret, err := cmd.Flags().GetString("certificate-secret")
				if err != nil {
					log.Fatal().Err(err).Msg("cannot parse --certificate-secret value")
				} else {
					password = certificateSecret
				}
			}
			connection.Credentials = append(connection.Credentials, &vault.Credential{
				Type:           vault.CredentialType_pkcs12,
				User:           username,
				PrivateKeyPath: certificatepPath,
				Password:       password,
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
	case providers.ProviderType_OKTA:
		connection.Backend = providerType

		if organization, err := cmd.Flags().GetString("organization"); err != nil {
			log.Fatal().Err(err).Msg("cannot parse --organization value")
		} else if organization != "" {
			connection.Options["organization"] = organization
		}

		// the env var has precedence over --token
		token := os.Getenv("OKTA_CLIENT_TOKEN")
		if token == "" {
			if token, err = cmd.Flags().GetString("token"); err != nil {
				log.Fatal().Err(err).Msg("cannot parse --token value")
			}
		}

		if token != "" {
			connection.Credentials = append(connection.Credentials, &vault.Credential{
				Type:     vault.CredentialType_password,
				Password: token,
			})
		}
	case providers.ProviderType_GOOGLE_WORKSPACE:
		connection.Backend = providerType

		envVars := []string{
			"GOOGLE_APPLICATION_CREDENTIALS",
			"GOOGLEWORKSPACE_CREDENTIALS",
			"GOOGLEWORKSPACE_CLOUD_KEYFILE_JSON",
			"GOOGLE_CREDENTIALS",
		}
		serviceAccount := getGoogleCreds(cmd, envVars...)
		if serviceAccount != nil {
			connection.Credentials = append(connection.Credentials, &vault.Credential{
				Type:   vault.CredentialType_json,
				Secret: serviceAccount,
			})
		}
		if customerID, err := cmd.Flags().GetString("customer-id"); err != nil {
			log.Fatal().Err(err).Msg("cannot parse --customer-id value")
		} else if customerID != "" {
			connection.Options["customer-id"] = customerID
		}

		if impersonatedUserEmail, err := cmd.Flags().GetString("impersonated-user-email"); err != nil {
			log.Fatal().Err(err).Msg("cannot parse --impersonated-user-email value")
		} else if impersonatedUserEmail != "" {
			connection.Options["impersonated-user-email"] = impersonatedUserEmail
		}
	case providers.ProviderType_SLACK:
		connection.Backend = providerType

		// the env var has precedence over --token
		token := os.Getenv("SLACK_TOKEN")
		if token == "" {
			if token, err = cmd.Flags().GetString("token"); err != nil {
				log.Fatal().Err(err).Msg("cannot parse --token value")
			}
		}
		if token != "" {
			connection.Credentials = append(connection.Credentials, &vault.Credential{
				Type:     vault.CredentialType_password,
				Password: token,
			})
		}
	case providers.ProviderType_VCD:
		connection.Backend = providerType

		cred := &vault.Credential{
			Type:     vault.CredentialType_password,
			Password: password,
		}

		if x, err := cmd.Flags().GetString("user"); err != nil {
			log.Fatal().Err(err).Msg("cannot parse --user value")
		} else if x != "" {
			cred.User = x
		}

		if x, err := cmd.Flags().GetString("organization"); err != nil {
			log.Fatal().Err(err).Msg("cannot parse --organization value")
		} else if x != "" {
			connection.Options["organization"] = x
		}

		if x, err := cmd.Flags().GetString("host"); err != nil {
			log.Fatal().Err(err).Msg("cannot parse --host value")
		} else if x != "" {
			connection.Host = x
		}

		connection.Credentials = append(connection.Credentials, cred)
	case providers.ProviderType_FS:
		connection.Backend = providerType
		connection.Options["path"] = filepath
	case providers.ProviderType_OPCUA:
		connection.Backend = providerType

		cred := &vault.Credential{
			Type:     vault.CredentialType_password,
			Password: password,
		}

		if x, err := cmd.Flags().GetString("endpoint"); err != nil {
			log.Fatal().Err(err).Msg("cannot parse --endpoint value")
		} else if x != "" {
			connection.Options["endpoint"] = x
		}

		connection.Credentials = append(connection.Credentials, cred)
	}

	parsedAsset.Connections = []*providers.Config{connection}
	return parsedAsset
}

// returns only the env vars that have a set value
func readEnvs(envs ...string) []string {
	vals := []string{}
	for i := range envs {
		val := os.Getenv(envs[i])
		if val != "" {
			vals = append(vals, val)
		}
	}

	return vals
}

// to be used by gcp/googleworkspace cmds, fetches the creds from either the env vars provided or from a flag in the provided cmd
func getGoogleCreds(cmd *cobra.Command, envs ...string) []byte {
	var credsPaths []string
	// env vars have precedence over the --credentials-path arg
	credsPaths = readEnvs(envs...)

	if credPath, err := cmd.Flags().GetString("credentials-path"); err != nil {
		log.Fatal().Err(err).Msg("cannot parse --credentials-path value")
	} else if credPath != "" {
		credsPaths = append(credsPaths, credPath)
	}

	for i := range credsPaths {
		path := credsPaths[i]

		serviceAccount, err := os.ReadFile(path)
		if err == nil {
			return serviceAccount
		}
	}
	return nil
}
