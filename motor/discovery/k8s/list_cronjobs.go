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
func ListCronJobs(
	p k8s.KubernetesProvider,
	connection *providers.Config,
	clusterIdentifier string,
	namespaceFilter []string,
	resFilter map[string][]K8sResourceIdentifier,
	od *k8s.PlatformIdOwnershipDirectory,
) ([]*asset.Asset, error) {
	cronJobs := []batchv1.CronJob{}

	if len(resFilter) > 0 {
		// If there is a resources filter we should only retrieve the cronjobs that are in the filter.
		if len(resFilter["cronjob"]) == 0 {
			return []*asset.Asset{}, nil
		}

		for _, res := range resFilter["cronjob"] {
			cj, err := p.CronJob(res.Namespace, res.Name)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to get cronjob %s/%s", res.Namespace, res.Name)
			}

			cronJobs = append(cronJobs, *cj)
		}
	} else {
		namespaces, err := p.Namespaces()
		if err != nil {
			return nil, errors.Wrap(err, "could not list kubernetes namespaces")
		}

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
