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
func ListJobs(
	p k8s.KubernetesProvider,
	connection *providers.Config,
	clusterIdentifier string,
	namespaceFilter []string,
	resFilter map[string][]K8sResourceIdentifier,
	od *k8s.PlatformIdOwnershipDirectory,
) ([]*asset.Asset, error) {
	jobs := []batchv1.Job{}

	if len(resFilter) > 0 {
		// If there is a resources filter we should only retrieve the jobs that are in the filter.
		if len(resFilter["job"]) == 0 {
			return []*asset.Asset{}, nil
		}

		for _, res := range resFilter["job"] {
			j, err := p.Job(res.Namespace, res.Name)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to get job %s/%s", res.Namespace, res.Name)
			}

			jobs = append(jobs, *j)
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

			jobsPerNamespace, err := p.Jobs(namespace)
			if err != nil {
				return nil, errors.Wrap(err, "failed to list Jobs")
			}

			jobs = append(jobs, jobsPerNamespace...)
		}
	}

	assets := []*asset.Asset{}
	for i := range jobs {
		job := jobs[i]
		if od != nil {
			od.Add(&job)
		}
		asset, err := createAssetFromObject(&job, p.Runtime(), connection, clusterIdentifier)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create asset from job")
		}

		log.Debug().Str("name", job.Name).Str("connection", asset.Connections[0].Host).Msg("resolved Job")

		assets = append(assets, asset)
	}

	return assets, nil
}
