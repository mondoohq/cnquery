// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"regexp"
	"strings"
	"time"

	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
	"go.mondoo.com/mql/v13/providers/mistral/connection"
	"go.mondoo.com/mql/v13/providers/mistral/internal/mistralai"
	"go.mondoo.com/mql/v13/types"
)

func mistralConn(runtime *plugin.Runtime) *connection.MistralConnection {
	return runtime.Connection.(*connection.MistralConnection)
}

func (r *mqlMistral) id() (string, error) {
	return "mistral", nil
}

func (r *mqlMistral) ownedBy() (string, error) {
	conn := mistralConn(r.MqlRuntime)
	workspace := conn.Workspace()
	if workspace != "" {
		return workspace, nil
	}
	return "mistralai", nil
}

func (r *mqlMistral) models() ([]interface{}, error) {
	conn := mistralConn(r.MqlRuntime)
	client := conn.Client()

	resp, err := client.ListModels(context.Background())
	if mistralai.IsAccessDenied(err) {
		return []interface{}{}, nil
	}
	if err != nil {
		return nil, err
	}

	res := make([]interface{}, 0, len(resp.Data))
	for _, m := range resp.Data {
		var created *time.Time
		if m.Created > 0 {
			t := time.Unix(m.Created, 0)
			created = &t
		}

		var deprecation *time.Time
		if m.Deprecation != nil && *m.Deprecation != "" {
			if t, err := time.Parse(time.RFC3339, *m.Deprecation); err == nil {
				deprecation = &t
			}
		}

		var defaultTemp float64
		if m.DefaultModelTemperature != nil {
			defaultTemp = *m.DefaultModelTemperature
		}

		name := ""
		if m.Name != nil {
			name = *m.Name
		}
		description := ""
		if m.Description != nil {
			description = *m.Description
		}

		aliases := make([]interface{}, 0, len(m.Aliases))
		for _, a := range m.Aliases {
			aliases = append(aliases, a)
		}

		mqlModel, err := CreateResource(r.MqlRuntime, "mistral.model", map[string]*llx.RawData{
			"__id":                         llx.StringData(m.ID),
			"id":                           llx.StringData(m.ID),
			"type":                         llx.StringData(m.Type),
			"ownedBy":                      llx.StringData(m.OwnedBy),
			"name":                         llx.StringData(name),
			"description":                  llx.StringData(description),
			"maxContextLength":             llx.IntData(m.MaxContextLength),
			"created":                      llx.TimeDataPtr(created),
			"aliases":                      llx.ArrayData(aliases, types.String),
			"deprecation":                  llx.TimeDataPtr(deprecation),
			"defaultModelTemperature":      llx.FloatData(defaultTemp),
			"capabilityChat":               llx.BoolData(m.Capabilities.CompletionChat),
			"capabilityFunctionCalling":    llx.BoolData(m.Capabilities.FunctionCalling),
			"capabilityFim":                llx.BoolData(m.Capabilities.CompletionFim),
			"capabilityFineTuning":         llx.BoolData(m.Capabilities.FineTuning),
			"capabilityVision":             llx.BoolData(m.Capabilities.Vision),
			"capabilityOcr":                llx.BoolData(m.Capabilities.OCR),
			"capabilityClassification":     llx.BoolData(m.Capabilities.Classification),
			"capabilityModeration":         llx.BoolData(m.Capabilities.Moderation),
			"capabilityAudio":              llx.BoolData(m.Capabilities.Audio),
			"capabilityAudioTranscription": llx.BoolData(m.Capabilities.AudioTranscription),
			"job":                          llx.StringData(m.Job),
			"root":                         llx.StringData(m.Root),
			"archived":                     llx.BoolData(m.Archived),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlModel)
	}

	return res, nil
}

func (r *mqlMistral) fineTuningJobs() ([]interface{}, error) {
	conn := mistralConn(r.MqlRuntime)
	client := conn.Client()

	jobs, err := client.ListFineTuningJobs(context.Background())
	if mistralai.IsAccessDenied(err) {
		return []interface{}{}, nil
	}
	if err != nil {
		return nil, err
	}

	res := make([]interface{}, 0, len(jobs))
	for _, j := range jobs {
		createdAt := timeFromUnix(j.CreatedAt)
		modifiedAt := timeFromUnix(j.ModifiedAt)

		fineTunedModel := ""
		if j.FineTunedModel != nil {
			fineTunedModel = *j.FineTunedModel
		}
		suffix := ""
		if j.Suffix != nil {
			suffix = *j.Suffix
		}
		var trainedTokens int64
		if j.TrainedTokens != nil {
			trainedTokens = *j.TrainedTokens
		}

		trainingFiles := make([]interface{}, 0, len(j.TrainingFiles))
		for _, f := range j.TrainingFiles {
			trainingFiles = append(trainingFiles, f)
		}
		validationFiles := make([]interface{}, 0, len(j.ValidationFiles))
		for _, f := range j.ValidationFiles {
			validationFiles = append(validationFiles, f)
		}

		var trainingSteps int64
		if j.Hyperparameters.TrainingSteps != nil {
			trainingSteps = *j.Hyperparameters.TrainingSteps
		}
		var epochs float64
		if j.Hyperparameters.Epochs != nil {
			epochs = *j.Hyperparameters.Epochs
		}

		var expectedDuration int64
		var cost float64
		var costCurrency string
		if j.Metadata != nil {
			if j.Metadata.ExpectedDurationSeconds != nil {
				expectedDuration = *j.Metadata.ExpectedDurationSeconds
			}
			if j.Metadata.Cost != nil {
				cost = *j.Metadata.Cost
			}
			if j.Metadata.CostCurrency != nil {
				costCurrency = *j.Metadata.CostCurrency
			}
		}

		mqlJob, err := CreateResource(r.MqlRuntime, "mistral.fineTuningJob", map[string]*llx.RawData{
			"__id":                    llx.StringData(j.ID),
			"id":                      llx.StringData(j.ID),
			"status":                  llx.StringData(j.Status),
			"model":                   llx.StringData(j.Model),
			"fineTunedModel":          llx.StringData(fineTunedModel),
			"suffix":                  llx.StringData(suffix),
			"autoStart":               llx.BoolData(j.AutoStart),
			"trainingFiles":           llx.ArrayData(trainingFiles, types.String),
			"validationFiles":         llx.ArrayData(validationFiles, types.String),
			"trainedTokens":           llx.IntData(trainedTokens),
			"createdAt":               llx.TimeDataPtr(createdAt),
			"modifiedAt":              llx.TimeDataPtr(modifiedAt),
			"jobType":                 llx.StringData(j.JobType),
			"trainingSteps":           llx.IntData(trainingSteps),
			"learningRate":            llx.FloatData(j.Hyperparameters.LearningRate),
			"epochs":                  llx.FloatData(epochs),
			"expectedDurationSeconds": llx.IntData(expectedDuration),
			"cost":                    llx.FloatData(cost),
			"costCurrency":            llx.StringData(costCurrency),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlJob)
	}

	return res, nil
}

func (r *mqlMistral) files() ([]interface{}, error) {
	conn := mistralConn(r.MqlRuntime)
	client := conn.Client()

	files, err := client.ListFiles(context.Background())
	if mistralai.IsAccessDenied(err) {
		return []interface{}{}, nil
	}
	if err != nil {
		return nil, err
	}

	res := make([]interface{}, 0, len(files))
	for _, f := range files {
		createdAt := timeFromUnix(f.CreatedAt)

		var numLines int64
		if f.NumLines != nil {
			numLines = *f.NumLines
		}
		mimeType := ""
		if f.MimeType != nil {
			mimeType = *f.MimeType
		}

		mqlFile, err := CreateResource(r.MqlRuntime, "mistral.file", map[string]*llx.RawData{
			"__id":       llx.StringData(f.ID),
			"id":         llx.StringData(f.ID),
			"filename":   llx.StringData(f.Filename),
			"purpose":    llx.StringData(f.Purpose),
			"bytes":      llx.IntData(f.Bytes),
			"createdAt":  llx.TimeDataPtr(createdAt),
			"sampleType": llx.StringData(f.SampleType),
			"source":     llx.StringData(f.Source),
			"numLines":   llx.IntData(numLines),
			"mimeType":   llx.StringData(mimeType),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlFile)
	}

	return res, nil
}

func (r *mqlMistral) batchJobs() ([]interface{}, error) {
	conn := mistralConn(r.MqlRuntime)
	client := conn.Client()

	batches, err := client.ListBatchJobs(context.Background())
	if mistralai.IsAccessDenied(err) {
		return []interface{}{}, nil
	}
	if err != nil {
		return nil, err
	}

	res := make([]interface{}, 0, len(batches))
	for _, b := range batches {
		createdAt := timeFromUnix(b.CreatedAt)
		startedAt := timeFromUnixPtr(b.StartedAt)
		completedAt := timeFromUnixPtr(b.CompletedAt)

		model := ""
		if b.Model != nil {
			model = *b.Model
		}
		outputFile := ""
		if b.OutputFile != nil {
			outputFile = *b.OutputFile
		}
		errorFile := ""
		if b.ErrorFile != nil {
			errorFile = *b.ErrorFile
		}

		inputFiles := make([]interface{}, 0, len(b.InputFiles))
		for _, f := range b.InputFiles {
			inputFiles = append(inputFiles, f)
		}

		mqlBatch, err := CreateResource(r.MqlRuntime, "mistral.batchJob", map[string]*llx.RawData{
			"__id":              llx.StringData(b.ID),
			"id":                llx.StringData(b.ID),
			"status":            llx.StringData(b.Status),
			"endpoint":          llx.StringData(b.Endpoint),
			"model":             llx.StringData(model),
			"inputFiles":        llx.ArrayData(inputFiles, types.String),
			"outputFile":        llx.StringData(outputFile),
			"errorFile":         llx.StringData(errorFile),
			"totalRequests":     llx.IntData(b.TotalRequests),
			"completedRequests": llx.IntData(b.CompletedRequests),
			"succeededRequests": llx.IntData(b.SucceededRequests),
			"failedRequests":    llx.IntData(b.FailedRequests),
			"createdAt":         llx.TimeDataPtr(createdAt),
			"startedAt":         llx.TimeDataPtr(startedAt),
			"completedAt":       llx.TimeDataPtr(completedAt),
		})
		if err != nil {
			return nil, err
		}
		res = append(res, mqlBatch)
	}

	return res, nil
}

func (r *mqlMistralModel) id() (string, error) {
	return r.Id.Data, nil
}

var (
	mistralParamSizeRe = regexp.MustCompile(`[\-_](\d+(?:\.\d+)?(?:[xX]\d+)?[bBmM])[\-_]?`)
)

// mistralFamilies is ordered most-specific first so that e.g. "codestral"
// matches before the shorter "mistral" suffix it shares.
var mistralFamilies = []struct {
	substring string
	family    string
}{
	{"codestral", "Codestral"},
	{"devstral", "Devstral"},
	{"leanstral", "Leanstral"},
	{"magistral", "Magistral"},
	{"mathstral", "Mathstral"},
	{"ministral", "Ministral"},
	{"mixtral", "Mixtral"},
	{"pixtral", "Pixtral"},
	{"nemo", "Nemo"},
	{"embed", "Embed"},
	{"moderation", "Moderation"},
	{"mistral", "Mistral"},
}

func matchFamily(id string) string {
	lower := strings.ToLower(id)
	for _, f := range mistralFamilies {
		if strings.Contains(lower, f.substring) {
			return f.family
		}
	}
	return ""
}

func (r *mqlMistralModel) family() (string, error) {
	if f := matchFamily(r.Id.Data); f != "" {
		return f, nil
	}
	if root := r.Root.Data; root != "" {
		if f := matchFamily(root); f != "" {
			return f, nil
		}
	}
	return "", nil
}

func (r *mqlMistralModel) parameterSize() (string, error) {
	id := r.Id.Data
	m := mistralParamSizeRe.FindStringSubmatch(id)
	if m == nil {
		// Try root model for fine-tuned models
		root := r.Root.Data
		if root != "" {
			m = mistralParamSizeRe.FindStringSubmatch(root)
		}
	}
	if m == nil {
		return "", nil
	}
	size := m[1]
	last := size[len(size)-1]
	if last >= 'a' && last <= 'z' {
		size = size[:len(size)-1] + strings.ToUpper(string(last))
	}
	return size, nil
}

func (r *mqlMistralFineTuningJob) id() (string, error) {
	return r.Id.Data, nil
}

func (r *mqlMistralFile) id() (string, error) {
	return r.Id.Data, nil
}

func (r *mqlMistralBatchJob) id() (string, error) {
	return r.Id.Data, nil
}

func timeFromUnix(ts int64) *time.Time {
	if ts == 0 {
		return nil
	}
	t := time.Unix(ts, 0)
	return &t
}

func timeFromUnixPtr(ts *int64) *time.Time {
	if ts == nil || *ts == 0 {
		return nil
	}
	t := time.Unix(*ts, 0)
	return &t
}

var _ = plugin.StateIsNull
