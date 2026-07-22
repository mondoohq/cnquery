// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package mistralai

import (
	"encoding/json"
	"fmt"
)

type ModelList struct {
	Object string  `json:"object"`
	Data   []Model `json:"data"`
}

type Model struct {
	ID                          string            `json:"id"`
	Object                      string            `json:"object"`
	Created                     int64             `json:"created"`
	OwnedBy                     string            `json:"owned_by"`
	Capabilities                ModelCapabilities `json:"capabilities"`
	Name                        *string           `json:"name"`
	Description                 *string           `json:"description"`
	MaxContextLength            int64             `json:"max_context_length"`
	Aliases                     []string          `json:"aliases"`
	Deprecation                 *string           `json:"deprecation"`
	DeprecationReplacementModel *string           `json:"deprecation_replacement_model"`
	DefaultModelTemperature     *float64          `json:"default_model_temperature"`
	Type                        string            `json:"type"`
	// Fine-tuned model fields
	Job      string `json:"job"`
	Root     string `json:"root"`
	Archived bool   `json:"archived"`
}

type ModelCapabilities struct {
	CompletionChat     bool `json:"completion_chat"`
	FunctionCalling    bool `json:"function_calling"`
	CompletionFim      bool `json:"completion_fim"`
	FineTuning         bool `json:"fine_tuning"`
	Vision             bool `json:"vision"`
	OCR                bool `json:"ocr"`
	Classification     bool `json:"classification"`
	Moderation         bool `json:"moderation"`
	Audio              bool `json:"audio"`
	AudioTranscription bool `json:"audio_transcription"`
}

type FineTuningJob struct {
	ID              string                    `json:"id"`
	AutoStart       bool                      `json:"auto_start"`
	Model           string                    `json:"model"`
	Status          string                    `json:"status"`
	CreatedAt       int64                     `json:"created_at"`
	ModifiedAt      int64                     `json:"modified_at"`
	TrainingFiles   []string                  `json:"training_files"`
	ValidationFiles []string                  `json:"validation_files"`
	FineTunedModel  *string                   `json:"fine_tuned_model"`
	Suffix          *string                   `json:"suffix"`
	TrainedTokens   *int64                    `json:"trained_tokens"`
	Metadata        *FineTuningJobMetadata    `json:"metadata"`
	JobType         string                    `json:"job_type"`
	Hyperparameters FineTuningHyperparameters `json:"hyperparameters"`
	Integrations    []WandbIntegration        `json:"integrations"`
}

type FineTuningHyperparameters struct {
	TrainingSteps  *int64   `json:"training_steps"`
	LearningRate   float64  `json:"learning_rate"`
	WeightDecay    *float64 `json:"weight_decay"`
	WarmupFraction *float64 `json:"warmup_fraction"`
	Epochs         *float64 `json:"epochs"`
	SeqLen         *int64   `json:"seq_len"`
	FimRatio       *float64 `json:"fim_ratio"`
}

type FineTuningJobMetadata struct {
	ExpectedDurationSeconds *int64   `json:"expected_duration_seconds"`
	Cost                    *float64 `json:"cost"`
	CostCurrency            *string  `json:"cost_currency"`
	TrainTokensPerStep      *int64   `json:"train_tokens_per_step"`
	TrainTokens             *int64   `json:"train_tokens"`
	DataTokens              *int64   `json:"data_tokens"`
	EstimatedStartTime      *int64   `json:"estimated_start_time"`
}

type WandbIntegration struct {
	Type    string  `json:"type"`
	Project string  `json:"project"`
	Name    *string `json:"name"`
	RunName *string `json:"run_name"`
	URL     *string `json:"url"`
}

type File struct {
	ID         string  `json:"id"`
	Object     string  `json:"object"`
	Bytes      int64   `json:"bytes"`
	CreatedAt  int64   `json:"created_at"`
	Filename   string  `json:"filename"`
	Purpose    string  `json:"purpose"`
	SampleType string  `json:"sample_type"`
	Source     string  `json:"source"`
	NumLines   *int64  `json:"num_lines"`
	MimeType   *string `json:"mimetype"`
}

type BatchJob struct {
	ID                string            `json:"id"`
	Object            string            `json:"object"`
	InputFiles        []string          `json:"input_files"`
	Endpoint          string            `json:"endpoint"`
	Model             *string           `json:"model"`
	OutputFile        *string           `json:"output_file"`
	ErrorFile         *string           `json:"error_file"`
	Errors            []BatchError      `json:"errors"`
	Status            string            `json:"status"`
	CreatedAt         int64             `json:"created_at"`
	TotalRequests     int64             `json:"total_requests"`
	CompletedRequests int64             `json:"completed_requests"`
	SucceededRequests int64             `json:"succeeded_requests"`
	FailedRequests    int64             `json:"failed_requests"`
	StartedAt         *int64            `json:"started_at"`
	CompletedAt       *int64            `json:"completed_at"`
	Metadata          map[string]string `json:"metadata"`
}

type BatchError struct {
	Message string `json:"message"`
	Count   int    `json:"count"`
}

type APIError struct {
	Message string `json:"message"`
	// Detail carries validation-style error payloads (HTTP 422), which Mistral
	// returns under "detail" instead of "message" (either a string or an array
	// of objects). Kept raw so Error() can surface something meaningful rather
	// than an empty string.
	Detail     json.RawMessage `json:"detail"`
	StatusCode int             `json:"-"`
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	if len(e.Detail) > 0 {
		var s string
		if err := json.Unmarshal(e.Detail, &s); err == nil {
			return s
		}
		return string(e.Detail)
	}
	return fmt.Sprintf("mistral API error (status %d)", e.StatusCode)
}
