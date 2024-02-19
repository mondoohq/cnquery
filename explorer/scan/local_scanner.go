// Copyright (c) Mondoo, Inc.
// SPDX-License-Identifier: BUSL-1.1

package scan

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	sync "sync"
	"time"

	"github.com/mattn/go-isatty"
	"github.com/rs/zerolog/log"
	"github.com/segmentio/ksuid"
	"github.com/spf13/viper"
	"go.mondoo.com/cnquery/v10"
	"go.mondoo.com/cnquery/v10/cli/progress"
	"go.mondoo.com/cnquery/v10/explorer"
	"go.mondoo.com/cnquery/v10/explorer/executor"
	"go.mondoo.com/cnquery/v10/internal/datalakes/inmemory"
	"go.mondoo.com/cnquery/v10/llx"
	"go.mondoo.com/cnquery/v10/logger"
	"go.mondoo.com/cnquery/v10/mql"
	"go.mondoo.com/cnquery/v10/mrn"
	"go.mondoo.com/cnquery/v10/providers"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/inventory"
	"go.mondoo.com/cnquery/v10/providers-sdk/v1/upstream"
	"go.mondoo.com/cnquery/v10/utils/multierr"
	"go.mondoo.com/cnquery/v10/utils/slicesx"
	"go.mondoo.com/ranger-rpc/codes"
	"go.mondoo.com/ranger-rpc/status"
	"google.golang.org/protobuf/proto"
)

type LocalScanner struct {
	fetcher            *fetcher
	upstream           *upstream.UpstreamConfig
	recording          llx.Recording
	disableProgressBar bool
}

type ScannerOption func(*LocalScanner)

func WithUpstream(u *upstream.UpstreamConfig) func(s *LocalScanner) {
	return func(s *LocalScanner) {
		s.upstream = u
	}
}

func WithRecording(r llx.Recording) func(s *LocalScanner) {
	return func(s *LocalScanner) {
		s.recording = r
	}
}

func DisableProgressBar() ScannerOption {
	return func(s *LocalScanner) {
		s.disableProgressBar = true
	}
}

func NewLocalScanner(opts ...ScannerOption) *LocalScanner {
	ls := &LocalScanner{
		fetcher: newFetcher(),
	}

	for i := range opts {
		opts[i](ls)
	}

	if ls.recording == nil {
		ls.recording = providers.NullRecording{}
	}

	return ls
}

func (s *LocalScanner) Run(ctx context.Context, job *Job) (*explorer.ReportCollection, error) {
	if job == nil {
		return nil, status.Errorf(codes.InvalidArgument, "missing scan job")
	}

	if job.Inventory == nil {
		return nil, status.Errorf(codes.InvalidArgument, "missing inventory")
	}

	if ctx == nil {
		return nil, errors.New("no context provided to run job with local scanner")
	}

	upstreamConfig, err := s.getUpstreamConfig(job.Inventory, false)
	if err != nil {
		return nil, err
	}
	reports, err := s.distributeJob(job, ctx, upstreamConfig)
	if err != nil {
		if code := status.Code(err); code == codes.Unauthenticated {
			return nil, multierr.Wrap(err,
				"The Mondoo Platform credentials provided at "+viper.ConfigFileUsed()+
					" didn't successfully authenticate with Mondoo Platform. "+
					"Please re-authenticate with Mondoo Platform. "+
					"To learn how, read https://mondoo.com/docs/cnspec/cnspec-adv-install/registration/.")
		}
		return nil, err
	}

	return reports, nil
}

// returns the upstream config for the job. If the job has a specified config, it has precedence
// over the automatically detected one
func (s *LocalScanner) getUpstreamConfig(inv *inventory.Inventory, incognito bool) (*upstream.UpstreamConfig, error) {
	jobCreds := inv.GetSpec().GetUpstreamCredentials()
	if s.upstream == nil && jobCreds == nil {
		return nil, errors.New("no default or job upstream config provided")
	}
	u := proto.Clone(s.upstream).(*upstream.UpstreamConfig)
	u.Incognito = incognito
	if jobCreds != nil {
		u.ApiEndpoint = jobCreds.GetApiEndpoint()
		u.Creds = jobCreds
		u.SpaceMrn = jobCreds.GetParentMrn()
	}
	return u, nil
}

func (s *LocalScanner) RunIncognito(ctx context.Context, job *Job) (*explorer.ReportCollection, error) {
	if job == nil {
		return nil, status.Errorf(codes.InvalidArgument, "missing scan job")
	}

	if job.Inventory == nil {
		return nil, status.Errorf(codes.InvalidArgument, "missing inventory")
	}

	if ctx == nil {
		return nil, errors.New("no context provided to run job with local scanner")
	}

	// skip the error check, we are running in incognito
	upstreamConfig, _ := s.getUpstreamConfig(job.Inventory, true)
	reports, err := s.distributeJob(job, ctx, upstreamConfig)
	if err != nil {
		return nil, err
	}

	return reports, nil
}

// preprocessPolicyFilters expends short registry mrns into full mrns
func preprocessQueryPackFilters(filters []string) []string {
	res := make([]string, len(filters))
	for i := range filters {
		f := filters[i]
		if strings.HasPrefix(f, "//") {
			res[i] = f
			continue
		}

		// expand short registry mrns
		m := strings.Split(f, "/")
		if len(m) == 2 {
			res[i] = explorer.NewQueryPackMrn(m[0], m[1])
		} else {
			res[i] = f
		}
	}
	return res
}

func CreateProgressBar(discoveredAssets *DiscoveredAssets, disableProgressBar bool) (progress.MultiProgress, error) {
	var multiprogress progress.MultiProgress
	if isatty.IsTerminal(os.Stdout.Fd()) && !disableProgressBar && !strings.EqualFold(logger.GetLevel(), "debug") && !strings.EqualFold(logger.GetLevel(), "trace") {
		progressBarElements := map[string]string{}
		orderedKeys := []string{}
		for i := range discoveredAssets.Assets {
			asset := discoveredAssets.Assets[i].Asset
			// this shouldn't happen, but might
			// it normally indicates a bug in the provider
			if presentAsset, present := progressBarElements[asset.PlatformIds[0]]; present {
				return nil, fmt.Errorf("asset %s and %s have the same platform id %s", presentAsset, asset.Name, asset.PlatformIds[0])
			}
			progressBarElements[asset.PlatformIds[0]] = asset.Name
			orderedKeys = append(orderedKeys, asset.PlatformIds[0])
		}
		var err error
		multiprogress, err = progress.NewMultiProgressBars(progressBarElements, orderedKeys, progress.WithScore())
		if err != nil {
			return nil, multierr.Wrap(err, "failed to create progress bars")
		}
	} else {
		// TODO: adjust naming
		multiprogress = progress.NoopMultiProgressBars{}
	}
	return multiprogress, nil
}

func (s *LocalScanner) distributeJob(job *Job, ctx context.Context, upstream *upstream.UpstreamConfig) (*explorer.ReportCollection, error) {
	log.Info().Msgf("discover related assets for %d asset(s)", len(job.Inventory.Spec.Assets))

	discoveredAssets, err := DiscoverAssets(ctx, job.Inventory, upstream, s.recording)
	if err != nil {
		return nil, err
	}

	// For each discovered asset, we initialize a new runtime and connect to it.
	// Within this process, we set up a catch-all deferred function, that shuts
	// down all runtimes, in case we exit early.
	defer func() {
		for _, asset := range discoveredAssets.Assets {
			// we can call close multiple times and it will only execute once
			if asset.Runtime != nil {
				asset.Runtime.Close()
			}
		}
	}()

	// plan scan jobs
	reporter := NewAggregateReporter()
	// if we had asset errors we want to place them into the reporter
	for i := range discoveredAssets.Errors {
		reporter.AddScanError(discoveredAssets.Errors[i].Asset, discoveredAssets.Errors[i].Err)
	}

	if len(discoveredAssets.Assets) == 0 {
		return reporter.Reports(), nil
	}

	multiprogress, err := CreateProgressBar(discoveredAssets, s.disableProgressBar)
	if err != nil {
		return nil, err
	}
	// start the progress bar
	scanGroups := sync.WaitGroup{}
	scanGroups.Add(1)
	go func() {
		defer scanGroups.Done()
		if err := multiprogress.Open(); err != nil {
			log.Error().Err(err).Msg("failed to open progress bar")
		}
	}()

	assetBatches := slicesx.Batch(discoveredAssets.Assets, 100)
	for i := range assetBatches {
		batch := assetBatches[i]

		// sync assets
		if upstream != nil && upstream.ApiEndpoint != "" && !upstream.Incognito {
			log.Info().Msg("synchronize assets")
			client, err := upstream.InitClient()
			if err != nil {
				return nil, err
			}

			services, err := explorer.NewRemoteServices(client.ApiEndpoint, client.Plugins, client.HttpClient)
			if err != nil {
				return nil, err
			}

			assetsToSync := make([]*inventory.Asset, 0, len(batch))
			for i := range batch {
				assetsToSync = append(assetsToSync, batch[i].Asset)
			}

			resp, err := services.SynchronizeAssets(ctx, &explorer.SynchronizeAssetsReq{
				SpaceMrn: client.SpaceMrn,
				List:     assetsToSync,
			})
			if err != nil {
				return nil, err
			}
			log.Debug().Int("assets", len(resp.Details)).Msg("got assets details")
			platformAssetMapping := make(map[string]*explorer.SynchronizeAssetsRespAssetDetail)
			for i := range resp.Details {
				log.Debug().Str("platform-mrn", resp.Details[i].PlatformMrn).Str("asset", resp.Details[i].AssetMrn).Msg("asset mapping")
				platformAssetMapping[resp.Details[i].PlatformMrn] = resp.Details[i]
			}

			// attach the asset details to the assets list
			for i := range batch {
				asset := batch[i].Asset
				log.Debug().Str("asset", asset.Name).Strs("platform-ids", asset.PlatformIds).Msg("update asset")
				platformMrn := asset.PlatformIds[0]
				asset.Mrn = platformAssetMapping[platformMrn].AssetMrn
				asset.Url = platformAssetMapping[platformMrn].Url
			}
		} else {
			// ensure we have non-empty asset MRNs
			for i := range batch {
				asset := batch[i].Asset
				if asset.Mrn == "" {
					randID := "//" + explorer.SERVICE_NAME + "/" + explorer.MRN_RESOURCE_ASSET + "/" + ksuid.New().String()
					x, err := mrn.NewMRN(randID)
					if err != nil {
						return nil, multierr.Wrap(err, "failed to generate a random asset MRN")
					}
					asset.Mrn = x.String()
				}
			}
		}

		// if a bundle was provided check that it matches the filter, bundles can also be downloaded
		// later therefore we do not want to stop execution here
		if job.Bundle != nil && job.Bundle.FilterQueryPacks(job.QueryPackFilters) {
			return nil, errors.New("all available packs filtered out. nothing to do")
		}

		wg := sync.WaitGroup{}
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range batch {
				asset := batch[i].Asset
				runtime := batch[i].Runtime

				// Make sure the context has not been canceled in the meantime. Note that this approach works only for single threaded execution. If we have more than 1 thread calling this function,
				// we need to solve this at a different level.
				select {
				case <-ctx.Done():
					log.Warn().Msg("request context has been canceled")
					// When we scan concurrently, we need to call Errored(asset.Mrn) status for this asset
					multiprogress.Close()
					return
				default:
				}

				p := &progress.MultiProgressAdapter{Key: asset.PlatformIds[0], Multi: multiprogress}
				s.RunAssetJob(&AssetJob{
					DoRecord:         job.DoRecord,
					UpstreamConfig:   upstream,
					Asset:            asset,
					Bundle:           job.Bundle,
					Props:            job.Props,
					QueryPackFilters: preprocessQueryPackFilters(job.QueryPackFilters),
					Ctx:              ctx,
					Reporter:         reporter,
					ProgressReporter: p,
					runtime:          runtime,
				})

				// runtimes are single-use only. Close them once they are done.
				runtime.Close()
			}
		}()
		wg.Wait()
	}
	scanGroups.Wait()
	return reporter.Reports(), nil
}

func (s *LocalScanner) RunAssetJob(job *AssetJob) {
	log.Debug().Msgf("connecting to asset %s", job.Asset.HumanName())
	results, err := s.runMotorizedAsset(job)
	if err != nil {
		log.Debug().Err(err).Str("asset", job.Asset.Name).Msg("could not scan asset")
		job.Reporter.AddScanError(job.Asset, err)

		es := explorer.NewErrorStatus(err)
		if es.ErrorCode() == explorer.NotApplicable {
			job.ProgressReporter.NotApplicable()
		} else {
			job.ProgressReporter.Errored()
		}
		return
	}

	job.Reporter.AddReport(job.Asset, results)
}

func (s *LocalScanner) runMotorizedAsset(job *AssetJob) (*AssetReport, error) {
	var res *AssetReport
	var scanErr error

	runtimeErr := inmemory.WithDb(job.runtime, func(db *inmemory.Db, services *explorer.LocalServices) error {
		if job.UpstreamConfig != nil && job.UpstreamConfig.ApiEndpoint != "" && !job.UpstreamConfig.Incognito {
			log.Debug().Msg("using API endpoint " + job.UpstreamConfig.ApiEndpoint)
			client, err := job.UpstreamConfig.InitClient()
			if err != nil {
				return err
			}

			upstream, err := explorer.NewRemoteServices(client.ApiEndpoint, client.Plugins, client.HttpClient)
			if err != nil {
				return err
			}
			services.Upstream = upstream
		}

		scanner := &localAssetScanner{
			db:       db,
			services: services,
			job:      job,
			fetcher:  s.fetcher,
			Runtime:  job.runtime,
		}
		res, scanErr = scanner.run()
		return scanErr
	})
	if runtimeErr != nil {
		return res, runtimeErr
	}

	return res, scanErr
}

type localAssetScanner struct {
	db       *inmemory.Db
	services *explorer.LocalServices
	job      *AssetJob
	fetcher  *fetcher

	Runtime  llx.Runtime
	Progress progress.Progress
}

func (s *localAssetScanner) run() (*AssetReport, error) {
	if err := s.prepareAsset(); err != nil {
		return nil, err
	}

	res, err := s.runQueryPack()
	log.Debug().Str("asset", s.job.Asset.Mrn).Msg("scan complete")
	return res, err
}

func (s *localAssetScanner) prepareAsset() error {
	var hub explorer.QueryHub = s.services
	var conductor explorer.QueryConductor = s.services

	// if we are using upstream we get the bundle from there
	if s.job.UpstreamConfig != nil && !s.job.UpstreamConfig.Incognito {
		return nil
	}

	if err := s.ensureBundle(); err != nil {
		return err
	}

	if s.job.Bundle == nil {
		return errors.New("no bundle provided to run")
	}

	if len(s.job.Bundle.Packs) == 0 {
		return errors.New("bundle doesn't contain any query packs")
	}

	// FIXME: we do not currently respect bundle filters!
	_, err := hub.SetBundle(s.job.Ctx, s.job.Bundle)
	if err != nil {
		return err
	}

	querypackMrns := make([]string, len(s.job.Bundle.Packs))
	for i := range s.job.Bundle.Packs {
		querypackMrns[i] = s.job.Bundle.Packs[i].Mrn
	}

	_, err = conductor.Assign(s.job.Ctx, &explorer.Assignment{
		AssetMrn: s.job.Asset.Mrn,
		PackMrns: querypackMrns,
	})
	if err != nil {
		return err
	}

	if len(s.job.Props) != 0 {
		propsReq := explorer.PropsReq{
			EntityMrn: s.job.Asset.Mrn,
			Props:     make([]*explorer.Property, len(s.job.Props)),
		}
		i := 0
		for k, v := range s.job.Props {
			propsReq.Props[i] = &explorer.Property{
				Uid: k,
				Mql: v,
			}
			i++
		}

		_, err = conductor.SetProps(s.job.Ctx, &propsReq)
		if err != nil {
			return err
		}
	}

	return nil
}

var _assetDetectBundle *llx.CodeBundle

func assetDetectBundle() *llx.CodeBundle {
	if _assetDetectBundle == nil {
		// NOTE: we need to make sure this is loaded after the logger has been
		// initialized, otherwise the provider detection will print annoying logs
		_assetDetectBundle = executor.MustCompile("asset { kind platform runtime version family }")
	}
	return _assetDetectBundle
}

func (s *localAssetScanner) ensureBundle() error {
	if s.job.Bundle != nil {
		return nil
	}

	features := cnquery.GetFeatures(s.job.Ctx)
	res, err := mql.ExecuteCode(s.Runtime, assetDetectBundle(), nil, features)
	if err != nil {
		panic(err)
	}

	if err != nil {
		return multierr.Wrap(err, "failed to run asset detection query")
	}

	// FIXME: remove hardcoded lookup and use embedded datastructures instead
	data := res["IA0bVPKFxIh8Z735sqDh7bo/FNIYUQ/B4wLijN+YhiBZePu1x2sZCMcHoETmWM9jocdWbwGykKvNom/7QSm8ew=="].Data.Value.(map[string]interface{})
	kind := data["1oxYPIhW1eZ+14s234VsQ0Q7p9JSmUaT/RTWBtDRG1ZwKr8YjMcXz76x10J9iu13AcMmGZd43M1NNqPXZtTuKQ=="].(*llx.RawData).Value.(string)
	platform := data["W+8HW/v60Fx0nqrVz+yTIQjImy4ki4AiqxcedooTPP3jkbCESy77ptEhq9PlrKjgLafHFn8w4vrimU4bwCi6aQ=="].(*llx.RawData).Value.(string)
	runtime := data["a3RMPjrhk+jqkeXIISqDSi7EEP8QybcXCeefqNJYVUNcaDGcVDdONFvcTM2Wts8qTRXL3akVxpskitXWuI/gdA=="].(*llx.RawData).Value.(string)
	version := data["5d4FZxbPkZu02MQaHp3C356NJ9TeVsJBw8Enu+TDyBGdWlZM/AE+J5UT/TQ72AmDViKZe97Hxz1Jt3MjcEH/9Q=="].(*llx.RawData).Value.(string)
	fraw := data["l/aGjrixdNHvCxu5ib4NwkYb0Qrh3sKzcrGTkm7VxNWfWaaVbOxOEoGEMnjGJTo31jhYNeRm39/zpepZaSbUIw=="].(*llx.RawData).Value.([]interface{})
	family := make([]string, len(fraw))
	for i := range fraw {
		family[i] = fraw[i].(string)
	}

	var hub explorer.QueryHub = s.services
	urls, err := hub.DefaultPacks(s.job.Ctx, &explorer.DefaultPacksReq{
		Kind:     kind,
		Platform: platform,
		Runtime:  runtime,
		Version:  version,
		Family:   family,
	})
	if err != nil {
		return err
	}

	if len(urls.Urls) == 0 {
		return errors.New("cannot find any default policies for this asset (" + platform + ")")
	}

	s.job.Bundle, err = s.fetcher.fetchBundles(s.job.Ctx, s.Runtime.Schema(), urls.Urls...)
	if err != nil {
		return err
	}

	// filter bundle by ID
	if s.job.Bundle.FilterQueryPacks(s.job.QueryPackFilters) {
		return errors.New("all available packs filtered out. nothing to do.")
	}

	return err
}

func (s *localAssetScanner) runQueryPack() (*AssetReport, error) {
	var hub explorer.QueryHub = s.services
	var conductor explorer.QueryConductor = s.services

	log.Debug().Str("asset", s.job.Asset.Mrn).Msg("client> request bundle for asset")
	assetBundle, err := hub.GetBundle(s.job.Ctx, &explorer.Mrn{Mrn: s.job.Asset.Mrn})
	if err != nil {
		return nil, err
	}
	log.Debug().Msg("client> got bundle")
	logger.TraceJSON(assetBundle)
	logger.DebugDumpJSON("assetBundle", assetBundle)

	rawFilters, err := hub.GetFilters(s.job.Ctx, &explorer.Mrn{Mrn: s.job.Asset.Mrn})
	if err != nil {
		return nil, err
	}
	log.Debug().Str("asset", s.job.Asset.Mrn).Msg("client> got filters")
	logger.TraceJSON(rawFilters)

	filters, err := s.UpdateFilters(rawFilters, 5*time.Second)
	if err != nil {
		return nil, err
	}
	log.Debug().Str("asset", s.job.Asset.Mrn).Msg("client> shell update filters")
	logger.DebugJSON(filters)

	resolvedPack, err := conductor.Resolve(s.job.Ctx, &explorer.ResolveReq{
		EntityMrn:    s.job.Asset.Mrn,
		AssetFilters: filters,
	})
	if err != nil {
		return nil, err
	}
	log.Debug().Str("asset", s.job.Asset.Mrn).Msg("client> got resolved bundle for asset")
	logger.DebugDumpJSON("resolvedPack", resolvedPack)

	features := cnquery.GetFeatures(s.job.Ctx)
	e, err := executor.RunExecutionJob(s.Runtime, conductor, s.job.Asset.Mrn, resolvedPack.ExecutionJob, features, s.job.ProgressReporter)
	if err != nil {
		return nil, err
	}

	err = e.WaitUntilDone(10 * time.Second)
	if err != nil {
		return nil, err
	}

	err = e.StoreQueryData()
	if err != nil {
		return nil, err
	}

	if cnquery.GetFeatures(s.job.Ctx).IsActive(cnquery.StoreResourcesData) {
		recording := s.Runtime.Recording()
		data, ok := recording.GetAssetData(s.job.Asset.Mrn)
		if !ok {
			log.Debug().Msg("not storing resource data for this asset, nothing available")
		} else {
			_, err = conductor.StoreResults(context.Background(), &explorer.StoreResultsReq{
				AssetMrn:  s.job.Asset.Mrn,
				Resources: data,
			})
			if err != nil {
				return nil, err
			}
		}
	}

	ar := &AssetReport{
		Mrn:      s.job.Asset.Mrn,
		Bundle:   assetBundle,
		Resolved: resolvedPack,
	}

	log.Debug().Str("asset", s.job.Asset.Mrn).Msg("generate report")
	report, err := conductor.GetReport(s.job.Ctx, &explorer.EntityDataRequest{
		// NOTE: we assign packs to the asset before we execute the tests,
		// therefore this resolves all packs assigned to the asset
		EntityMrn: s.job.Asset.Mrn,
		DataMrn:   s.job.Asset.Mrn,
	})
	if err != nil {
		ar.Report = &explorer.Report{
			EntityMrn: s.job.Asset.Mrn,
			PackMrn:   s.job.Asset.Mrn,
		}
		return ar, err
	}

	ar.Report = report
	return ar, nil
}

// FilterQueries returns all queries whose result is truthy
func (s *localAssetScanner) FilterQueries(queries []*explorer.Mquery, timeout time.Duration) ([]*explorer.Mquery, []error) {
	return executor.ExecuteFilterQueries(s.Runtime, queries, timeout)
}

// UpdateFilters takes a list of test filters and runs them against the backend
// to return the matching ones
func (s *localAssetScanner) UpdateFilters(filters *explorer.Mqueries, timeout time.Duration) ([]*explorer.Mquery, error) {
	queries, errs := s.FilterQueries(filters.Items, timeout)

	var err error
	if len(errs) != 0 {
		w := strings.Builder{}
		for i := range errs {
			w.WriteString(errs[i].Error() + "\n")
		}
		err = errors.New("received multiple errors: " + w.String())
	}

	return queries, err
}
