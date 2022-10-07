package scan

import (
	"context"
	"strings"
	"time"

	"github.com/gogo/status"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/segmentio/ksuid"
	"go.mondoo.com/cnquery"
	"go.mondoo.com/cnquery/cli/progress"
	"go.mondoo.com/cnquery/explorer"
	"go.mondoo.com/cnquery/explorer/executor"
	"go.mondoo.com/cnquery/internal/datalakes/inmemory"
	"go.mondoo.com/cnquery/logger"
	"go.mondoo.com/cnquery/motor"
	"go.mondoo.com/cnquery/motor/discovery"
	"go.mondoo.com/cnquery/motor/inventory"
	"go.mondoo.com/cnquery/motor/providers/resolver"
	"go.mondoo.com/cnquery/mrn"
	"go.mondoo.com/cnquery/resources"
	"go.mondoo.com/cnquery/resources/packs/all"
	"google.golang.org/grpc/codes"
)

type LocalScanner struct {
	ctx context.Context
}

func NewLocalScanner() *LocalScanner {
	return &LocalScanner{}
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

	if job.Bundle == nil {
		return nil, errors.New("no bundle provided to run")
	}

	if len(job.Bundle.Packs) == 0 {
		return nil, errors.New("bundle doesn't contain any query packs")
	}

	if job.Bundle.OwnerMrn == "" {
		job.Bundle.OwnerMrn = "//bundle.api.mondoo.app"
	}

	dctx := discovery.InitCtx(ctx)

	reports, _, err := s.distributeJob(job, dctx)
	if err != nil {
		return nil, err
	}

	return reports, nil
}

func (s *LocalScanner) distributeJob(job *Job, ctx context.Context) (*explorer.ReportCollection, bool, error) {
	log.Info().Msgf("discover related assets for %d asset(s)", len(job.Inventory.Spec.Assets))
	im, err := inventory.New(inventory.WithInventory(job.Inventory))
	if err != nil {
		return nil, false, errors.Wrap(err, "could not load asset information")
	}

	assetErrors := im.Resolve(ctx)
	if len(assetErrors) > 0 {
		for a := range assetErrors {
			log.Error().Err(assetErrors[a]).Str("asset", a.Name).Msg("could not resolve asset")
		}
		return nil, false, errors.New("failed to resolve multiple assets")
	}

	assetList := im.GetAssets()
	if len(assetList) == 0 {
		return nil, false, errors.New("could not find an asset that we can connect to")
	}

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

	reporter := NewAggregateReporter(job.Bundle, assetList)

	job.Bundle.FilterQueryPacks(job.QueryPackFilters)

	for i := range assetList {
		asset := assetList[i]

		// Make sure the context has not been canceled in the meantime. Note that this approach works only for single threaded execution. If we have more than 1 thread calling this function,
		// we need to solve this at a different level.
		select {
		case <-ctx.Done():
			log.Warn().Msg("request context has been canceled")
			return reporter.Reports(), false, reporter.Error()
		default:
		}

		s.RunAssetJob(&AssetJob{
			DoRecord:         job.DoRecord,
			Asset:            asset,
			Bundle:           job.Bundle,
			QueryPackFilters: job.QueryPackFilters,
			Ctx:              ctx,
			GetCredential:    im.GetCredential,
			Reporter:         reporter,
		})
	}

	return reporter.Reports(), true, reporter.Error()
}

func (s *LocalScanner) RunAssetJob(job *AssetJob) {
	log.Info().Msgf("connecting to asset %s", job.Asset.HumanName())

	// run over all connections
	connections, err := resolver.OpenAssetConnections(job.Ctx, job.Asset, job.GetCredential, job.DoRecord)
	if err != nil {
		job.Reporter.AddScanError(job.Asset, err)
		return
	}

	for c := range connections {
		// We use a function since we want to close the motor once the current iteration finishes. If we directly
		// use defer in the loop m.Close() for each connection will only be executed once the entire loop is
		// finished.
		func(m *motor.Motor) {
			// ensures temporary files get deleted
			defer m.Close()

			log.Debug().Msg("established connection")
			// It's possible that the platform information was not collected at all or only partially during the
			// discovery phase.
			// For example, the ebs discovery does not detect the platform because it requires mounting
			// the filesystem. Another example is the docker container discovery, where it collects a lot of metadata
			// but does not have platform name and arch available.
			// TODO: It feels like this will only happen for performance optimizations. I think a better approach
			// would be to make it so that the motor used in the discovery phase gets reused here, instead
			// of being recreated.
			if job.Asset.Platform == nil || job.Asset.Platform.Name == "" {
				p, err := m.Platform()
				if err != nil {
					log.Warn().Err(err).Msg("failed to query platform information")
				} else {
					job.Asset.Platform = p
					// resyncAssets = append(resyncAssets, assetEntry)
				}
			}

			job.connection = m
			results, err := s.runMotorizedAsset(job)
			if err != nil {
				job.Reporter.AddScanError(job.Asset, err)
				return
			}

			job.Reporter.AddReport(job.Asset, results)
		}(connections[c])
	}
}

func (s *LocalScanner) runMotorizedAsset(job *AssetJob) (*AssetReport, error) {
	var res *AssetReport
	var scanErr error

	runtimeErr := inmemory.WithDb(func(db *inmemory.Db, services *explorer.LocalServices) error {
		if services.Upstream != nil {
			panic("cannot work with upstream yet")
		}

		registry := all.Registry
		schema := registry.Schema()
		runtime := resources.NewRuntime(registry, job.connection)

		scanner := &localAssetScanner{
			db:       db,
			services: services,
			job:      job,
			Registry: registry,
			Schema:   schema,
			Runtime:  runtime,
			Progress: progress.New(job.Asset.Mrn, job.Asset.Name),
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

	Registry *resources.Registry
	Schema   *resources.Schema
	Runtime  *resources.Runtime
	Progress progress.Progress
}

func (s *localAssetScanner) run() (*AssetReport, error) {
	s.Progress.Open()

	// fallback to always close the progressbar if we error before getting the report
	defer s.Progress.Close()

	if err := s.prepareAsset(); err != nil {
		return nil, err
	}

	bundle, report, err := s.runQueryPack()
	if err != nil {
		return nil, err
	}

	log.Debug().Str("asset", s.job.Asset.Mrn).Msg("scan complete")
	return &AssetReport{
		Mrn:    s.job.Asset.Mrn,
		Bundle: bundle,
		Report: report,
	}, nil
}

func (s *localAssetScanner) prepareAsset() error {
	var hub explorer.QueryHub = s.services
	var conductor explorer.QueryConductor = s.services

	if err := s.ensureBundle(); err != nil {
		return err
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

	return err
}

func (s *localAssetScanner) ensureBundle() error {
	if s.job.Bundle != nil {
		return nil
	}

	return errors.New("Default Query Packs are NOT YET IMPLEMENTED")
}

func (s *localAssetScanner) runQueryPack() (*explorer.Bundle, *explorer.Report, error) {
	var hub explorer.QueryHub = s.services
	var conductor explorer.QueryConductor = s.services

	log.Debug().Str("asset", s.job.Asset.Mrn).Msg("client> request bundle for asset")
	assetBundle, err := hub.GetBundle(s.job.Ctx, &explorer.Mrn{Mrn: s.job.Asset.Mrn})
	if err != nil {
		return nil, nil, err
	}
	log.Debug().Msg("client> got bundle")
	logger.TraceJSON(assetBundle)
	logger.DebugDumpJSON("assetBundle", assetBundle)

	rawFilters, err := hub.GetFilters(s.job.Ctx, &explorer.Mrn{Mrn: s.job.Asset.Mrn})
	if err != nil {
		return nil, nil, err
	}
	log.Debug().Str("asset", s.job.Asset.Mrn).Msg("client> got filters")
	logger.TraceJSON(rawFilters)

	filters, err := s.UpdateFilters(rawFilters, 5*time.Second)
	if err != nil {
		return s.job.Bundle, nil, err
	}
	log.Debug().Str("asset", s.job.Asset.Mrn).Msg("client> shell update filters")
	logger.DebugJSON(filters)

	resolvedPack, err := conductor.Resolve(s.job.Ctx, &explorer.ResolveReq{
		EntityMrn:    s.job.Asset.Mrn,
		AssetFilters: filters,
	})
	if err != nil {
		return s.job.Bundle, nil, err
	}
	log.Debug().Str("asset", s.job.Asset.Mrn).Msg("client> got resolved bundle for asset")
	logger.DebugDumpJSON("resolvedPack", resolvedPack)

	features := cnquery.GetFeatures(s.job.Ctx)
	e, err := executor.RunExecutionJob(s.Schema, s.Runtime, conductor, s.job.Asset.Mrn, resolvedPack.ExecutionJob, features, s.Progress)
	if err != nil {
		return s.job.Bundle, nil, err
	}

	err = e.WaitUntilDone(10 * time.Second)
	if err != nil {
		return s.job.Bundle, nil, err
	}

	log.Debug().Str("asset", s.job.Asset.Mrn).Msg("generate report")
	report, err := conductor.GetReport(s.job.Ctx, &explorer.EntityDataRequest{
		// NOTE: we assign packs to the asset before we execute the tests,
		// therefore this resolves all packs assigned to the asset
		EntityMrn: s.job.Asset.Mrn,
		DataMrn:   s.job.Asset.Mrn,
	})
	if err != nil {
		return s.job.Bundle, nil, err
	}

	return s.job.Bundle, report, nil
}

// FilterQueries returns all queries whose result is truthy
func (s *localAssetScanner) FilterQueries(queries []*explorer.Mquery, timeout time.Duration) ([]*explorer.Mquery, []error) {
	return executor.RunFilterQueries(s.Schema, s.Runtime, queries, timeout)
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
