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

func initVirustotalHash(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if _, ok := args["hash"]; !ok {
		return nil, nil, errors.New("missing required argument 'hash'")
	}
	return args, nil, nil
}

func (r *mqlVirustotalHash) id() (string, error) {
	if !r.Hash.IsSet() || r.Hash.Data == "" {
		return "", errors.New("hash not set")
	}
	return ResourceVirustotalHash + "/" + strings.ToLower(r.Hash.Data), nil
}

func (r *mqlVirustotalHash) detections() (int64, error) {
	if err := r.fetchHashData(); err != nil {
		return 0, err
	}
	return r.Detections.Data, r.Detections.Error
}

func (r *mqlVirustotalHash) lastAnalysisStats() (map[string]any, error) {
	if err := r.fetchHashData(); err != nil {
		return nil, err
	}
	return r.LastAnalysisStats.Data, r.LastAnalysisStats.Error
}

func (r *mqlVirustotalHash) lastAnalysisDate() (*time.Time, error) {
	if err := r.fetchHashData(); err != nil {
		return nil, err
	}
	return r.LastAnalysisDate.Data, r.LastAnalysisDate.Error
}

func (r *mqlVirustotalHash) fetchHashData() error {
	if r.Detections.IsSet() {
		return r.Detections.Error
	}

	if !r.Hash.IsSet() || r.Hash.Data == "" {
		err := errors.New("hash not set")
		r.setHashError(err)
		return err
	}

	hash := strings.ToLower(r.Hash.Data)

	conn := r.MqlRuntime.Connection.(*connection.VirustotalConnection)
	client := conn.Client()
	if client == nil {
		err := errors.New("cannot retrieve new data while using a mock connection")
		r.setHashError(err)
		return err
	}

	obj, err := client.GetObject(vt.URL("files/%s", hash))
	if err != nil {
		if isNotFoundError(err) {
			r.setHashNoData()
			return nil
		}
		r.setHashError(err)
		return err
	}

	r.populateHash(obj)
	return nil
}

func (r *mqlVirustotalHash) setHashError(err error) {
	r.Detections = plugin.TValue[int64]{State: plugin.StateIsSet | plugin.StateIsNull, Error: err}
	r.LastAnalysisStats = plugin.TValue[map[string]any]{State: plugin.StateIsSet | plugin.StateIsNull, Error: err}
	r.LastAnalysisDate = plugin.TValue[*time.Time]{State: plugin.StateIsSet | plugin.StateIsNull, Error: err}
}

func (r *mqlVirustotalHash) setHashNoData() {
	r.Detections = plugin.TValue[int64]{State: plugin.StateIsSet | plugin.StateIsNull}
	r.LastAnalysisStats = plugin.TValue[map[string]any]{State: plugin.StateIsSet | plugin.StateIsNull}
	r.LastAnalysisDate = plugin.TValue[*time.Time]{State: plugin.StateIsSet | plugin.StateIsNull}
}

func (r *mqlVirustotalHash) populateHash(obj *vt.Object) {
	stats, detections, isNull, err := analysisStatsAttribute(obj)
	if err != nil {
		r.LastAnalysisStats = plugin.TValue[map[string]any]{State: plugin.StateIsSet | plugin.StateIsNull, Error: err}
		r.Detections = plugin.TValue[int64]{State: plugin.StateIsSet | plugin.StateIsNull, Error: err}
	} else if isNull {
		r.LastAnalysisStats = plugin.TValue[map[string]any]{State: plugin.StateIsSet | plugin.StateIsNull}
		r.Detections = plugin.TValue[int64]{State: plugin.StateIsSet | plugin.StateIsNull}
	} else {
		r.LastAnalysisStats = plugin.TValue[map[string]any]{Data: stats, State: plugin.StateIsSet}
		r.Detections = plugin.TValue[int64]{Data: detections, State: plugin.StateIsSet}
	}

	if ts, err := obj.GetTime("last_analysis_date"); err == nil && !ts.IsZero() {
		t := ts
		r.LastAnalysisDate = plugin.TValue[*time.Time]{Data: &t, State: plugin.StateIsSet}
	} else {
		r.LastAnalysisDate = plugin.TValue[*time.Time]{State: plugin.StateIsSet | plugin.StateIsNull}
	}
}
