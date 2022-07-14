package container_registry

import (
	"fmt"
	"net/http"

	"github.com/google/go-containerregistry/pkg/v1/remote/transport"
)

func handleUnauthorizedError(err error, repoName string) error {
	if err != nil {
		if tErr, ok := err.(*transport.Error); ok && tErr.StatusCode == http.StatusUnauthorized {
			err = fmt.Errorf("cannot list repo %s due to missing container registry credentials", repoName)
		}
	}
	return err
}
