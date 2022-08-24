package k8s

import (
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/motor/asset"
	"go.mondoo.com/cnquery/motor/providers"
	"go.mondoo.com/cnquery/motor/providers/k8s"

	batchv1 "k8s.io/api/batch/v1"
)

// ListJobs list all jobs in the cluster.
func ListJobs(p k8s.KubernetesProvider, connection *providers.Config, clusterIdentifier string, namespaceFilter []string) ([]*asset.Asset, error) {
	namespaces, err := p.Namespaces()
	if err != nil {
		return nil, errors.Wrap(err, "could not list kubernetes namespaces")
	}

	jobs := []batchv1.Job{}
	for i := range namespaces {
		namespace := namespaces[i]
		if !isIncluded(namespace.Name, namespaceFilter) {
			log.Info().Str("namespace", namespace.Name).Strs("filter", namespaceFilter).Msg("namespace not included")
			continue
		}

		jobsPerNamespace, err := p.Jobs(namespace)
		if err != nil {
			return nil, errors.Wrap(err, "failed to list Jobs")
		}

		jobs = append(jobs, jobsPerNamespace...)
	}

	assets := []*asset.Asset{}
	for i := range jobs {
		job := jobs[i]
		asset, err := createAssetFromObject(&job, p.Runtime(), connection, clusterIdentifier)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create asset from job")
		}

		log.Debug().Str("name", job.Name).Str("connection", asset.Connections[0].Host).Msg("resolved Job")

		assets = append(assets, asset)
	}

	return assets, nil
}
