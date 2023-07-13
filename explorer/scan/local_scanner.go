package scan

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	sync "sync"
	"time"

	"github.com/mattn/go-isatty"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/segmentio/ksuid"
	"github.com/spf13/viper"
	"go.mondoo.com/cnquery"
	"go.mondoo.com/cnquery/cli/progress"
	"go.mondoo.com/cnquery/explorer"
	"go.mondoo.com/cnquery/explorer/executor"
	"go.mondoo.com/cnquery/internal/datalakes/inmemory"
	"go.mondoo.com/cnquery/llx"
	"go.mondoo.com/cnquery/logger"
	"go.mondoo.com/cnquery/mql"
	"go.mondoo.com/cnquery/mrn"
	"go.mondoo.com/cnquery/providers"
	"go.mondoo.com/cnquery/providers-sdk/v1/inventory/manager"
	"go.mondoo.com/ranger-rpc"
	"go.mondoo.com/ranger-rpc/codes"
	"go.mondoo.com/ranger-rpc/status"
)

type LocalScanner struct {
	ctx     context.Context
	fetcher *fetcher

	// for remote connectivity
	apiEndpoint string
	spaceMrn    string
	plugins     []ranger.ClientPlugin
	httpClient  *http.Client
}

type ScannerOption func(*LocalScanner)

func WithUpstream(apiEndpoint string, spaceMrn string, plugins []ranger.ClientPlugin, httpClient *http.Client) func(s *LocalScanner) {
	if httpClient == nil {
		httpClient = ranger.DefaultHttpClient()
	}
	return func(s *LocalScanner) {
		s.apiEndpoint = apiEndpoint
		s.plugins = plugins
		s.spaceMrn = spaceMrn
		s.httpClient = httpClient
	}
}

func NewLocalScanner(opts ...ScannerOption) *LocalScanner {
	ls := &LocalScanner{
		fetcher: newFetcher(),
	}

	for i := range opts {
		opts[i](ls)
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

	upstreamConfig := providers.UpstreamConfig{
		SpaceMrn:    s.spaceMrn,
		ApiEndpoint: s.apiEndpoint,
		Incognito:   false,
		Plugins:     s.plugins,
	}

	reports, _, err := s.distributeJob(job, ctx, upstreamConfig)
	if err != nil {
		if code := status.Code(err); code == codes.Unauthenticated {
			return nil, errors.Wrapf(err,
				"The Mondoo Platform credentials provided at %s didn't successfully authenticate with Mondoo Platform. Please re-authenticate with Mondoo Platform. To learn how, read https://mondoo.com/docs/cnspec/cnspec-adv-install/registration.",
				viper.ConfigFileUsed())
		}
		return nil, err
	}

	return reports, nil
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

	upstreamConfig := providers.UpstreamConfig{
		Incognito: true,
	}

	reports, _, err := s.distributeJob(job, ctx, upstreamConfig)
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

func (s *LocalScanner) distributeJob(job *Job, ctx context.Context, upstreamConfig providers.UpstreamConfig) (*explorer.ReportCollection, bool, error) {
	log.Info().Msgf("discover related assets for %d asset(s)", len(job.Inventory.Spec.Assets))
	im, err := manager.NewManager(manager.WithInventory(job.Inventory))
	if err != nil {
		return nil, false, errors.Wrap(err, "could not load asset information")
	}

	panic("IMPLEMENT RESOLVE")
	// assetErrors := im.Resolve(ctx)
	// if len(assetErrors) > 0 {
	// 	for a := range assetErrors {
	// 		log.Error().Err(assetErrors[a]).Str("asset", a.Name).Msg("could not resolve asset")
	// 	}
	// 	return nil, false, errors.New("failed to resolve multiple assets")
	// }

	assetList := im.GetAssets()
	if len(assetList) == 0 {
		return nil, false, errors.New("could not find an asset that we can connect to")
	}

	// sync assets
	if upstreamConfig.ApiEndpoint != "" && !upstreamConfig.Incognito {
		log.Info().Msg("synchronize assets")
		upstream, err := explorer.NewRemoteServices(s.apiEndpoint, s.plugins, s.httpClient)
		if err != nil {
			return nil, false, err
		}
		resp, err := upstream.SynchronizeAssets(ctx, &explorer.SynchronizeAssetsReq{
			SpaceMrn: s.spaceMrn,
			List:     assetList,
		})
		if err != nil {
			return nil, false, err
		}
		log.Debug().Int("assets", len(resp.Details)).Msg("got assets details")
		platformAssetMapping := make(map[string]*explorer.SynchronizeAssetsRespAssetDetail)
		for i := range resp.Details {
			log.Debug().Str("platform-mrn", resp.Details[i].PlatformMrn).Str("asset", resp.Details[i].AssetMrn).Msg("asset mapping")
			platformAssetMapping[resp.Details[i].PlatformMrn] = resp.Details[i]
		}

		// attach the asset details to the assets list
		for i := range assetList {
			log.Debug().Str("asset", assetList[i].Name).Strs("platform-ids", assetList[i].PlatformIds).Msg("update asset")
			platformMrn := assetList[i].PlatformIds[0]
			assetList[i].Mrn = platformAssetMapping[platformMrn].AssetMrn
			assetList[i].Url = platformAssetMapping[platformMrn].Url
		}
	} else {
		// ensure we have non-empty asset MRNs
		for i := range assetList {
			cur := assetList[i]
			if cur.Mrn == "" && cur.Id == "" {
				randID := "//" + explorer.SERVICE_NAME + "/" + explorer.MRN_RESOURCE_ASSET + "/" + ksuid.New().String()
				x, err := mrn.NewMRN(randID)
				if err != nil {
					return nil, false, errors.Wrap(err, "failed to generate a random asset MRN")
				}
				cur.Mrn = x.String()
			}
		}
	}

	// plan scan jobs
	reporter := NewAggregateReporter(assetList)
	// if a bundle was provided check that it matches the filter, bundles can also be downloaded
	// later therefore we do not want to stop execution here
	if job.Bundle != nil && job.Bundle.FilterQueryPacks(job.QueryPackFilters) {
		return nil, false, errors.New("all available packs filtered out. nothing to do.")
	}

	progressBarElements := map[string]string{}
	orderedKeys := []string{}
	for i := range assetList {
		// this shouldn't happen, but might
		// it normally indicates a bug in the provider
		if presentAsset, present := progressBarElements[assetList[i].PlatformIds[0]]; present {
			return nil, false, fmt.Errorf("asset %s and %s have the same platform id %s", presentAsset, assetList[i].Name, assetList[i].PlatformIds[0])
		}
		progressBarElements[assetList[i].PlatformIds[0]] = assetList[i].Name
		orderedKeys = append(orderedKeys, assetList[i].PlatformIds[0])
	}
	var multiprogress progress.MultiProgress
	if isatty.IsTerminal(os.Stdout.Fd()) && !strings.EqualFold(logger.GetLevel(), "debug") && !strings.EqualFold(logger.GetLevel(), "trace") {
		multiprogress, err = progress.NewMultiProgressBars(progressBarElements, orderedKeys)
		if err != nil {
			return nil, false, errors.Wrap(err, "failed to create progress bars")
		}
	} else {
		// TODO: adjust naming
		multiprogress = progress.NoopMultiProgressBars{}
	}

	scanGroup := sync.WaitGroup{}
	scanGroup.Add(1)
	finished := false
	go func() {
		defer scanGroup.Done()
		for i := range assetList {
			asset := assetList[i]

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
				UpstreamConfig:   upstreamConfig,
				Asset:            asset,
				Bundle:           job.Bundle,
				Props:            job.Props,
				QueryPackFilters: preprocessQueryPackFilters(job.QueryPackFilters),
				Ctx:              ctx,
				CredsResolver:    im.GetCredsResolver(),
				Reporter:         reporter,
				ProgressReporter: p,
			})
		}
		finished = true
	}()

	scanGroup.Add(1)
	go func() {
		defer scanGroup.Done()
		multiprogress.Open()
	}()
	scanGroup.Wait()
	return reporter.Reports(), finished, nil
}

func (s *LocalScanner) RunAssetJob(job *AssetJob) {
	log.Debug().Msgf("connecting to asset %s", job.Asset.HumanName())

	// run over all connections
	panic("NEEDS MIGRATION")
	// runtimes, err := resolver.OpenAssetConnections(job.Ctx, job.Asset, job.CredsResolver, job.DoRecord)
	var runtimes []*providers.Runtime
	var err error
	if err != nil {
		job.Reporter.AddScanError(job.Asset, err)
		es := explorer.NewErrorStatus(err)
		if es.ErrorCode() == explorer.NotApplicable {
			job.ProgressReporter.NotApplicable()
		} else {
			job.ProgressReporter.Errored()
		}
		return
	}

	for c := range runtimes {
		// We use a function since we want to close the motor once the current iteration finishes. If we directly
		// use defer in the loop m.Close() for each connection will only be executed once the entire loop is
		// finished.
		func(m *providers.Runtime) {
			// ensures temporary files get deleted
			defer m.Close()

			log.Debug().Msg("established connection")

			job.runtime = m
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
		}(runtimes[c])
	}
}

func (s *LocalScanner) runMotorizedAsset(job *AssetJob) (*AssetReport, error) {
	var res *AssetReport
	var scanErr error

	runtimeErr := inmemory.WithDb(func(db *inmemory.Db, services *explorer.LocalServices) error {
		if job.UpstreamConfig.ApiEndpoint != "" && !job.UpstreamConfig.Incognito {
			log.Debug().Msg("using API endpoint " + s.apiEndpoint)
			upstream, err := explorer.NewRemoteServices(s.apiEndpoint, s.plugins, s.httpClient)
			if err != nil {
				return err
			}
			services.Upstream = upstream
		}

		runtime := providers.Coordinator.NewRuntime()
		// runtime := resources.NewRuntime(registry, job.connection)
		panic("PORT SCANNER")
		runtime.UpstreamConfig = &job.UpstreamConfig

		scanner := &localAssetScanner{
			db:       db,
			services: services,
			job:      job,
			fetcher:  s.fetcher,
			Runtime:  runtime,
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
	if !s.job.UpstreamConfig.Incognito {
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

var assetDetectBundle = executor.MustCompile("asset { kind platform runtime version family }")

func (s *localAssetScanner) ensureBundle() error {
	if s.job.Bundle != nil {
		return nil
	}

	features := cnquery.GetFeatures(s.job.Ctx)
	res, err := mql.ExecuteCode(s.Runtime, assetDetectBundle, nil, features)
	if err != nil {
		panic(err)
	}

	if err != nil {
		return errors.Wrap(err, "failed to run asset detection query")
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

	s.job.Bundle, err = s.fetcher.fetchBundles(s.job.Ctx, urls.Urls...)
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

	err = e.StoreData()
	if err != nil {
		return nil, err
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
	return executor.RunFilterQueries(s.Runtime, queries, timeout)
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
