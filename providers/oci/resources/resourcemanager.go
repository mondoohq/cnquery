// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/oracle/oci-go-sdk/v65/common"
	"github.com/oracle/oci-go-sdk/v65/resourcemanager"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/util/convert"
	"go.mondoo.com/mql/v13/providers/oci/connection"
	"go.mondoo.com/mql/v13/types"
)

func (o *mqlOciResourceManager) id() (string, error) {
	return "oci.resourceManager", nil
}

func (o *mqlOciResourceManager) stacks() ([]any, error) {
	conn := o.MqlRuntime.Connection.(*connection.OciConnection)

	regions, err := ociRegionsFor(o.MqlRuntime)
	if err != nil {
		return nil, err
	}

	return ociRunCompartmentRegionPool(conn, regions,
		func(ctx context.Context, region string, compartmentID string) ([]any, error) {
			client, err := conn.ResourceManagerClient(region)
			if err != nil {
				return nil, err
			}

			stacks := []resourcemanager.StackSummary{}
			var page *string
			for {
				response, err := client.ListStacks(ctx, resourcemanager.ListStacksRequest{
					CompartmentId: common.String(compartmentID),
					Page:          page,
				})
				if err != nil {
					return nil, err
				}

				stacks = append(stacks, response.Items...)

				if response.OpcNextPage == nil {
					break
				}
				page = response.OpcNextPage
			}

			res := make([]any, 0, len(stacks))
			for i := range stacks {
				s := stacks[i]

				mqlStack, err := CreateResource(o.MqlRuntime, "oci.resourceManager.stack", map[string]*llx.RawData{
					"id":               llx.StringDataPtr(s.Id),
					"name":             llx.StringDataPtr(s.DisplayName),
					"description":      llx.StringDataPtr(s.Description),
					"terraformVersion": llx.StringDataPtr(s.TerraformVersion),
					"state":            llx.StringData(string(s.LifecycleState)),
					"created":          sdkTimeData(s.TimeCreated),
					"freeformTags":     llx.MapData(strMapToAny(s.FreeformTags), types.String),
					"definedTags":      llx.MapData(definedTagsToAny(s.DefinedTags), types.Any),
					"systemTags":       llx.MapData(definedTagsToAny(s.SystemTags), types.Dict),
				})
				if err != nil {
					return nil, err
				}
				mqlStackTyped := mqlStack.(*mqlOciResourceManagerStack)
				mqlStackTyped.cacheCompartmentId = stringValue(s.CompartmentId)
				mqlStackTyped.cacheRegion = region
				res = append(res, mqlStackTyped)
			}

			return res, nil
		})
}

// Drift status, the config source and the variable names are all detail-only,
// so they share one fetch.
type mqlOciResourceManagerStackInternal struct {
	cacheCompartmentId string
	cacheRegion        string

	detailLock    sync.Mutex
	detailFetched bool
	detail        *resourcemanager.Stack
}

func (o *mqlOciResourceManagerStack) id() (string, error) {
	return "oci.resourceManager.stack/" + o.Id.Data, nil
}

func (o *mqlOciResourceManagerStack) compartment() (*mqlOciCompartment, error) {
	return resolveOciCompartment(o.MqlRuntime, o.cacheCompartmentId, &o.Compartment)
}

func (o *mqlOciResourceManagerStack) getDetail() (*resourcemanager.Stack, error) {
	o.detailLock.Lock()
	defer o.detailLock.Unlock()
	if o.detailFetched {
		return o.detail, nil
	}

	conn := o.MqlRuntime.Connection.(*connection.OciConnection)
	client, err := conn.ResourceManagerClient(o.cacheRegion)
	if err != nil {
		return nil, err
	}

	response, err := client.GetStack(context.Background(), resourcemanager.GetStackRequest{
		StackId: common.String(o.Id.Data),
	})
	if err != nil {
		return nil, err
	}

	o.detail = &response.Stack
	o.detailFetched = true
	return o.detail, nil
}

func (o *mqlOciResourceManagerStack) driftStatus() (string, error) {
	detail, err := o.getDetail()
	if err != nil {
		return "", err
	}
	// An unset drift status means drift has never been evaluated, which is a
	// distinct answer from IN_SYNC and must not be reported as an empty string.
	if detail.StackDriftStatus == "" {
		return string(resourcemanager.StackStackDriftStatusNotChecked), nil
	}
	return string(detail.StackDriftStatus), nil
}

func (o *mqlOciResourceManagerStack) driftLastChecked() (*time.Time, error) {
	detail, err := o.getDetail()
	if err != nil {
		return nil, err
	}
	if detail.TimeDriftLastChecked == nil {
		return nil, nil
	}
	return &detail.TimeDriftLastChecked.Time, nil
}

func (o *mqlOciResourceManagerStack) configSource() (any, error) {
	detail, err := o.getDetail()
	if err != nil {
		return nil, err
	}
	if detail.ConfigSource == nil {
		return nil, nil
	}
	return convert.JsonToDict(detail.ConfigSource)
}

// variableNames returns the names of a stack's Terraform variables, never the
// values.
//
// Resource Manager stores variable values as plain strings with no sensitivity
// marking of any kind, and stacks routinely carry database passwords, API
// tokens and private keys in them. Returning the values would copy every one of
// those secrets into the scan result and anywhere that result is stored. The
// names alone answer the auditable question - whether a stack is carrying
// credentials it should be sourcing from Vault.
func (o *mqlOciResourceManagerStack) variableNames() ([]any, error) {
	detail, err := o.getDetail()
	if err != nil {
		return nil, err
	}

	names := make([]string, 0, len(detail.Variables))
	for name := range detail.Variables {
		names = append(names, name)
	}
	// Go randomizes map iteration; sort so the list is stable across queries.
	sort.Strings(names)

	return stringsToAny(names), nil
}

func (o *mqlOciResourceManagerStack) isThirdPartyProviderExperienceEnabled() (bool, error) {
	detail, err := o.getDetail()
	if err != nil {
		return false, err
	}
	return boolValue(detail.IsThirdPartyProviderExperienceEnabled), nil
}

func (o *mqlOciResourceManagerStack) jobs() ([]any, error) {
	conn := o.MqlRuntime.Connection.(*connection.OciConnection)

	client, err := conn.ResourceManagerClient(o.cacheRegion)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	jobs := []resourcemanager.JobSummary{}
	var page *string
	for {
		response, err := client.ListJobs(ctx, resourcemanager.ListJobsRequest{
			StackId: common.String(o.Id.Data),
			Page:    page,
		})
		if err != nil {
			return nil, err
		}

		jobs = append(jobs, response.Items...)

		if response.OpcNextPage == nil {
			break
		}
		page = response.OpcNextPage
	}

	res := make([]any, 0, len(jobs))
	for i := range jobs {
		job := jobs[i]

		mqlJob, err := CreateResource(o.MqlRuntime, "oci.resourceManager.job", map[string]*llx.RawData{
			"id":        llx.StringDataPtr(job.Id),
			"name":      llx.StringDataPtr(job.DisplayName),
			"operation": llx.StringData(string(job.Operation)),
			"state":     llx.StringData(string(job.LifecycleState)),
			"created":   sdkTimeData(job.TimeCreated),
			"finished":  sdkTimeData(job.TimeFinished),
		})
		if err != nil {
			return nil, err
		}
		mqlJobTyped := mqlJob.(*mqlOciResourceManagerJob)
		mqlJobTyped.cacheCompartmentId = stringValue(job.CompartmentId)
		res = append(res, mqlJobTyped)
	}

	return res, nil
}

type mqlOciResourceManagerJobInternal struct {
	cacheCompartmentId string
}

func (o *mqlOciResourceManagerJob) id() (string, error) {
	return "oci.resourceManager.job/" + o.Id.Data, nil
}

func (o *mqlOciResourceManagerJob) compartment() (*mqlOciCompartment, error) {
	return resolveOciCompartment(o.MqlRuntime, o.cacheCompartmentId, &o.Compartment)
}
