package k8s

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mondoo.com/cnquery/motor/asset"
	"go.mondoo.com/cnquery/motor/providers"
	"go.mondoo.com/cnquery/motor/providers/k8s"
	"go.mondoo.com/cnquery/motor/providers/k8s/resources"
)

func TestManifestResolver(t *testing.T) {
	resolver := &Resolver{}
	manifestFile := "../../providers/k8s/resources/testdata/pod.yaml"

	ctx := resources.SetDiscoveryCache(context.Background(), resources.NewDiscoveryCache())

	assetList, err := resolver.Resolve(ctx, &asset.Asset{}, &providers.Config{
		PlatformId: "//platform/k8s/uid/123/namespace/default/pods/name/mondoo",
		Backend:    providers.ProviderType_K8S,
		Options: map[string]string{
			"path": manifestFile,
		},
		Discover: &providers.Discovery{
			Targets: []string{"all"},
		},
	}, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, 4, len(assetList))
	assert.Equal(t, assetList[1].Platform.Name, "k8s-pod")
	assert.Contains(t, assetList[1].Platform.Family, "k8s-workload")
	assert.Contains(t, assetList[1].Platform.Family, "k8s")
	assert.Equal(t, assetList[3].Platform.Runtime, "docker-registry")
}

func TestAdmissionReviewResolver(t *testing.T) {
	resolver := &Resolver{}

	ctx := resources.SetDiscoveryCache(context.Background(), resources.NewDiscoveryCache())

	assetList, err := resolver.Resolve(ctx, &asset.Asset{}, &providers.Config{
		Backend: providers.ProviderType_K8S,
		Options: map[string]string{
			k8s.OPTION_ADMISSION: "ewogICAiYXBpVmVyc2lvbiI6ICJhZG1pc3Npb24uazhzLmlvL3YxIiwKICAgImtpbmQiOiAiQWRtaXNzaW9uUmV2aWV3IiwKICAgICAgInJlcXVlc3QiOnsKICAgICAgICAgInVpZCI6IjdmMTg3YzhlLThiM2YtNGEyNi1hZDkyLWEwNWRkZTcwOWIxZSIsCiAgICAgICAgICJraW5kIjp7CiAgICAgICAgICAgICJncm91cCI6IiIsCiAgICAgICAgICAgICJ2ZXJzaW9uIjoidjEiLAogICAgICAgICAgICAia2luZCI6IlBvZCIKICAgICAgICAgfSwKICAgICAgICAgInJlc291cmNlIjp7CiAgICAgICAgICAgICJncm91cCI6IiIsCiAgICAgICAgICAgICJ2ZXJzaW9uIjoidjEiLAogICAgICAgICAgICAicmVzb3VyY2UiOiJwb2RzIgogICAgICAgICB9LAogICAgICAgICAicmVxdWVzdEtpbmQiOnsKICAgICAgICAgICAgImdyb3VwIjoiIiwKICAgICAgICAgICAgInZlcnNpb24iOiJ2MSIsCiAgICAgICAgICAgICJraW5kIjoiUG9kIgogICAgICAgICB9LAogICAgICAgICAicmVxdWVzdFJlc291cmNlIjp7CiAgICAgICAgICAgICJncm91cCI6IiIsCiAgICAgICAgICAgICJ2ZXJzaW9uIjoidjEiLAogICAgICAgICAgICAicmVzb3VyY2UiOiJwb2RzIgogICAgICAgICB9LAogICAgICAgICAibmFtZSI6InRlc3QtZGVwLTVmNjU2OTdmOGQtZnhjbHIiLAogICAgICAgICAibmFtZXNwYWNlIjoiZGVmYXVsdCIsCiAgICAgICAgICJvcGVyYXRpb24iOiJDUkVBVEUiLAogICAgICAgICAidXNlckluZm8iOnsKICAgICAgICAgICAgInVzZXJuYW1lIjoic3lzdGVtOnNlcnZpY2VhY2NvdW50Omt1YmUtc3lzdGVtOnJlcGxpY2FzZXQtY29udHJvbGxlciIsCiAgICAgICAgICAgICJ1aWQiOiI0Y2RmNTE3My04OGM3LTQyY2YtYmIzZi0wZTg3M2U2ZTI2NTUiLAogICAgICAgICAgICAiZ3JvdXBzIjpbCiAgICAgICAgICAgICAgICJzeXN0ZW06c2VydmljZWFjY291bnRzIiwKICAgICAgICAgICAgICAgInN5c3RlbTpzZXJ2aWNlYWNjb3VudHM6a3ViZS1zeXN0ZW0iLAogICAgICAgICAgICAgICAic3lzdGVtOmF1dGhlbnRpY2F0ZWQiCiAgICAgICAgICAgIF0KICAgICAgICAgfSwKICAgICAgICAgIm9iamVjdCI6ewogICAgICAgICAgICAia2luZCI6IlBvZCIsCiAgICAgICAgICAgICJhcGlWZXJzaW9uIjoidjEiLAogICAgICAgICAgICAibWV0YWRhdGEiOnsKICAgICAgICAgICAgICAgIm5hbWUiOiJ0ZXN0LWRlcC01ZjY1Njk3ZjhkLWZ4Y2xyIiwKICAgICAgICAgICAgICAgImdlbmVyYXRlTmFtZSI6InRlc3QtZGVwLTVmNjU2OTdmOGQtIiwKICAgICAgICAgICAgICAgIm5hbWVzcGFjZSI6ImRlZmF1bHQiLAogICAgICAgICAgICAgICAidWlkIjoiOWRkNjQ4MDEtZGVmYy00MTNiLWE1YjQtOGRmY2I0MzUwMjgwIiwKICAgICAgICAgICAgICAgImNyZWF0aW9uVGltZXN0YW1wIjoiMjAyMi0wOS0xOVQxNToxMjowNFoiLAogICAgICAgICAgICAgICAibGFiZWxzIjp7CiAgICAgICAgICAgICAgICAgICJhcHAiOiJ0ZXN0LWRlcCIsCiAgICAgICAgICAgICAgICAgICJwb2QtdGVtcGxhdGUtaGFzaCI6IjVmNjU2OTdmOGQiCiAgICAgICAgICAgICAgIH0sCiAgICAgICAgICAgICAgICJvd25lclJlZmVyZW5jZXMiOlsKICAgICAgICAgICAgICAgICAgewogICAgICAgICAgICAgICAgICAgICAiYXBpVmVyc2lvbiI6ImFwcHMvdjEiLAogICAgICAgICAgICAgICAgICAgICAia2luZCI6IlJlcGxpY2FTZXQiLAogICAgICAgICAgICAgICAgICAgICAibmFtZSI6InRlc3QtZGVwLTVmNjU2OTdmOGQiLAogICAgICAgICAgICAgICAgICAgICAidWlkIjoiNTI5MzhiNDAtODZhMy00YTRkLTk2ZDMtY2NjMzI5YTFiNjI2IiwKICAgICAgICAgICAgICAgICAgICAgImNvbnRyb2xsZXIiOnRydWUsCiAgICAgICAgICAgICAgICAgICAgICJibG9ja093bmVyRGVsZXRpb24iOnRydWUKICAgICAgICAgICAgICAgICAgfQogICAgICAgICAgICAgICBdLAogICAgICAgICAgICAgICAibWFuYWdlZEZpZWxkcyI6WwogICAgICAgICAgICAgICAgICB7CiAgICAgICAgICAgICAgICAgICAgICJtYW5hZ2VyIjoia3ViZS1jb250cm9sbGVyLW1hbmFnZXIiLAogICAgICAgICAgICAgICAgICAgICAib3BlcmF0aW9uIjoiVXBkYXRlIiwKICAgICAgICAgICAgICAgICAgICAgImFwaVZlcnNpb24iOiJ2MSIsCiAgICAgICAgICAgICAgICAgICAgICJ0aW1lIjoiMjAyMi0wOS0xOVQxNToxMjowNFoiLAogICAgICAgICAgICAgICAgICAgICAiZmllbGRzVHlwZSI6IkZpZWxkc1YxIiwKICAgICAgICAgICAgICAgICAgICAgImZpZWxkc1YxIjp7CiAgICAgICAgICAgICAgICAgICAgICAgICJmOm1ldGFkYXRhIjp7CiAgICAgICAgICAgICAgICAgICAgICAgICAgICJmOmdlbmVyYXRlTmFtZSI6ewogICAgICAgICAgICAgICAgICAgICAgICAgICAgICAKICAgICAgICAgICAgICAgICAgICAgICAgICAgfSwKICAgICAgICAgICAgICAgICAgICAgICAgICAgImY6bGFiZWxzIjp7CiAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICIuIjp7CiAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgIAogICAgICAgICAgICAgICAgICAgICAgICAgICAgICB9LAogICAgICAgICAgICAgICAgICAgICAgICAgICAgICAiZjphcHAiOnsKICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgCiAgICAgICAgICAgICAgICAgICAgICAgICAgICAgIH0sCiAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICJmOnBvZC10ZW1wbGF0ZS1oYXNoIjp7CiAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgIAogICAgICAgICAgICAgICAgICAgICAgICAgICAgICB9CiAgICAgICAgICAgICAgICAgICAgICAgICAgIH0sCiAgICAgICAgICAgICAgICAgICAgICAgICAgICJmOm93bmVyUmVmZXJlbmNlcyI6ewogICAgICAgICAgICAgICAgICAgICAgICAgICAgICAiLiI6ewogICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAKICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgfSwKICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgIms6e1widWlkXCI6XCI1MjkzOGI0MC04NmEzLTRhNGQtOTZkMy1jY2MzMjlhMWI2MjZcIn0iOnsKICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgCiAgICAgICAgICAgICAgICAgICAgICAgICAgICAgIH0KICAgICAgICAgICAgICAgICAgICAgICAgICAgfQogICAgICAgICAgICAgICAgICAgICAgICB9LAogICAgICAgICAgICAgICAgICAgICAgICAiZjpzcGVjIjp7CiAgICAgICAgICAgICAgICAgICAgICAgICAgICJmOmNvbnRhaW5lcnMiOnsKICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgIms6e1wibmFtZVwiOlwicmVkaXNcIn0iOnsKICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgIi4iOnsKICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgCiAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgIH0sCiAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICJmOmltYWdlIjp7CiAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgIAogICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICB9LAogICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAiZjppbWFnZVB1bGxQb2xpY3kiOnsKICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgCiAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgIH0sCiAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICJmOm5hbWUiOnsKICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgCiAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgIH0sCiAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICJmOnJlc291cmNlcyI6ewogICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAKICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgfSwKICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgImY6dGVybWluYXRpb25NZXNzYWdlUGF0aCI6ewogICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAKICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgfSwKICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgImY6dGVybWluYXRpb25NZXNzYWdlUG9saWN5Ijp7CiAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgIAogICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICB9CiAgICAgICAgICAgICAgICAgICAgICAgICAgICAgIH0KICAgICAgICAgICAgICAgICAgICAgICAgICAgfSwKICAgICAgICAgICAgICAgICAgICAgICAgICAgImY6ZG5zUG9saWN5Ijp7CiAgICAgICAgICAgICAgICAgICAgICAgICAgICAgIAogICAgICAgICAgICAgICAgICAgICAgICAgICB9LAogICAgICAgICAgICAgICAgICAgICAgICAgICAiZjplbmFibGVTZXJ2aWNlTGlua3MiOnsKICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgCiAgICAgICAgICAgICAgICAgICAgICAgICAgIH0sCiAgICAgICAgICAgICAgICAgICAgICAgICAgICJmOnJlc3RhcnRQb2xpY3kiOnsKICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgCiAgICAgICAgICAgICAgICAgICAgICAgICAgIH0sCiAgICAgICAgICAgICAgICAgICAgICAgICAgICJmOnNjaGVkdWxlck5hbWUiOnsKICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgCiAgICAgICAgICAgICAgICAgICAgICAgICAgIH0sCiAgICAgICAgICAgICAgICAgICAgICAgICAgICJmOnNlY3VyaXR5Q29udGV4dCI6ewogICAgICAgICAgICAgICAgICAgICAgICAgICAgICAKICAgICAgICAgICAgICAgICAgICAgICAgICAgfSwKICAgICAgICAgICAgICAgICAgICAgICAgICAgImY6dGVybWluYXRpb25HcmFjZVBlcmlvZFNlY29uZHMiOnsKICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgCiAgICAgICAgICAgICAgICAgICAgICAgICAgIH0KICAgICAgICAgICAgICAgICAgICAgICAgfQogICAgICAgICAgICAgICAgICAgICB9CiAgICAgICAgICAgICAgICAgIH0KICAgICAgICAgICAgICAgXQogICAgICAgICAgICB9LAogICAgICAgICAgICAic3BlYyI6ewogICAgICAgICAgICAgICAidm9sdW1lcyI6WwogICAgICAgICAgICAgICAgICB7CiAgICAgICAgICAgICAgICAgICAgICJuYW1lIjoia3ViZS1hcGktYWNjZXNzLTlzemRzIiwKICAgICAgICAgICAgICAgICAgICAgInByb2plY3RlZCI6ewogICAgICAgICAgICAgICAgICAgICAgICAic291cmNlcyI6WwogICAgICAgICAgICAgICAgICAgICAgICAgICB7CiAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICJzZXJ2aWNlQWNjb3VudFRva2VuIjp7CiAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICJleHBpcmF0aW9uU2Vjb25kcyI6MzYwNywKICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgInBhdGgiOiJ0b2tlbiIKICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgfQogICAgICAgICAgICAgICAgICAgICAgICAgICB9LAogICAgICAgICAgICAgICAgICAgICAgICAgICB7CiAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICJjb25maWdNYXAiOnsKICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgIm5hbWUiOiJrdWJlLXJvb3QtY2EuY3J0IiwKICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgIml0ZW1zIjpbCiAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgIHsKICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgImtleSI6ImNhLmNydCIsCiAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICJwYXRoIjoiY2EuY3J0IgogICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICB9CiAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgIF0KICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgfQogICAgICAgICAgICAgICAgICAgICAgICAgICB9LAogICAgICAgICAgICAgICAgICAgICAgICAgICB7CiAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICJkb3dud2FyZEFQSSI6ewogICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAiaXRlbXMiOlsKICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgewogICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAicGF0aCI6Im5hbWVzcGFjZSIsCiAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICJmaWVsZFJlZiI6ewogICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAiYXBpVmVyc2lvbiI6InYxIiwKICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgImZpZWxkUGF0aCI6Im1ldGFkYXRhLm5hbWVzcGFjZSIKICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgfQogICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICB9CiAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgIF0KICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgfQogICAgICAgICAgICAgICAgICAgICAgICAgICB9CiAgICAgICAgICAgICAgICAgICAgICAgIF0sCiAgICAgICAgICAgICAgICAgICAgICAgICJkZWZhdWx0TW9kZSI6NDIwCiAgICAgICAgICAgICAgICAgICAgIH0KICAgICAgICAgICAgICAgICAgfQogICAgICAgICAgICAgICBdLAogICAgICAgICAgICAgICAiY29udGFpbmVycyI6WwogICAgICAgICAgICAgICAgICB7CiAgICAgICAgICAgICAgICAgICAgICJuYW1lIjoicmVkaXMiLAogICAgICAgICAgICAgICAgICAgICAiaW1hZ2UiOiJyZWRpcyIsCiAgICAgICAgICAgICAgICAgICAgICJyZXNvdXJjZXMiOnsKICAgICAgICAgICAgICAgICAgICAgICAgCiAgICAgICAgICAgICAgICAgICAgIH0sCiAgICAgICAgICAgICAgICAgICAgICJ2b2x1bWVNb3VudHMiOlsKICAgICAgICAgICAgICAgICAgICAgICAgewogICAgICAgICAgICAgICAgICAgICAgICAgICAibmFtZSI6Imt1YmUtYXBpLWFjY2Vzcy05c3pkcyIsCiAgICAgICAgICAgICAgICAgICAgICAgICAgICJyZWFkT25seSI6dHJ1ZSwKICAgICAgICAgICAgICAgICAgICAgICAgICAgIm1vdW50UGF0aCI6Ii92YXIvcnVuL3NlY3JldHMva3ViZXJuZXRlcy5pby9zZXJ2aWNlYWNjb3VudCIKICAgICAgICAgICAgICAgICAgICAgICAgfQogICAgICAgICAgICAgICAgICAgICBdLAogICAgICAgICAgICAgICAgICAgICAidGVybWluYXRpb25NZXNzYWdlUGF0aCI6Ii9kZXYvdGVybWluYXRpb24tbG9nIiwKICAgICAgICAgICAgICAgICAgICAgInRlcm1pbmF0aW9uTWVzc2FnZVBvbGljeSI6IkZpbGUiLAogICAgICAgICAgICAgICAgICAgICAiaW1hZ2VQdWxsUG9saWN5IjoiQWx3YXlzIgogICAgICAgICAgICAgICAgICB9CiAgICAgICAgICAgICAgIF0sCiAgICAgICAgICAgICAgICJyZXN0YXJ0UG9saWN5IjoiQWx3YXlzIiwKICAgICAgICAgICAgICAgInRlcm1pbmF0aW9uR3JhY2VQZXJpb2RTZWNvbmRzIjozMCwKICAgICAgICAgICAgICAgImRuc1BvbGljeSI6IkNsdXN0ZXJGaXJzdCIsCiAgICAgICAgICAgICAgICJzZXJ2aWNlQWNjb3VudE5hbWUiOiJkZWZhdWx0IiwKICAgICAgICAgICAgICAgInNlcnZpY2VBY2NvdW50IjoiZGVmYXVsdCIsCiAgICAgICAgICAgICAgICJzZWN1cml0eUNvbnRleHQiOnsKICAgICAgICAgICAgICAgICAgCiAgICAgICAgICAgICAgIH0sCiAgICAgICAgICAgICAgICJzY2hlZHVsZXJOYW1lIjoiZGVmYXVsdC1zY2hlZHVsZXIiLAogICAgICAgICAgICAgICAidG9sZXJhdGlvbnMiOlsKICAgICAgICAgICAgICAgICAgewogICAgICAgICAgICAgICAgICAgICAia2V5Ijoibm9kZS5rdWJlcm5ldGVzLmlvL25vdC1yZWFkeSIsCiAgICAgICAgICAgICAgICAgICAgICJvcGVyYXRvciI6IkV4aXN0cyIsCiAgICAgICAgICAgICAgICAgICAgICJlZmZlY3QiOiJOb0V4ZWN1dGUiLAogICAgICAgICAgICAgICAgICAgICAidG9sZXJhdGlvblNlY29uZHMiOjMwMAogICAgICAgICAgICAgICAgICB9LAogICAgICAgICAgICAgICAgICB7CiAgICAgICAgICAgICAgICAgICAgICJrZXkiOiJub2RlLmt1YmVybmV0ZXMuaW8vdW5yZWFjaGFibGUiLAogICAgICAgICAgICAgICAgICAgICAib3BlcmF0b3IiOiJFeGlzdHMiLAogICAgICAgICAgICAgICAgICAgICAiZWZmZWN0IjoiTm9FeGVjdXRlIiwKICAgICAgICAgICAgICAgICAgICAgInRvbGVyYXRpb25TZWNvbmRzIjozMDAKICAgICAgICAgICAgICAgICAgfQogICAgICAgICAgICAgICBdLAogICAgICAgICAgICAgICAicHJpb3JpdHkiOjAsCiAgICAgICAgICAgICAgICJlbmFibGVTZXJ2aWNlTGlua3MiOnRydWUsCiAgICAgICAgICAgICAgICJwcmVlbXB0aW9uUG9saWN5IjoiUHJlZW1wdExvd2VyUHJpb3JpdHkiCiAgICAgICAgICAgIH0sCiAgICAgICAgICAgICJzdGF0dXMiOnsKICAgICAgICAgICAgICAgInBoYXNlIjoiUGVuZGluZyIsCiAgICAgICAgICAgICAgICJxb3NDbGFzcyI6IkJlc3RFZmZvcnQiCiAgICAgICAgICAgIH0KICAgICAgICAgfSwKICAgICAgICAgIm9sZE9iamVjdCI6bnVsbCwKICAgICAgICAgImRyeVJ1biI6ZmFsc2UsCiAgICAgICAgICJvcHRpb25zIjp7CiAgICAgICAgICAgICJraW5kIjoiQ3JlYXRlT3B0aW9ucyIsCiAgICAgICAgICAgICJhcGlWZXJzaW9uIjoibWV0YS5rOHMuaW8vdjEiCiAgICAgICAgIH0KICAgICAgfQogICAKfQo=",
		},
		Discover: &providers.Discovery{
			Targets: []string{"all"},
		},
	}, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, 4, len(assetList))
	assert.Equal(t, assetList[1].Platform.Name, "k8s-pod")
	assert.Contains(t, assetList[1].Platform.Family, "k8s-workload")
	assert.Contains(t, assetList[1].Platform.Family, "k8s")
	assert.Equal(t, assetList[2].Platform.Runtime, "docker-registry")
	assert.Equal(t, assetList[3].Platform.Runtime, "k8s-admission")
}

func TestManifestResolverDiscoveries(t *testing.T) {
	testCases := []struct {
		kind            string
		discoveryOption string
		platformName    string
		numAssets       int
	}{
		{
			kind:            "pod",
			discoveryOption: "pods",
			platformName:    "k8s-pod",
			numAssets:       3,
		},
		{
			kind:            "cronjob",
			discoveryOption: "cronjobs",
			platformName:    "k8s-cronjob",
			numAssets:       2,
		},
		{
			kind:            "job",
			discoveryOption: "jobs",
			platformName:    "k8s-job",
			numAssets:       2,
		},
		{
			kind:            "statefulset",
			discoveryOption: "statefulsets",
			platformName:    "k8s-statefulset",
			numAssets:       2,
		},
		{
			kind:            "daemonset",
			discoveryOption: "daemonsets",
			platformName:    "k8s-daemonset",
			numAssets:       2,
		},
		{
			kind:            "replicaset",
			discoveryOption: "replicasets",
			platformName:    "k8s-replicaset",
			numAssets:       2,
		},
		{
			kind:            "deployment",
			discoveryOption: "deployments",
			platformName:    "k8s-deployment",
			numAssets:       2,
		},
	}

	for _, testCase := range testCases {
		t.Run("discover k8s "+testCase.kind, func(t *testing.T) {
			resolver := &Resolver{}
			manifestFile := "../../providers/k8s/resources/testdata/" + testCase.kind + ".yaml"

			ctx := resources.SetDiscoveryCache(context.Background(), resources.NewDiscoveryCache())

			assetList, err := resolver.Resolve(ctx, &asset.Asset{}, &providers.Config{
				PlatformId: "//platform/k8s/uid/123/namespace/default/" + testCase.discoveryOption + "/name/mondoo",
				Backend:    providers.ProviderType_K8S,
				Options: map[string]string{
					"path": manifestFile,
				},
				Discover: &providers.Discovery{
					Targets: []string{testCase.discoveryOption},
				},
			}, nil, nil)
			require.NoError(t, err)
			// When this check fails locally, check your kubeconfig.
			// context has to reference the default namespace
			assert.Equal(t, testCase.numAssets, len(assetList))
			assert.Contains(t, assetList[1].Platform.Family, "k8s-workload")
			assert.Contains(t, assetList[1].Platform.Family, "k8s")
			assert.Equal(t, "k8s-manifest", assetList[1].Platform.Runtime)
			assert.Equal(t, testCase.platformName, assetList[1].Platform.Name)
			assert.Equal(t, "default/mondoo", assetList[1].Name)
		})
	}
}

func TestManifestResolverMultiPodDiscovery(t *testing.T) {
	resolver := &Resolver{}
	manifestFile := "../../providers/k8s/resources/testdata/pod.yaml"

	ctx := resources.SetDiscoveryCache(context.Background(), resources.NewDiscoveryCache())

	assetList, err := resolver.Resolve(ctx, &asset.Asset{}, &providers.Config{
		PlatformId: "//platform/k8s/uid/123/namespace/default/pods/name/mondoo",
		Backend:    providers.ProviderType_K8S,
		Options: map[string]string{
			"path": manifestFile,
		},
		Discover: &providers.Discovery{
			Targets: []string{"pods"},
		},
	}, nil, nil)
	require.NoError(t, err)
	// When this check fails locally, check your kubeconfig.
	// context has to reference the default namespace
	assert.Equal(t, 3, len(assetList))
	assert.Contains(t, assetList[1].Platform.Family, "k8s-workload")
	assert.Contains(t, assetList[1].Platform.Family, "k8s")
	assert.Equal(t, "k8s-manifest", assetList[1].Platform.Runtime)
	assert.Equal(t, "k8s-pod", assetList[1].Platform.Name)
	assert.Equal(t, "default/mondoo", assetList[1].Name)
	assert.Equal(t, "k8s-manifest", assetList[2].Platform.Runtime)
	assert.Equal(t, "k8s-pod", assetList[2].Platform.Name)
	assert.Equal(t, "default/hello-pod-2", assetList[2].Name)
}

func TestManifestResolverWrongDiscovery(t *testing.T) {
	resolver := &Resolver{}
	manifestFile := "../../providers/k8s/resources/testdata/cronjob.yaml"

	ctx := resources.SetDiscoveryCache(context.Background(), resources.NewDiscoveryCache())

	assetList, err := resolver.Resolve(ctx, &asset.Asset{}, &providers.Config{
		Backend: providers.ProviderType_K8S,
		Options: map[string]string{
			"path":      manifestFile,
			"namespace": "default",
		},
		Discover: &providers.Discovery{
			Targets: []string{"pods"},
		},
	}, nil, nil)
	require.NoError(t, err)
	// When this check fails locally, check your kubeconfig.
	// context has to reference the default namespace
	assert.Equalf(t, 1, len(assetList), "discovering pods in a cronjob manifest should only result in the manifest")
}

func TestResourceFilter(t *testing.T) {
	cfg := &providers.Config{
		Backend: providers.ProviderType_K8S,
		Options: map[string]string{
			"k8s-resources": "pod:default:nginx, pod:default:redis, deployment:test:redis, node:node1",
		},
	}

	resFilters, err := resourceFilters(cfg)
	require.NoError(t, err)

	expected := map[string][]K8sResourceIdentifier{
		"pod": {
			{Type: "pod", Namespace: "default", Name: "nginx"},
			{Type: "pod", Namespace: "default", Name: "redis"},
		},
		"deployment": {
			{Type: "deployment", Namespace: "test", Name: "redis"},
		},
		"node": {
			{Type: "node", Namespace: "", Name: "node1"},
		},
	}

	assert.Equal(t, expected, resFilters)
}
