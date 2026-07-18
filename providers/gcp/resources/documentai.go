// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"errors"
	"fmt"
	"strings"

	documentai "cloud.google.com/go/documentai/apiv1"
	"cloud.google.com/go/documentai/apiv1/documentaipb"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/gcp/connection"
	googleoauth "golang.org/x/oauth2/google"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// documentaiLocations lists the Document AI processing locations. Processors
// are created in one of these two multi-regions.
// See https://cloud.google.com/document-ai/docs/regions
var documentaiLocations = []string{"us", "eu"}

// documentaiEndpoint returns the regional API endpoint for a location.
func documentaiEndpoint(location string) string {
	return fmt.Sprintf("%s-documentai.googleapis.com:443", location)
}

// documentaiLocationFromName extracts the location from a Document AI resource
// name of the form projects/{project}/locations/{location}/....
func documentaiLocationFromName(name string) string {
	parts := strings.Split(name, "/")
	for i := 0; i+1 < len(parts); i++ {
		if parts[i] == "locations" {
			return parts[i+1]
		}
	}
	return ""
}

// isDocumentaiSkippable returns true for errors indicating the Document AI API
// is not enabled or the location is not available for this project.
func isDocumentaiSkippable(err error) bool {
	if s, ok := status.FromError(err); ok {
		switch s.Code() {
		case codes.PermissionDenied, codes.Unimplemented, codes.InvalidArgument, codes.NotFound:
			return true
		}
	}
	return false
}

func (g *mqlGcpProject) documentai() (*mqlGcpProjectDocumentaiService, error) {
	if g.Id.Error != nil {
		return nil, g.Id.Error
	}
	res, err := CreateResource(g.MqlRuntime, "gcp.project.documentaiService", map[string]*llx.RawData{
		"projectId": llx.StringData(g.Id.Data),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlGcpProjectDocumentaiService), nil
}

func initGcpProjectDocumentaiService(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if len(args) > 0 {
		return args, nil, nil
	}
	conn, ok := runtime.Connection.(*connection.GcpConnection)
	if !ok {
		return nil, nil, errors.New("invalid connection provided, it is not a GCP connection")
	}
	args["projectId"] = llx.StringData(conn.ResourceID())
	return args, nil, nil
}

func (g *mqlGcpProjectDocumentaiService) id() (string, error) {
	if g.ProjectId.Error != nil {
		return "", g.ProjectId.Error
	}
	return fmt.Sprintf("gcp.project/%s/documentaiService", g.ProjectId.Data), nil
}

func (g *mqlGcpProjectDocumentaiService) listProcessorsInLocation(ctx context.Context, creds *googleoauth.Credentials, projectId, location string) ([]any, error) {
	client, err := documentai.NewDocumentProcessorClient(ctx,
		option.WithCredentials(creds), connection.GRPCClientTraceOption(),
		option.WithEndpoint(documentaiEndpoint(location)),
	)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	var items []any
	it := client.ListProcessors(ctx, &documentaipb.ListProcessorsRequest{
		Parent: fmt.Sprintf("projects/%s/locations/%s", projectId, location),
	})
	for {
		p, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			if isDocumentaiSkippable(err) {
				break
			}
			return nil, err
		}

		res, err := CreateResource(g.MqlRuntime, "gcp.project.documentaiService.processor", map[string]*llx.RawData{
			"name":                    llx.StringData(p.Name),
			"type":                    llx.StringData(p.Type),
			"displayName":             llx.StringData(p.DisplayName),
			"state":                   llx.StringData(p.State.String()),
			"location":                llx.StringData(location),
			"processEndpoint":         llx.StringData(p.ProcessEndpoint),
			"defaultProcessorVersion": llx.StringData(p.DefaultProcessorVersion),
			"satisfiesPzs":            llx.BoolData(p.SatisfiesPzs),
			"createdAt":               llx.TimeDataPtr(timestampAsTimePtr(p.CreateTime)),
		})
		if err != nil {
			return nil, err
		}
		res.(*mqlGcpProjectDocumentaiServiceProcessor).cacheKmsKeyName = p.KmsKeyName
		items = append(items, res)
	}
	return items, nil
}

func (g *mqlGcpProjectDocumentaiService) processors() ([]any, error) {
	if g.ProjectId.Error != nil {
		return nil, g.ProjectId.Error
	}
	projectId := g.ProjectId.Data

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)
	creds, err := conn.Credentials(documentai.DefaultAuthScopes()...)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	var items []any
	for _, location := range documentaiLocations {
		locationItems, err := g.listProcessorsInLocation(ctx, creds, projectId, location)
		if err != nil {
			return nil, err
		}
		items = append(items, locationItems...)
	}
	return items, nil
}

func (g *mqlGcpProjectDocumentaiServiceProcessor) id() (string, error) {
	return g.Name.Data, g.Name.Error
}

type mqlGcpProjectDocumentaiServiceProcessorInternal struct {
	cacheKmsKeyName string
}

func (g *mqlGcpProjectDocumentaiServiceProcessor) kmsKey() (*mqlGcpProjectKmsServiceKeyringCryptokey, error) {
	return newKmsCryptoKeyRef(g.MqlRuntime, &g.KmsKey, g.cacheKmsKeyName)
}

func (g *mqlGcpProjectDocumentaiServiceProcessor) versions() ([]any, error) {
	if g.Name.Error != nil {
		return nil, g.Name.Error
	}
	processorName := g.Name.Data
	location := documentaiLocationFromName(processorName)
	if location == "" {
		return []any{}, nil
	}

	conn := g.MqlRuntime.Connection.(*connection.GcpConnection)
	creds, err := conn.Credentials(documentai.DefaultAuthScopes()...)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	client, err := documentai.NewDocumentProcessorClient(ctx,
		option.WithCredentials(creds), connection.GRPCClientTraceOption(),
		option.WithEndpoint(documentaiEndpoint(location)),
	)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	var items []any
	it := client.ListProcessorVersions(ctx, &documentaipb.ListProcessorVersionsRequest{Parent: processorName})
	for {
		v, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			if isDocumentaiSkippable(err) {
				break
			}
			return nil, err
		}

		deprecated := false
		deprecationTime := llx.NilData
		replacementVersion := ""
		if v.DeprecationInfo != nil {
			deprecated = true
			deprecationTime = llx.TimeDataPtr(timestampAsTimePtr(v.DeprecationInfo.DeprecationTime))
			replacementVersion = v.DeprecationInfo.ReplacementProcessorVersion
		}

		res, err := CreateResource(g.MqlRuntime, "gcp.project.documentaiService.processor.version", map[string]*llx.RawData{
			"name":               llx.StringData(v.Name),
			"displayName":        llx.StringData(v.DisplayName),
			"state":              llx.StringData(v.State.String()),
			"googleManaged":      llx.BoolData(v.GoogleManaged),
			"modelType":          llx.StringData(v.ModelType.String()),
			"deprecated":         llx.BoolData(deprecated),
			"deprecationTime":    deprecationTime,
			"replacementVersion": llx.StringData(replacementVersion),
			"createdAt":          llx.TimeDataPtr(timestampAsTimePtr(v.CreateTime)),
		})
		if err != nil {
			return nil, err
		}
		res.(*mqlGcpProjectDocumentaiServiceProcessorVersion).cacheKmsKeyName = v.KmsKeyName
		items = append(items, res)
	}
	return items, nil
}

func (g *mqlGcpProjectDocumentaiServiceProcessorVersion) id() (string, error) {
	return g.Name.Data, g.Name.Error
}

type mqlGcpProjectDocumentaiServiceProcessorVersionInternal struct {
	cacheKmsKeyName string
}

func (g *mqlGcpProjectDocumentaiServiceProcessorVersion) kmsKey() (*mqlGcpProjectKmsServiceKeyringCryptokey, error) {
	return newKmsCryptoKeyRef(g.MqlRuntime, &g.KmsKey, g.cacheKmsKeyName)
}
