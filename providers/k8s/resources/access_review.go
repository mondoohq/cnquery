// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
)

type mqlK8sAccessReviewInternal struct {
	resolveOnce sync.Once
	allowedData bool
	reasonData  string
	resolveErr  error
}

// subjectAccessChecker is implemented by the live (api) connection, which can
// run a SubjectAccessReview against the cluster. Manifest connections do not
// implement it, so k8s.accessReview reports a clear error there.
type subjectAccessChecker interface {
	SubjectAccessAllowed(ctx context.Context, subject, namespace, verb, group, resource, name string) (bool, string, error)
}

func initK8sAccessReview(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	for _, f := range []string{"subject", "verb", "resource", "group", "namespace", "name"} {
		if _, ok := args[f]; !ok {
			args[f] = llx.StringData("")
		}
	}
	str := func(k string) string {
		if v, ok := args[k]; ok && v.Value != nil {
			if s, ok := v.Value.(string); ok {
				return s
			}
		}
		return ""
	}
	args["__id"] = llx.StringData(fmt.Sprintf("k8s.accessReview/%s/%s/%s/%s/%s/%s",
		str("subject"), str("namespace"), str("group"), str("resource"), str("name"), str("verb")))
	return args, nil, nil
}

// resolve runs the SubjectAccessReview once and caches the decision, so the
// allowed and reason fields share a single API call.
func (k *mqlK8sAccessReview) resolve() error {
	k.resolveOnce.Do(func() {
		conn, ok := k.MqlRuntime.Connection.(subjectAccessChecker)
		if !ok {
			k.resolveErr = errors.New("k8s.accessReview requires a live Kubernetes cluster connection")
			return
		}
		k.allowedData, k.reasonData, k.resolveErr = conn.SubjectAccessAllowed(context.Background(),
			k.Subject.Data, k.Namespace.Data, k.Verb.Data, k.Group.Data, k.Resource.Data, k.Name.Data)
	})
	return k.resolveErr
}

func (k *mqlK8sAccessReview) allowed() (bool, error) {
	if err := k.resolve(); err != nil {
		return false, err
	}
	return k.allowedData, nil
}

func (k *mqlK8sAccessReview) reason() (string, error) {
	if err := k.resolve(); err != nil {
		return "", err
	}
	return k.reasonData, nil
}
