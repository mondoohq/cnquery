package resolver

import (
	"go.mondoo.io/mondoo/lumi/gql"
	"go.mondoo.io/mondoo/motor"
)

func New(motor *motor.Motor) (*Resolver, error) {
	return &Resolver{
		Runtime: &Runtime{
			Motor: motor,
		},
	}, nil
}

type Runtime struct {
	Motor *motor.Motor
}

type Resolver struct {
	Runtime *Runtime
}

func (r *Resolver) File() gql.FileResolver {
	return &fileResolver{r}
}

func (r *Resolver) Query() gql.QueryResolver {
	return &queryResolver{r}
}

func (r *Resolver) Kernel() gql.KernelResolver {
	return &kernelResolver{r}
}

func (r *Resolver) Sshd() gql.SshdResolver {
	return &sshdResolver{r}
}

func (r *Resolver) SshConfig() gql.SshConfigResolver {
	return &sshConfigResolver{r}
}

func (r *Resolver) Docker() gql.DockerResolver {
	return &dockerResolver{r}
}

func (r *Resolver) GoogleCloudPlatform() gql.GoogleCloudPlatformResolver {
	return &googleCloudPlatformResolver{r}
}

func (r *Resolver) GcpCompute() gql.GcpComputeResolver {
	return &gcpComputeResolver{r}
}

func (r *Resolver) GcpStorage() gql.GcpStorageResolver {
	return &gcpStorageResolver{r}
}

type queryResolver struct{ *Resolver }
