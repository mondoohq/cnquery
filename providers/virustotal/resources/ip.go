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

func initVirustotalIp(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if _, ok := args["address"]; !ok {
		return nil, nil, errors.New("missing required argument 'address'")
	}
	return args, nil, nil
}

func (r *mqlVirustotalIp) id() (string, error) {
	if !r.Address.IsSet() || r.Address.Data == "" {
		return "", errors.New("ip address not set")
	}
	return ResourceVirustotalIp + "/" + strings.ToLower(r.Address.Data), nil
}

func (r *mqlVirustotalIp) reputation() (int64, error) {
	if err := r.fetchIPData(); err != nil {
		return 0, err
	}
	return r.Reputation.Data, r.Reputation.Error
}

func (r *mqlVirustotalIp) country() (string, error) {
	if err := r.fetchIPData(); err != nil {
		return "", err
	}
	return r.Country.Data, r.Country.Error
}

func (r *mqlVirustotalIp) asOwner() (string, error) {
	if err := r.fetchIPData(); err != nil {
		return "", err
	}
	return r.AsOwner.Data, r.AsOwner.Error
}

func (r *mqlVirustotalIp) network() (string, error) {
	if err := r.fetchIPData(); err != nil {
		return "", err
	}
	return r.Network.Data, r.Network.Error
}

func (r *mqlVirustotalIp) lastAnalysisStats() (map[string]any, error) {
	if err := r.fetchIPData(); err != nil {
		return nil, err
	}
	return r.LastAnalysisStats.Data, r.LastAnalysisStats.Error
}

func (r *mqlVirustotalIp) lastAnalysisDate() (*time.Time, error) {
	if err := r.fetchIPData(); err != nil {
		return nil, err
	}
	return r.LastAnalysisDate.Data, r.LastAnalysisDate.Error
}

func (r *mqlVirustotalIp) fetchIPData() error {
	if r.Reputation.IsSet() {
		return r.Reputation.Error
	}

	if !r.Address.IsSet() || r.Address.Data == "" {
		err := errors.New("ip address not set")
		r.setIPError(err)
		return err
	}

	conn := r.MqlRuntime.Connection.(*connection.VirustotalConnection)
	client := conn.Client()
	if client == nil {
		err := errors.New("cannot retrieve new data while using a mock connection")
		r.setIPError(err)
		return err
	}

	obj, err := client.GetObject(vt.URL("ip_addresses/%s", r.Address.Data))
	if err != nil {
		r.setIPError(err)
		return err
	}

	r.populateIP(obj)
	return nil
}

func (r *mqlVirustotalIp) setIPError(err error) {
	r.Reputation = plugin.TValue[int64]{State: plugin.StateIsSet | plugin.StateIsNull, Error: err}
	r.Country = plugin.TValue[string]{State: plugin.StateIsSet | plugin.StateIsNull, Error: err}
	r.AsOwner = plugin.TValue[string]{State: plugin.StateIsSet | plugin.StateIsNull, Error: err}
	r.Network = plugin.TValue[string]{State: plugin.StateIsSet | plugin.StateIsNull, Error: err}
	r.LastAnalysisStats = plugin.TValue[map[string]any]{State: plugin.StateIsSet | plugin.StateIsNull, Error: err}
	r.LastAnalysisDate = plugin.TValue[*time.Time]{State: plugin.StateIsSet | plugin.StateIsNull, Error: err}
}

func (r *mqlVirustotalIp) populateIP(obj *vt.Object) {
	if rep, err := obj.GetInt64("reputation"); err == nil {
		r.Reputation = plugin.TValue[int64]{Data: rep, State: plugin.StateIsSet}
	} else {
		r.Reputation = plugin.TValue[int64]{State: plugin.StateIsSet | plugin.StateIsNull}
	}

	if country, err := obj.GetString("country"); err == nil && country != "" {
		r.Country = plugin.TValue[string]{Data: country, State: plugin.StateIsSet}
	} else {
		r.Country = plugin.TValue[string]{State: plugin.StateIsSet | plugin.StateIsNull}
	}

	if owner, err := obj.GetString("as_owner"); err == nil && owner != "" {
		r.AsOwner = plugin.TValue[string]{Data: owner, State: plugin.StateIsSet}
	} else {
		r.AsOwner = plugin.TValue[string]{State: plugin.StateIsSet | plugin.StateIsNull}
	}

	if network, err := obj.GetString("network"); err == nil && network != "" {
		r.Network = plugin.TValue[string]{Data: network, State: plugin.StateIsSet}
	} else {
		r.Network = plugin.TValue[string]{State: plugin.StateIsSet | plugin.StateIsNull}
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
