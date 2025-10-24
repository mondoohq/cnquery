package resources

import (
	"errors"
	"strings"
	"time"

	vt "github.com/VirusTotal/vt-go"
	"go.mondoo.com/cnquery/v12/llx"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v12/providers/virustotal/connection"
)

func initVirustotalDomain(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if _, ok := args["name"]; !ok {
		return nil, nil, errors.New("missing required argument 'name'")
	}
	return args, nil, nil
}

func (r *mqlVirustotalDomain) id() (string, error) {
	if !r.Name.IsSet() || r.Name.Data == "" {
		return "", errors.New("domain name not set")
	}
	return ResourceVirustotalDomain + "/" + strings.ToLower(r.Name.Data), nil
}

func (r *mqlVirustotalDomain) reputation() (int64, error) {
	if err := r.fetchDomainData(); err != nil {
		return 0, err
	}
	return r.Reputation.Data, r.Reputation.Error
}

func (r *mqlVirustotalDomain) categories() (map[string]any, error) {
	if err := r.fetchDomainData(); err != nil {
		return nil, err
	}
	return r.Categories.Data, r.Categories.Error
}

func (r *mqlVirustotalDomain) lastAnalysisStats() (map[string]any, error) {
	if err := r.fetchDomainData(); err != nil {
		return nil, err
	}
	return r.LastAnalysisStats.Data, r.LastAnalysisStats.Error
}

func (r *mqlVirustotalDomain) lastAnalysisDate() (*time.Time, error) {
	if err := r.fetchDomainData(); err != nil {
		return nil, err
	}
	return r.LastAnalysisDate.Data, r.LastAnalysisDate.Error
}

func (r *mqlVirustotalDomain) fetchDomainData() error {
	if r.Reputation.IsSet() {
		return r.Reputation.Error
	}

	if !r.Name.IsSet() || r.Name.Data == "" {
		err := errors.New("domain name not set")
		r.setDomainError(err)
		return err
	}

	conn := r.MqlRuntime.Connection.(*connection.VirustotalConnection)
	client := conn.Client()
	if client == nil {
		err := errors.New("cannot retrieve new data while using a mock connection")
		r.setDomainError(err)
		return err
	}

	obj, err := client.GetObject(vt.URL("domains/%s", r.Name.Data))
	if err != nil {
		r.setDomainError(err)
		return err
	}

	r.populateDomain(obj)
	return nil
}

func (r *mqlVirustotalDomain) setDomainError(err error) {
	r.Reputation = plugin.TValue[int64]{State: plugin.StateIsSet | plugin.StateIsNull, Error: err}
	r.Categories = plugin.TValue[map[string]any]{State: plugin.StateIsSet | plugin.StateIsNull, Error: err}
	r.LastAnalysisStats = plugin.TValue[map[string]any]{State: plugin.StateIsSet | plugin.StateIsNull, Error: err}
	r.LastAnalysisDate = plugin.TValue[*time.Time]{State: plugin.StateIsSet | plugin.StateIsNull, Error: err}
}

func (r *mqlVirustotalDomain) populateDomain(obj *vt.Object) {
	if rep, err := obj.GetInt64("reputation"); err == nil {
		r.Reputation = plugin.TValue[int64]{Data: rep, State: plugin.StateIsSet}
	} else {
		r.Reputation = plugin.TValue[int64]{State: plugin.StateIsSet | plugin.StateIsNull}
	}

	categories, isNullCategories, err := stringMapAttribute(obj, "categories")
	if err != nil {
		r.Categories = plugin.TValue[map[string]any]{State: plugin.StateIsSet | plugin.StateIsNull, Error: err}
	} else if isNullCategories {
		r.Categories = plugin.TValue[map[string]any]{State: plugin.StateIsSet | plugin.StateIsNull}
	} else {
		r.Categories = plugin.TValue[map[string]any]{Data: categories, State: plugin.StateIsSet}
	}

	stats, _, isNullStats, err := analysisStatsAttribute(obj)
	if err != nil {
		r.LastAnalysisStats = plugin.TValue[map[string]any]{State: plugin.StateIsSet | plugin.StateIsNull, Error: err}
	} else if isNullStats {
		r.LastAnalysisStats = plugin.TValue[map[string]any]{State: plugin.StateIsSet | plugin.StateIsNull}
	} else {
		r.LastAnalysisStats = plugin.TValue[map[string]any]{Data: stats, State: plugin.StateIsSet}
	}

	if ts, err := obj.GetTime("last_analysis_date"); err == nil && !ts.IsZero() {
		t := ts
		r.LastAnalysisDate = plugin.TValue[*time.Time]{Data: &t, State: plugin.StateIsSet}
	} else {
		r.LastAnalysisDate = plugin.TValue[*time.Time]{State: plugin.StateIsSet | plugin.StateIsNull}
	}
}
