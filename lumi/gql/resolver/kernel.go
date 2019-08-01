package resolver

import (
	"context"
	"errors"

	"go.mondoo.io/mondoo/lumi/gql"
)

func (r *queryResolver) Kernel(ctx context.Context) (*gql.Kernel, error) {
	return &gql.Kernel{}, nil
}

type kernelResolver struct{ *Resolver }

func (r *kernelResolver) Parameters(ctx context.Context, obj *gql.Kernel) ([]*gql.KeyValue, error) {
	if obj == nil {
		return nil, errors.New("no parent object defined")
	}

	return nil, errors.New("not implemented yet")

	// 	// this resource is only supported on linux
	// 	platform, err := r.Runtime.Motor.Platform()
	// 	if err != nil {
	// 		return nil, err
	// 	}

	// 	supported := false
	// 	for _, f := range platform.Family {
	// 		if f == "linux" {
	// 			supported = true
	// 		}
	// 	}
	// 	if supported == false {
	// 		return nil, errors.New("kernel resource is only supported for linux platforms")
	// 	}

	// 	sysctlPath := "/proc/sys/"
	// 	f, err := r.Runtime.Motor.Transport.File(sysctlPath)
	// 	if err != nil {
	// 		return nil, err
	// 	}

	// 	// TAR is not implemented in new motor filesystem yet
	// 	tarStream, err := f.Tar()
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// 	defer tarStream.Close()

	// 	kernelParameters, err := procfs.ParseLinuxSysctl(sysctlPath, tarStream)
	// 	if err != nil {
	// 		return nil, err
	// 	}

	// 	res := []gql.KeyValue{}
	// 	for k := range kernelParameters {
	// 		key := k
	// 		value := kernelParameters[k]
	// 		res = append(res, gql.KeyValue{
	// 			Key:   &key,
	// 			Value: &value,
	// 		})
	// 	}
	// 	return res, nil
}
