// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	rmclient "github.com/alibabacloud-go/resourcemanager-20200331/v3/client"
	tea "github.com/alibabacloud-go/tea/tea"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/alicloud/connection"
)

// rmParseTime parses a Resource Management ISO-8601 timestamp, which may carry
// fractional seconds. Returns nil on a nil or unparseable input.
func rmParseTime(s *string) *time.Time {
	if s == nil || *s == "" {
		return nil
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02T15:04:05.000Z", "2006-01-02T15:04:05Z"} {
		if t, err := time.Parse(layout, *s); err == nil {
			return &t
		}
	}
	return nil
}

// mqlAlicloudResourceManagerInternal memoizes the resource directory detail
// shared by the identity accessors.
type mqlAlicloudResourceManagerInternal struct {
	dirLock    sync.Mutex
	dirFetched atomic.Bool
	dir        *rmclient.GetResourceDirectoryResponseBodyResourceDirectory
}

func (r *mqlAlicloudResourceManager) id() (string, error) {
	return "alicloud.resourceManager", nil
}

// directory lazily fetches and caches the resource directory detail. A
// transient error is not cached and is returned so the identity accessors
// surface the failure rather than empty strings.
func (r *mqlAlicloudResourceManager) directory() (*rmclient.GetResourceDirectoryResponseBodyResourceDirectory, error) {
	if r.dirFetched.Load() {
		return r.dir, nil
	}
	r.dirLock.Lock()
	defer r.dirLock.Unlock()
	if r.dirFetched.Load() {
		return r.dir, nil
	}

	conn := r.MqlRuntime.Connection.(*connection.AlicloudConnection)
	client, err := conn.ResourceManagerClient()
	if err != nil {
		return nil, err
	}
	resp, err := client.GetResourceDirectory()
	if err != nil {
		return nil, err
	}
	if resp != nil && resp.Body != nil {
		r.dir = resp.Body.ResourceDirectory
	}
	r.dirFetched.Store(true)
	return r.dir, nil
}

func (r *mqlAlicloudResourceManager) resourceDirectoryId() (string, error) {
	d, err := r.directory()
	if err != nil || d == nil {
		return "", err
	}
	return tea.StringValue(d.ResourceDirectoryId), nil
}

func (r *mqlAlicloudResourceManager) rootFolderId() (string, error) {
	d, err := r.directory()
	if err != nil || d == nil {
		return "", err
	}
	return tea.StringValue(d.RootFolderId), nil
}

func (r *mqlAlicloudResourceManager) masterAccountId() (string, error) {
	d, err := r.directory()
	if err != nil || d == nil {
		return "", err
	}
	return tea.StringValue(d.MasterAccountId), nil
}

func (r *mqlAlicloudResourceManager) masterAccountName() (string, error) {
	d, err := r.directory()
	if err != nil || d == nil {
		return "", err
	}
	return tea.StringValue(d.MasterAccountName), nil
}

func (r *mqlAlicloudResourceManager) memberDeletionStatus() (string, error) {
	d, err := r.directory()
	if err != nil || d == nil {
		return "", err
	}
	return tea.StringValue(d.MemberDeletionStatus), nil
}

func (r *mqlAlicloudResourceManager) controlPolicyStatus() (string, error) {
	d, err := r.directory()
	if err != nil || d == nil {
		return "", err
	}
	return tea.StringValue(d.ControlPolicyStatus), nil
}

func (r *mqlAlicloudResourceManager) createTime() (*time.Time, error) {
	d, err := r.directory()
	if err != nil || d == nil {
		return nil, err
	}
	return rmParseTime(d.CreateTime), nil
}

func (r *mqlAlicloudResourceManager) accounts() ([]any, error) {
	conn := r.MqlRuntime.Connection.(*connection.AlicloudConnection)
	client, err := conn.ResourceManagerClient()
	if err != nil {
		return nil, err
	}

	res := []any{}
	pageNumber := int32(1)
	pageSize := int32(100)
	for {
		resp, err := client.ListAccounts(&rmclient.ListAccountsRequest{
			PageNumber: tea.Int32(pageNumber),
			PageSize:   tea.Int32(pageSize),
		})
		if err != nil {
			return nil, err
		}
		if resp == nil || resp.Body == nil || resp.Body.Accounts == nil {
			break
		}

		items := resp.Body.Accounts.Account
		for _, a := range items {
			if a == nil || a.AccountId == nil {
				continue
			}
			resource, err := CreateResource(r.MqlRuntime, "alicloud.resourceManager.account", map[string]*llx.RawData{
				"__id":                llx.StringDataPtr(a.AccountId),
				"accountId":           llx.StringDataPtr(a.AccountId),
				"displayName":         llx.StringDataPtr(a.DisplayName),
				"status":              llx.StringDataPtr(a.Status),
				"type":                llx.StringDataPtr(a.Type),
				"folderId":            llx.StringDataPtr(a.FolderId),
				"resourceDirectoryId": llx.StringDataPtr(a.ResourceDirectoryId),
				"joinMethod":          llx.StringDataPtr(a.JoinMethod),
				"joinTime":            llx.TimeDataPtr(rmParseTime(a.JoinTime)),
				"modifyTime":          llx.TimeDataPtr(rmParseTime(a.ModifyTime)),
			})
			if err != nil {
				return nil, err
			}
			res = append(res, resource)
		}

		total := tea.Int32Value(resp.Body.TotalCount)
		if len(items) < int(pageSize) || (total > 0 && pageNumber*pageSize >= total) {
			break
		}
		pageNumber++
	}
	return res, nil
}

func (r *mqlAlicloudResourceManager) folders() ([]any, error) {
	conn := r.MqlRuntime.Connection.(*connection.AlicloudConnection)
	client, err := conn.ResourceManagerClient()
	if err != nil {
		return nil, err
	}

	rootFolderId, err := r.rootFolderId()
	if err != nil || rootFolderId == "" {
		return []any{}, nil
	}

	res := []any{}
	// Walk the folder tree breadth-first from the root, listing each parent's
	// children. The root folder itself is not returned by the API, only its
	// descendants. A visited set guards against a scan hanging if the API ever
	// returns a folder that re-references an ancestor.
	visited := map[string]struct{}{}
	queue := []string{rootFolderId}
	for len(queue) > 0 {
		parent := queue[0]
		queue = queue[1:]
		if _, seen := visited[parent]; seen {
			continue
		}
		visited[parent] = struct{}{}

		pageNumber := int32(1)
		pageSize := int32(100)
		for {
			resp, err := client.ListFoldersForParent(&rmclient.ListFoldersForParentRequest{
				ParentFolderId: tea.String(parent),
				PageNumber:     tea.Int32(pageNumber),
				PageSize:       tea.Int32(pageSize),
			})
			if err != nil || resp == nil || resp.Body == nil || resp.Body.Folders == nil {
				break
			}
			items := resp.Body.Folders.Folder
			for _, f := range items {
				if f == nil || f.FolderId == nil {
					continue
				}
				resource, err := newResourceManagerFolder(r.MqlRuntime, f.FolderId, f.FolderName, tea.String(parent), f.CreateTime)
				if err != nil {
					return nil, err
				}
				res = append(res, resource)
				queue = append(queue, tea.StringValue(f.FolderId))
			}
			total := tea.Int32Value(resp.Body.TotalCount)
			if len(items) < int(pageSize) || (total > 0 && pageNumber*pageSize >= total) {
				break
			}
			pageNumber++
		}
	}
	return res, nil
}

func (r *mqlAlicloudResourceManager) controlPolicies() ([]any, error) {
	conn := r.MqlRuntime.Connection.(*connection.AlicloudConnection)
	client, err := conn.ResourceManagerClient()
	if err != nil {
		return nil, err
	}

	res := []any{}
	for _, policyType := range []string{"System", "Custom"} {
		pageNumber := int32(1)
		pageSize := int32(100)
		for {
			resp, err := client.ListControlPolicies(&rmclient.ListControlPoliciesRequest{
				PolicyType: tea.String(policyType),
				PageNumber: tea.Int32(pageNumber),
				PageSize:   tea.Int32(pageSize),
			})
			if err != nil || resp == nil || resp.Body == nil || resp.Body.ControlPolicies == nil {
				break
			}
			items := resp.Body.ControlPolicies.ControlPolicy
			for _, p := range items {
				if p == nil || p.PolicyId == nil {
					continue
				}
				attachmentCount := int64(0)
				if p.AttachmentCount != nil {
					if n, err := strconv.Atoi(*p.AttachmentCount); err == nil {
						attachmentCount = int64(n)
					}
				}
				resource, err := CreateResource(r.MqlRuntime, "alicloud.resourceManager.controlPolicy", map[string]*llx.RawData{
					"__id":            llx.StringDataPtr(p.PolicyId),
					"policyId":        llx.StringDataPtr(p.PolicyId),
					"policyName":      llx.StringDataPtr(p.PolicyName),
					"policyType":      llx.StringDataPtr(p.PolicyType),
					"description":     llx.StringDataPtr(p.Description),
					"effectScope":     llx.StringDataPtr(p.EffectScope),
					"attachmentCount": llx.IntData(attachmentCount),
					"createDate":      llx.TimeDataPtr(rmParseTime(p.CreateDate)),
					"updateDate":      llx.TimeDataPtr(rmParseTime(p.UpdateDate)),
				})
				if err != nil {
					return nil, err
				}
				res = append(res, resource)
			}
			total := tea.Int32Value(resp.Body.TotalCount)
			if len(items) < int(pageSize) || (total > 0 && pageNumber*pageSize >= total) {
				break
			}
			pageNumber++
		}
	}
	return res, nil
}

func (r *mqlAlicloudResourceManagerAccount) id() (string, error) {
	return r.AccountId.Data, nil
}

func (r *mqlAlicloudResourceManagerAccount) folder() (*mqlAlicloudResourceManagerFolder, error) {
	folderID := r.FolderId.Data
	if folderID == "" {
		r.Folder.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	res, err := NewResource(r.MqlRuntime, "alicloud.resourceManager.folder", map[string]*llx.RawData{
		"folderId": llx.StringData(folderID),
	})
	if err != nil {
		r.Folder.State = plugin.StateIsSet | plugin.StateIsNull
		return nil, nil
	}
	return res.(*mqlAlicloudResourceManagerFolder), nil
}

// newResourceManagerFolder builds a folder resource, carrying the parent id from
// the listing context since the list item does not include it.
func newResourceManagerFolder(runtime *plugin.Runtime, folderID, folderName, parentFolderID, createTime *string) (*mqlAlicloudResourceManagerFolder, error) {
	resource, err := CreateResource(runtime, "alicloud.resourceManager.folder", map[string]*llx.RawData{
		"__id":           llx.StringDataPtr(folderID),
		"folderId":       llx.StringDataPtr(folderID),
		"folderName":     llx.StringDataPtr(folderName),
		"parentFolderId": llx.StringDataPtr(parentFolderID),
		"createTime":     llx.TimeDataPtr(rmParseTime(createTime)),
	})
	if err != nil {
		return nil, err
	}
	return resource.(*mqlAlicloudResourceManagerFolder), nil
}

// initAlicloudResourceManagerFolder resolves a folder by id, reusing an
// already-listed folder from the resource cache and otherwise fetching it via
// GetFolder.
func initAlicloudResourceManagerFolder(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 1 {
		return args, nil, nil
	}

	folderID, err := requiredStringArg(args, "folderId", "alicloud.resourceManager.folder")
	if err != nil {
		return nil, nil, err
	}

	if x, ok := runtime.Resources.Get("alicloud.resourceManager.folder\x00" + folderID); ok {
		return nil, x, nil
	}

	conn := runtime.Connection.(*connection.AlicloudConnection)
	client, err := conn.ResourceManagerClient()
	if err != nil {
		return nil, nil, err
	}
	resp, err := client.GetFolder(&rmclient.GetFolderRequest{FolderId: tea.String(folderID)})
	if err != nil {
		return nil, nil, err
	}
	if resp == nil || resp.Body == nil || resp.Body.Folder == nil {
		return nil, nil, fmt.Errorf("alicloud.resourceManager.folder %q not found", folderID)
	}
	f := resp.Body.Folder
	res, err := newResourceManagerFolder(runtime, f.FolderId, f.FolderName, f.ParentFolderId, f.CreateTime)
	if err != nil {
		return nil, nil, err
	}
	return nil, res, nil
}

func (r *mqlAlicloudResourceManagerFolder) id() (string, error) {
	return r.FolderId.Data, nil
}

func (r *mqlAlicloudResourceManagerControlPolicy) id() (string, error) {
	return r.PolicyId.Data, nil
}
