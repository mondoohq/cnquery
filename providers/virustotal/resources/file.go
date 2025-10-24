package resources

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"strings"
	"time"

	vt "github.com/VirusTotal/vt-go"
	"go.mondoo.com/cnquery/v12/llx"
	"go.mondoo.com/cnquery/v12/providers-sdk/v1/plugin"
	"go.mondoo.com/cnquery/v12/providers/virustotal/connection"
)

func initVirustotalFile(runtime *plugin.Runtime, args map[string]*llx.RawData) (map[string]*llx.RawData, plugin.Resource, error) {
	if _, ok := args["path"]; !ok {
		return nil, nil, errors.New("missing required argument 'path'")
	}
	return args, nil, nil
}

func (r *mqlVirustotalFile) id() (string, error) {
	if !r.Path.IsSet() || r.Path.Data == "" {
		return "", errors.New("file path not set")
	}

	sum := sha256.Sum256([]byte(r.Path.Data))
	return ResourceVirustotalFile + "/" + hex.EncodeToString(sum[:]), nil
}

func (r *mqlVirustotalFile) hash() (string, error) {
	if err := r.fetchFileData(); err != nil {
		return "", err
	}
	return r.Hash.Data, r.Hash.Error
}

func (r *mqlVirustotalFile) detections() (int64, error) {
	if err := r.fetchFileData(); err != nil {
		return 0, err
	}
	return r.Detections.Data, r.Detections.Error
}

func (r *mqlVirustotalFile) lastAnalysisStats() (map[string]any, error) {
	if err := r.fetchFileData(); err != nil {
		return nil, err
	}
	return r.LastAnalysisStats.Data, r.LastAnalysisStats.Error
}

func (r *mqlVirustotalFile) lastAnalysisDate() (*time.Time, error) {
	if err := r.fetchFileData(); err != nil {
		return nil, err
	}
	return r.LastAnalysisDate.Data, r.LastAnalysisDate.Error
}

func (r *mqlVirustotalFile) fetchFileData() error {
	if r.Detections.IsSet() {
		return r.Detections.Error
	}

	if !r.Path.IsSet() || r.Path.Data == "" {
		err := errors.New("file path not set")
		r.setFileError(err)
		return err
	}

	hash, err := computeFileHash(r.Path.Data)
	if err != nil {
		r.setFileError(err)
		return err
	}

	r.Hash = plugin.TValue[string]{Data: hash, State: plugin.StateIsSet}

	conn := r.MqlRuntime.Connection.(*connection.VirustotalConnection)
	client := conn.Client()
	if client == nil {
		err := errors.New("cannot retrieve new data while using a mock connection")
		r.setFileError(err)
		return err
	}

	obj, err := client.GetObject(vt.URL("files/%s", hash))
	if err != nil {
		if isNotFoundError(err) {
			r.setFileNoData()
			return nil
		}
		r.setFileError(err)
		return err
	}

	r.populateFile(obj)
	return nil
}

func (r *mqlVirustotalFile) setFileError(err error) {
	r.Hash = plugin.TValue[string]{State: plugin.StateIsSet | plugin.StateIsNull, Error: err}
	r.Detections = plugin.TValue[int64]{State: plugin.StateIsSet | plugin.StateIsNull, Error: err}
	r.LastAnalysisStats = plugin.TValue[map[string]any]{State: plugin.StateIsSet | plugin.StateIsNull, Error: err}
	r.LastAnalysisDate = plugin.TValue[*time.Time]{State: plugin.StateIsSet | plugin.StateIsNull, Error: err}
}

func (r *mqlVirustotalFile) setFileNoData() {
	r.Detections = plugin.TValue[int64]{State: plugin.StateIsSet | plugin.StateIsNull}
	r.LastAnalysisStats = plugin.TValue[map[string]any]{State: plugin.StateIsSet | plugin.StateIsNull}
	r.LastAnalysisDate = plugin.TValue[*time.Time]{State: plugin.StateIsSet | plugin.StateIsNull}
}

func (r *mqlVirustotalFile) populateFile(obj *vt.Object) {
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

func computeFileHash(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	sum := sha256.New()
	if _, err := io.Copy(sum, file); err != nil {
		return "", err
	}

	return strings.ToLower(hex.EncodeToString(sum.Sum(nil))), nil
}
