package k8s

import (
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/motor/asset"
	"go.mondoo.com/cnquery/motor/providers"
	"go.mondoo.com/cnquery/motor/providers/k8s"

	batchv1 "k8s.io/api/batch/v1"
)

// ListCronJobs list all cronjobs in the cluster.
func ListCronJobs(p k8s.KubernetesProvider, connection *providers.Config, clusterIdentifier string, namespaceFilter []string, od *k8s.PlatformIdOwnershipDirectory) ([]*asset.Asset, error) {
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
		od.Add(&cronJob)
		asset, err := createAssetFromObject(&cronJob, p.Runtime(), connection, clusterIdentifier)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create asset from cronjob")
		}

		log.Debug().Str("name", cronJob.Name).Str("connection", asset.Connections[0].Host).Msg("resolved CronJob")

		assets = append(assets, asset)
	}

	return assets, nil
}
