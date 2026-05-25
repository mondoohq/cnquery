// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package resources

import (
	"context"
	"fmt"
	"time"

	"github.com/openai/openai-go/v3"
	"go.mondoo.com/mql/v13/llx"
	"go.mondoo.com/mql/v13/providers-sdk/v1/plugin"
)

type mqlOpenaiFineTuningJobInternal struct {
	cacheTrainingFileID   string
	cacheValidationFileID string
}

func (r *mqlOpenai) fineTuningJobs() ([]any, error) {
	conn := openaiConn(r.MqlRuntime)
	client := conn.Client()
	if client == nil {
		return []any{}, nil
	}
	ctx := context.Background()

	iter := client.FineTuning.Jobs.ListAutoPaging(ctx, openai.FineTuningJobListParams{})
	var res []any
	for iter.Next() {
		j := iter.Current()
		created := unixToTime(j.CreatedAt)

		var finishedAt *time.Time
		if j.FinishedAt != 0 {
			t := unixToTime(j.FinishedAt)
			finishedAt = &t
		}

		var trainedTokens int64
		if j.TrainedTokens != 0 {
			trainedTokens = j.TrainedTokens
		}

		var nEpochs any
		if j.Hyperparameters.NEpochs.OfAuto != "" {
			nEpochs = string(j.Hyperparameters.NEpochs.OfAuto)
		} else {
			nEpochs = j.Hyperparameters.NEpochs.OfInt
		}
		hyperparams := map[string]any{
			"n_epochs": nEpochs,
		}

		var errInfo any
		if j.Error.Code != "" {
			errInfo = map[string]any{
				"code":    j.Error.Code,
				"message": j.Error.Message,
				"param":   j.Error.Param,
			}
		}

		mqlJob, err := CreateResource(r.MqlRuntime, "openai.fineTuningJob", map[string]*llx.RawData{
			"__id":            llx.StringData(j.ID),
			"id":              llx.StringData(j.ID),
			"model":           llx.StringData(j.Model),
			"status":          llx.StringData(string(j.Status)),
			"createdAt":       llx.TimeData(created),
			"finishedAt":      llx.TimeDataPtr(finishedAt),
			"fineTunedModel":  llx.StringData(j.FineTunedModel),
			"trainedTokens":   llx.IntData(trainedTokens),
			"seed":            llx.IntData(j.Seed),
			"organizationId":  llx.StringData(j.OrganizationID),
			"hyperparameters": llx.DictData(hyperparams),
			"error":           llx.DictData(errInfo),
		})
		if err != nil {
			return nil, err
		}

		job := mqlJob.(*mqlOpenaiFineTuningJob)
		job.cacheTrainingFileID = j.TrainingFile
		job.cacheValidationFileID = j.ValidationFile

		res = append(res, mqlJob)
	}
	if err := iter.Err(); err != nil {
		if isAccessDenied(err) {
			return []any{}, nil
		}
		return nil, fmt.Errorf("failed to list fine-tuning jobs: %w", err)
	}
	return res, nil
}

func (r *mqlOpenaiFineTuningJob) trainingFile() (*mqlOpenaiFile, error) {
	if r.cacheTrainingFileID == "" {
		r.TrainingFile.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	res, err := NewResource(r.MqlRuntime, "openai.file", map[string]*llx.RawData{
		"__id": llx.StringData(r.cacheTrainingFileID),
		"id":   llx.StringData(r.cacheTrainingFileID),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlOpenaiFile), nil
}

func (r *mqlOpenaiFineTuningJob) validationFile() (*mqlOpenaiFile, error) {
	if r.cacheValidationFileID == "" {
		r.ValidationFile.State = plugin.StateIsNull | plugin.StateIsSet
		return nil, nil
	}
	res, err := NewResource(r.MqlRuntime, "openai.file", map[string]*llx.RawData{
		"__id": llx.StringData(r.cacheValidationFileID),
		"id":   llx.StringData(r.cacheValidationFileID),
	})
	if err != nil {
		return nil, err
	}
	return res.(*mqlOpenaiFile), nil
}
