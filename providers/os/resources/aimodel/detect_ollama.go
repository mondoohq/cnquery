// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package aimodel

import (
	"encoding/json"
	"path/filepath"
	"strings"

	"github.com/spf13/afero"
)

// OllamaDetector discovers locally cached Ollama models by walking the manifest
// directory (~/.ollama/models/manifests). Each manifest JSON references a
// config blob that contains the model family, quantization (file_type),
// parameter size (model_type), architecture, and license. The vendor is
// inferred from a prefix lookup table mapping model names to known publishers.
type OllamaDetector struct{}

type ollamaManifest struct {
	Config ollamaDescriptor `json:"config"`
	Layers []ollamaLayer    `json:"layers"`
}

type ollamaDescriptor struct {
	Digest string `json:"digest"`
}

type ollamaLayer struct {
	MediaType string `json:"mediaType"`
	Digest    string `json:"digest"`
	Size      int64  `json:"size"`
}

type ollamaExtracted struct {
	Family        string
	Architecture  string
	ParameterSize string
	License       string
	Quantization  string
}

// ollamaVendorPrefixes maps model name prefixes to vendor names.
// Lookup uses longest-prefix match so "deepseek-r1" matches "deepseek".
var ollamaVendorPrefixes = map[string]string{
	"llama":       "Meta",
	"codellama":   "Meta",
	"gemma":       "Google",
	"codegemma":   "Google",
	"shieldgemma": "Google",
	"medgemma":    "Google",
	"qwen":        "Alibaba",
	"qwq":         "Alibaba",
	"codeqwen":    "Alibaba",
	"marco-o1":    "Alibaba",
	"deepseek":    "DeepSeek",
	"mistral":     "Mistral AI",
	"mixtral":     "Mistral AI",
	"codestral":   "Mistral AI",
	"devstral":    "Mistral AI",
	"magistral":   "Mistral AI",
	"mathstral":   "Mistral AI",
	"ministral":   "Mistral AI",
	"phi":         "Microsoft",
	"wizardlm":    "Microsoft",
	"orca":        "Microsoft",
	"gpt-oss":     "OpenAI",
	"glm":         "Z.AI",
	"codegeex":    "Z.AI",
	"command-r":   "Cohere",
	"command-a":   "Cohere",
	"aya":         "Cohere",
	"grok":        "xAI",
	"yi":          "01.AI",
	"jamba":       "AI21 Labs",
	"granite":     "IBM",
	"falcon":      "TII",
	"nemotron":    "NVIDIA",
	"solar":       "Upstage",
	"dbrx":        "Databricks",
	"starcoder":   "BigCode",
	"stable":      "Stability AI",
	"olmo":        "Allen Institute for AI",
	"tulu":        "Allen Institute for AI",
	"smollm":      "Hugging Face",
	"nomic":       "Nomic",
	"snowflake":   "Snowflake",
	"internlm":    "Shanghai AI Lab",
	"minicpm":     "OpenBMB",
	"kimi":        "Moonshot AI",
	"minimax":     "MiniMax",
	"exaone":      "LG AI Research",
	"dolphin":     "Cognitive Computations",
	"tinydolphin": "Cognitive Computations",
	"hermes":      "Nous Research",
	"tinyllama":   "TinyLlama",
	"llava":       "LLaVA",
	"bakllava":    "LLaVA",
	"moondream":   "Moondream AI",
	"neural-chat": "Intel",
	"sailor":      "Sea AI Lab",
	"cogito":      "Deep Cogito",
	"lfm":         "Liquid AI",
	"reader-lm":   "Jina AI",
	"baichuan":    "Baichuan",
	"rwkv":        "RWKV Foundation",
}

func ollamaVendor(modelBase string) string {
	best := ""
	for prefix := range ollamaVendorPrefixes {
		if strings.HasPrefix(modelBase, prefix) && len(prefix) > len(best) {
			best = prefix
		}
	}
	if best != "" {
		return ollamaVendorPrefixes[best]
	}
	return ""
}

func (d *OllamaDetector) Detect(ctx DetectContext) []ModelInfo {
	modelsDir := filepath.Join(ctx.Home, ".ollama", "models")
	manifestsDir := filepath.Join(modelsDir, "manifests")

	// Ollama manifests follow a 4-level structure: registry/namespace/model/tag
	// (e.g. registry.ollama.ai/library/llama3/latest).
	// Walk each level explicitly to avoid unbounded traversal.
	registries, err := ctx.Fs.ReadDir(manifestsDir)
	if err != nil {
		return nil
	}

	var results []ModelInfo
	for _, registry := range registries {
		if !registry.IsDir() {
			continue
		}
		registryDir := filepath.Join(manifestsDir, registry.Name())
		namespaces, err := ctx.Fs.ReadDir(registryDir)
		if err != nil {
			continue
		}
		for _, ns := range namespaces {
			if !ns.IsDir() {
				continue
			}
			nsDir := filepath.Join(registryDir, ns.Name())
			models, err := ctx.Fs.ReadDir(nsDir)
			if err != nil {
				continue
			}
			for _, model := range models {
				if !model.IsDir() {
					continue
				}
				modelBase := model.Name()
				modelDir := filepath.Join(nsDir, modelBase)
				tags, err := ctx.Fs.ReadDir(modelDir)
				if err != nil {
					continue
				}
				for _, tag := range tags {
					if tag.IsDir() {
						continue
					}
					tagPath := filepath.Join(modelDir, tag.Name())
					data, err := ctx.Fs.ReadFile(tagPath)
					if err != nil {
						continue
					}

					var manifest ollamaManifest
					if json.Unmarshal(data, &manifest) != nil || len(manifest.Layers) == 0 {
						continue
					}

					name := modelBase + ":" + tag.Name()

					var totalSize int64
					for _, l := range manifest.Layers {
						totalSize += l.Size
					}

					extracted := readOllamaConfig(ctx.Fs, modelsDir, manifest.Config.Digest)

					version := tag.Name()

					quant := extracted.Quantization
					if quant == "" {
						if m := reQuantization.FindString(tag.Name()); m != "" {
							quant = strings.ToUpper(m)
						}
					}

					paramSize := extracted.ParameterSize
					if paramSize == "" {
						if m := reParamSize.FindStringSubmatch(name); len(m) > 1 {
							paramSize = m[1] + "B"
						}
					}

					// Build tags from tag name parts (split on -)
					var modelTags []string
					for _, part := range strings.Split(tag.Name(), "-") {
						if part != "" && part != "latest" {
							modelTags = append(modelTags, part)
						}
					}

					results = append(results, ModelInfo{
						Name:          name,
						Source:        "ollama",
						Vendor:        ollamaVendor(modelBase),
						Family:        extracted.Family,
						Path:          tagPath,
						Size:          totalSize,
						ModifiedAt:    tag.ModTime(),
						Format:        "gguf",
						Version:       version,
						Quantization:  quant,
						ParameterSize: paramSize,
						Architecture:  extracted.Architecture,
						License:       extracted.License,
						Tags:          modelTags,
					})
				}
			}
		}
	}
	return results
}

func readOllamaConfig(afs *afero.Afero, modelsDir string, digest string) ollamaExtracted {
	var result ollamaExtracted
	if digest == "" {
		return result
	}
	blobName := strings.Replace(digest, ":", "-", 1)
	blobPath := filepath.Join(modelsDir, "blobs", blobName)
	data, err := afs.ReadFile(blobPath)
	if err != nil {
		return result
	}

	var raw map[string]any
	if json.Unmarshal(data, &raw) != nil {
		return result
	}

	if v, ok := raw["model_family"].(string); ok {
		result.Family = v
	}
	// file_type holds quantization (e.g. "Q4_0", "Q8_0")
	if v, ok := raw["file_type"].(string); ok {
		result.Quantization = v
	}
	// model_type holds human-readable parameter size (e.g. "8.0B", "70B")
	if v, ok := raw["model_type"].(string); ok && v != "" {
		result.ParameterSize = v
	}
	if v, ok := raw["license"].(string); ok {
		result.License = v
	}
	// general.architecture is the model arch in some blobs
	if v, ok := raw["general.architecture"].(string); ok {
		result.Architecture = v
	}
	// Fallback: use model_family as architecture if general.architecture is absent
	// (don't use "architecture" — that's the platform arch like "amd64")
	if result.Architecture == "" {
		result.Architecture = result.Family
	}

	return result
}
