// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package explorer

import (
	"context"
	"errors"
	"sort"
	"strings"

	"github.com/rs/zerolog/log"
	"go.mondoo.com/cnquery/v10"
	llx "go.mondoo.com/cnquery/v10/llx"
	"go.mondoo.com/cnquery/v10/mqlc"
	"go.mondoo.com/cnquery/v10/mrn"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/resources"
	"go.mondoo.com/cnquery/v10/utils/multierr"
	"go.mondoo.com/ranger-rpc/codes"
	"go.mondoo.com/ranger-rpc/status"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/protobuf/proto"
)

func (s *LocalServices) Assign(ctx context.Context, assignment *Assignment) (*Empty, error) {
	if len(assignment.PackMrns) == 0 {
		return nil, status.Error(codes.InvalidArgument, "no query pack MRNs were provided")
	}

	// all remote, call upstream
	if s.Upstream != nil && !s.Incognito {
		return s.Upstream.QueryConductor.Assign(ctx, assignment)
	}

	// cache everything from upstream
	if s.Upstream != nil && s.Incognito {
		// NOTE: we request the packs to cache them
		for i := range assignment.PackMrns {
			mrn := assignment.PackMrns[i]
			_, err := s.GetQueryPack(ctx, &Mrn{
				Mrn: mrn,
			})
			if err != nil {
				return nil, err
			}
		}
	}

	// assign a query pack locally
	deltas := map[string]*AssignmentDelta{}
	for i := range assignment.PackMrns {
		packMrn := assignment.PackMrns[i]
		deltas[packMrn] = &AssignmentDelta{
			Mrn:    packMrn,
			Action: AssignmentDelta_ADD,
		}
	}

	s.DataLake.EnsureAsset(ctx, assignment.AssetMrn)

	_, err := s.DataLake.MutateBundle(ctx, &BundleMutationDelta{
		OwnerMrn: assignment.AssetMrn,
		Deltas:   deltas,
	}, true)
	return globalEmpty, err
}

func (s *LocalServices) Unassign(ctx context.Context, assignment *Assignment) (*Empty, error) {
	if len(assignment.PackMrns) == 0 {
		return nil, status.Error(codes.InvalidArgument, "no query pack MRNs were provided")
	}

	// all remote, call upstream
	if s.Upstream != nil && !s.Incognito {
		return s.Upstream.QueryConductor.Unassign(ctx, assignment)
	}

	deltas := map[string]*AssignmentDelta{}
	for i := range assignment.PackMrns {
		packMrn := assignment.PackMrns[i]
		deltas[packMrn] = &AssignmentDelta{
			Mrn:    packMrn,
			Action: AssignmentDelta_DELETE,
		}
	}

	_, err := s.DataLake.MutateBundle(ctx, &BundleMutationDelta{
		OwnerMrn: assignment.AssetMrn,
		Deltas:   deltas,
	}, true)
	return globalEmpty, err
}

func (s *LocalServices) SetProps(ctx context.Context, req *PropsReq) (*Empty, error) {
	// validate that the queries compile and fill in checksums
	for i := range req.Props {
		prop := req.Props[i]
		conf := mqlc.NewConfig(s.runtime.Schema(), cnquery.DefaultFeatures)
		code, err := prop.RefreshChecksumAndType(conf)
		if err != nil {
			return nil, err
		}
		prop.CodeId = code.CodeV2.Id
	}

	return globalEmpty, s.DataLake.SetProps(ctx, req)
}

// Resolve executable bits for an asset (via asset filters)
func (s *LocalServices) Resolve(ctx context.Context, req *ResolveReq) (*ResolvedPack, error) {
	if s.Upstream != nil && !s.Incognito {
		res, err := s.Upstream.Resolve(ctx, req)
		if err != nil {
			return nil, err
		}

		err = s.DataLake.SetResolvedPack(req.EntityMrn, res.FiltersChecksum, res)
		if err != nil {
			return nil, err
		}

		err = s.DataLake.SetAssetResolvedPack(ctx, req.EntityMrn, res, V2Code)
		return res, err
	}

	bundle, err := s.DataLake.GetBundle(ctx, req.EntityMrn)
	if err != nil {
		return nil, err
	}

	bundleMap := bundle.ToMap()

	filtersChecksum, err := MatchFilters(req.EntityMrn, req.AssetFilters, bundle.Packs, s.runtime.Schema())
	if err != nil {
		return nil, err
	}

	supportedFilters := make(map[string]struct{}, len(req.AssetFilters))
	for i := range req.AssetFilters {
		f := req.AssetFilters[i]
		supportedFilters[f.CodeId] = struct{}{}
	}

	job := ExecutionJob{
		Queries:    make(map[string]*ExecutionQuery),
		Datapoints: make(map[string]*DataQueryInfo),
	}
	for i := range bundle.Packs {
		pack := bundle.Packs[i]

		if !pack.Filters.Supports(supportedFilters) {
			continue
		}

		props := NewPropsCache()
		props.Add(bundle.Props...)

		for i := range pack.Queries {
			err := s.addQueryToJob(ctx, pack.Queries[i], &job, props, supportedFilters, bundleMap)
			if err != nil {
				return nil, err
			}
		}

		for i := range pack.Groups {
			group := pack.Groups[i]

			if !group.Filters.Supports(supportedFilters) {
				continue
			}

			for i := range group.Queries {
				err := s.addQueryToJob(ctx, group.Queries[i], &job, props, supportedFilters, bundleMap)
				if err != nil {
					return nil, err
				}
			}
		}
	}

	res := &ResolvedPack{
		ExecutionJob: &job,
	}

	err = s.DataLake.SetResolvedPack(req.EntityMrn, filtersChecksum, res)
	if err != nil {
		return nil, err
	}

	err = s.DataLake.SetAssetResolvedPack(ctx, req.EntityMrn, res, V2Code)
	return res, err
}

func (s *LocalServices) addQueryToJob(ctx context.Context, query *Mquery, job *ExecutionJob, propsCache PropsCache, supportedFilters map[string]struct{}, bundle *BundleMap) error {
	if !query.Filters.Supports(supportedFilters) {
		return nil
	}

	compilerConfig := mqlc.NewConfig(s.runtime.Schema(), cnquery.DefaultFeatures)

	var props map[string]*llx.Primitive
	var propRefs map[string]string
	if len(query.Props) != 0 {
		props = map[string]*llx.Primitive{}
		propRefs = map[string]string{}

		for i := range query.Props {
			prop := query.Props[i]

			override, name, _ := propsCache.Get(prop.Mrn)
			if override != nil {
				prop = override
			}
			if name == "" {
				var err error
				name, err = mrn.GetResource(prop.Mrn, MRN_RESOURCE_QUERY)
				if err != nil {
					return errors.New("failed to get property name")
				}
			}

			props[name] = &llx.Primitive{Type: prop.Type}
			propRefs[name] = prop.CodeId

			if _, ok := job.Queries[prop.CodeId]; ok {
				continue
			}

			code, err := prop.Compile(nil, compilerConfig)
			if err != nil {
				return multierr.Wrap(err, "failed to compile property for query "+query.Mrn)
			}
			job.Queries[prop.CodeId] = &ExecutionQuery{
				Query:    prop.Mql,
				Checksum: prop.Checksum,
				Code:     code,
			}
		}
	}

	if len(query.Variants) != 0 {
		for i := range query.Variants {
			ref := query.Variants[i].Mrn
			err := s.addQueryToJob(ctx, bundle.Queries[ref], job, propsCache, supportedFilters, bundle)
			if err != nil {
				return err
			}
		}
		return nil
	}

	codeBundle, err := query.Compile(props, compilerConfig)
	if err != nil {
		return err
	}

	equery := &ExecutionQuery{
		Query:      query.Mql,
		Checksum:   query.Checksum,
		Code:       codeBundle,
		Properties: propRefs,
	}

	code := equery.Code.CodeV2
	refs := append(code.Datapoints(), code.Entrypoints()...)

	job.Queries[query.CodeId] = equery
	for i := range refs {
		ref := refs[i]
		checksum := code.Checksums[ref]
		typ := code.Chunk(ref).DereferencedTypeV2(code)

		job.Datapoints[checksum] = &DataQueryInfo{
			Type: string(typ),
		}
	}

	return nil
}

// MatchFilters will take the list of filters and only return the ones
// that are supported by the given querypacks.
func MatchFilters(entityMrn string, filters []*Mquery, packs []*QueryPack, schema resources.ResourcesSchema) (string, error) {
	supported := map[string]*Mquery{}
	for i := range packs {
		pack := packs[i]
		if pack.ComputedFilters == nil {
			continue
		}

		for k, v := range pack.ComputedFilters.Items {
			supported[k] = v
		}
	}

	matching := []*Mquery{}
	for i := range filters {
		cur := filters[i]

		if _, ok := supported[cur.CodeId]; ok {
			curCopy := proto.Clone(cur).(*Mquery)
			curCopy.Mrn = entityMrn + "/assetfilter/" + cur.CodeId
			curCopy.Title = curCopy.Query
			matching = append(matching, curCopy)
		}
	}

	if len(matching) == 0 {
		return "", NewAssetMatchError(entityMrn, "querypacks", "no-matching-packs", filters, &Filters{Items: supported})
	}

	conf := mqlc.NewConfig(schema, cnquery.DefaultFeatures)
	sum, err := ChecksumFilters(matching, conf)
	if err != nil {
		return "", err
	}

	return sum, nil
}

func NewAssetMatchError(mrn string, objectType string, errorReason string, assetFilters []*Mquery, supported *Filters) error {
	if len(assetFilters) == 0 {
		// send a proto error with details, so that the agent can render it properly
		msg := "asset doesn't support any " + objectType
		st := status.New(codes.InvalidArgument, msg)

		std, err := st.WithDetails(&errdetails.ErrorInfo{
			Domain: SERVICE_NAME,
			Reason: errorReason,
			Metadata: map[string]string{
				"mrn":       mrn,
				"errorCode": NotApplicable.String(),
			},
		})
		if err != nil {
			log.Error().Err(err).Msg("could not send status with additional information")
			return st.Err()
		}
		return std.Err()
	}

	supportedSummary := supported.Summarize()
	var supportedPrefix string
	if supportedSummary == "" {
		supportedPrefix = objectType + " didn't provide any filters"
	} else {
		supportedPrefix = objectType + " support: "
	}

	filters := make([]string, len(assetFilters))
	for i := range assetFilters {
		filters[i] = strings.TrimSpace(assetFilters[i].Mql)
	}
	sort.Strings(filters)
	foundSummary := strings.Join(filters, ", ")
	foundPrefix := "asset supports: "

	msg := "asset isn't supported by any " + objectType + "\n" +
		supportedPrefix + supportedSummary + "\n" +
		foundPrefix + foundSummary + "\n"
	return status.Error(codes.InvalidArgument, msg)
}

func (s *LocalServices) StoreResults(ctx context.Context, req *StoreResultsReq) (*Empty, error) {
	_, err := s.DataLake.UpdateData(ctx, req.AssetMrn, req.Data)
	if err != nil {
		return globalEmpty, err
	}

	if s.Upstream != nil && !s.Incognito {
		_, err := s.Upstream.QueryConductor.StoreResults(ctx, req)
		if err != nil {
			return globalEmpty, err
		}
	}

	return globalEmpty, nil
}

func (s *LocalServices) GetResourcesData(ctx context.Context, req *EntityResourcesReq) (*EntityResourcesRes, error) {
	res, err := s.DataLake.GetResources(ctx, req.EntityMrn, req.Resources)
	return &EntityResourcesRes{
		Resources: res,
		EntityMrn: req.EntityMrn,
	}, err
}

func (s *LocalServices) GetReport(ctx context.Context, req *EntityDataRequest) (*Report, error) {
	return s.DataLake.GetReport(ctx, req.EntityMrn, req.DataMrn)
}

func (s *LocalServices) SynchronizeAssets(context.Context, *SynchronizeAssetsReq) (*SynchronizeAssetsResp, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}
