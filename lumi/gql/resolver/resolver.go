package resolver

import (
	"go.mondoo.io/mondoo/lumi/gql"
	"go.mondoo.io/mondoo/motor"
	motor_resolver "go.mondoo.io/mondoo/motor/resolver"
	"go.mondoo.io/mondoo/motor/types"
)

func New() (*Resolver, error) {
	motor, err := motor_resolver.New(&types.Endpoint{
		Backend: "local",
	})
	if err != nil {
		return nil, err
	}
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

type queryResolver struct{ *Resolver }
