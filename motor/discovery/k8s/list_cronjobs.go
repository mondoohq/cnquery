package k8s

import (
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.io/mondoo/motor/asset"
	"go.mondoo.io/mondoo/motor/providers"
	"go.mondoo.io/mondoo/motor/providers/k8s"

	batchv1 "k8s.io/api/batch/v1"
)

// ListCronJobs list all cronjobs in the cluster.
func ListCronJobs(p k8s.KubernetesProvider, connection *providers.Config, clusterIdentifier string, namespaceFilter []string) ([]*asset.Asset, error) {
	namespaces, err := p.Namespaces()
	if err != nil {
		return nil, errors.Wrap(err, "could not list kubernetes namespaces")
	}

	cronJobs := []batchv1.CronJob{}
	for i := range namespaces {
		namespace := namespaces[i]
		if !isIncluded(namespace.Name, namespaceFilter) {
			log.Info().Str("namespace", namespace.Name).Strs("filter", namespaceFilter).Msg("namespace not included")
			continue
		}

		cronJobsPerNamespace, err := p.CronJobs(namespace)
		if err != nil {
			return nil, errors.Wrap(err, "failed to list CronJobs")
		}

		cronJobs = append(cronJobs, cronJobsPerNamespace...)
	}

	assets := []*asset.Asset{}
	for i := range cronJobs {
		cronJob := cronJobs[i]
		platformData := p.PlatformInfo()
		platformData.Version = cronJob.APIVersion
		platformData.Build = cronJob.ResourceVersion
		platformData.Labels = map[string]string{
			"namespace": cronJob.Namespace,
			"uid":       string(cronJob.UID),
		}
		platformData.Kind = providers.Kind_KIND_K8S_OBJECT
		asset := &asset.Asset{
			PlatformIds: []string{k8s.NewPlatformWorkloadId(clusterIdentifier, "cronjobs", cronJob.Namespace, cronJob.Name)},
			Name:        cronJob.Namespace + "/" + cronJob.Name,
			Platform:    platformData,
			Connections: []*providers.Config{connection},
			State:       asset.State_STATE_ONLINE,
			Labels:      cronJob.Labels,
		}
		if asset.Labels == nil {
			asset.Labels = map[string]string{
				"namespace": cronJob.Namespace,
			}
		} else {
			asset.Labels["namespace"] = cronJob.Namespace
		}
		log.Debug().Str("name", cronJob.Name).Str("connection", asset.Connections[0].Host).Msg("resolved CronJob")

		assets = append(assets, asset)
	}

	return assets, nil
}
