// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package aimodel

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/spf13/afero"
	"gopkg.in/yaml.v3"
)

// ModelInfo holds the metadata for a single discovered AI model cache entry.
type ModelInfo struct {
	Name          string
	Source        string
	Vendor        string
	Family        string
	Path          string
	Size          int64
	ModifiedAt    time.Time
	Format        string
	Version       string
	Quantization  string
	ParameterSize string
	Architecture  string
	License       string
	Tags          []string
	Description   string
}

var (
	reQuantization = regexp.MustCompile(`(?i)(Q[0-9]+_[A-Z0-9_]+|F16|F32|FP16|FP32)`)
	reParamSize    = regexp.MustCompile(`(?i)[-_: ](\d+\.?\d*)[bB](?:[-_. ]|$)`)
)

// ScanAll runs every scanner and returns the combined results.
func ScanAll(afs *afero.Afero, home, osFamily string) []ModelInfo {
	var all []ModelInfo
	scanners := []func(*afero.Afero, string) []ModelInfo{
		ScanOllama,
		ScanHuggingFace,
		ScanLMStudio,
		func(fs *afero.Afero, h string) []ModelInfo { return ScanGPT4All(fs, h, osFamily) },
		ScanPyTorchHub,
		ScanKeras,
		ScanTFHub,
		ScanJan,
	}
	for _, scan := range scanners {
		all = append(all, scan(afs, home)...)
	}
	return all
}

// --- Ollama ---

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

func ScanOllama(afs *afero.Afero, home string) []ModelInfo {
	modelsDir := filepath.Join(home, ".ollama", "models")
	manifestsDir := filepath.Join(modelsDir, "manifests")

	// Ollama manifests follow a 4-level structure: registry/namespace/model/tag
	// (e.g. registry.ollama.ai/library/llama3/latest).
	// Walk each level explicitly to avoid unbounded traversal.
	registries, err := afs.ReadDir(manifestsDir)
	if err != nil {
		return nil
	}

	var results []ModelInfo
	for _, registry := range registries {
		if !registry.IsDir() {
			continue
		}
		registryDir := filepath.Join(manifestsDir, registry.Name())
		namespaces, err := afs.ReadDir(registryDir)
		if err != nil {
			continue
		}
		for _, ns := range namespaces {
			if !ns.IsDir() {
				continue
			}
			nsDir := filepath.Join(registryDir, ns.Name())
			models, err := afs.ReadDir(nsDir)
			if err != nil {
				continue
			}
			for _, model := range models {
				if !model.IsDir() {
					continue
				}
				modelBase := model.Name()
				modelDir := filepath.Join(nsDir, modelBase)
				tags, err := afs.ReadDir(modelDir)
				if err != nil {
					continue
				}
				for _, tag := range tags {
					if tag.IsDir() {
						continue
					}
					tagPath := filepath.Join(modelDir, tag.Name())
					data, err := afs.ReadFile(tagPath)
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

					extracted := readOllamaConfig(afs, modelsDir, manifest.Config.Digest)

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

func formatParamCount(count int64) string {
	switch {
	case count >= 1_000_000_000:
		b := float64(count) / 1e9
		if b == float64(int64(b)) {
			return fmt.Sprintf("%dB", int64(b))
		}
		return fmt.Sprintf("%.1fB", b)
	case count >= 1_000_000:
		m := float64(count) / 1e6
		if m == float64(int64(m)) {
			return fmt.Sprintf("%dM", int64(m))
		}
		return fmt.Sprintf("%.1fM", m)
	default:
		return fmt.Sprintf("%d", count)
	}
}

// --- Hugging Face Hub ---

type hfConfig struct {
	ModelType          string         `json:"model_type"`
	Architectures      []string       `json:"architectures"`
	QuantizationConfig map[string]any `json:"quantization_config"`
}

type hfReadmeMeta struct {
	License     string   `yaml:"license"`
	Tags        []string `yaml:"tags"`
	PipelineTag string   `yaml:"pipeline_tag"`
}

func ScanHuggingFace(afs *afero.Afero, home string) []ModelInfo {
	hubDir := filepath.Join(home, ".cache", "huggingface", "hub")
	entries, err := afs.ReadDir(hubDir)
	if err != nil {
		return nil
	}

	var results []ModelInfo
	for _, entry := range entries {
		if !entry.IsDir() || !strings.HasPrefix(entry.Name(), "models--") {
			continue
		}

		// Parse name from directory: models--<org>--<repo>
		nameParts := strings.SplitN(entry.Name(), "--", 3)
		if len(nameParts) < 3 {
			continue
		}
		modelName := nameParts[1] + "/" + nameParts[2]

		modelDir := filepath.Join(hubDir, entry.Name())
		blobsDir := filepath.Join(modelDir, "blobs")

		totalSize, modTime := dirSizeAndLatestMtime(afs, blobsDir)
		meta := extractHuggingFaceMetadata(afs, modelDir)

		paramSize := ""
		if m := reParamSize.FindStringSubmatch(modelName); len(m) > 1 {
			paramSize = m[1] + "B"
		}

		results = append(results, ModelInfo{
			Name:          modelName,
			Source:        "huggingface",
			Vendor:        nameParts[1],
			Family:        meta.Family,
			Path:          modelDir,
			Size:          totalSize,
			ModifiedAt:    modTime,
			Format:        meta.Format,
			Version:       meta.Version,
			Quantization:  meta.Quantization,
			ParameterSize: paramSize,
			Architecture:  meta.Architecture,
			License:       meta.License,
			Tags:          meta.Tags,
			Description:   meta.Description,
		})
	}
	return results
}

type hfExtracted struct {
	Format       string
	Family       string
	Version      string
	Architecture string
	Quantization string
	License      string
	Tags         []string
	Description  string
}

func extractHuggingFaceMetadata(afs *afero.Afero, modelDir string) hfExtracted {
	var result hfExtracted
	snapshotsDir := filepath.Join(modelDir, "snapshots")
	snapshots, err := afs.ReadDir(snapshotsDir)
	if err != nil || len(snapshots) == 0 {
		result.Format = "unknown"
		return result
	}

	latest := snapshots[0]
	for _, s := range snapshots[1:] {
		if s.ModTime().After(latest.ModTime()) {
			latest = s
		}
	}
	latestSnapshot := filepath.Join(snapshotsDir, latest.Name())

	// Version = first 12 chars of snapshot dir name (revision hash)
	rev := latest.Name()
	if len(rev) > 12 {
		rev = rev[:12]
	}
	result.Version = rev

	// Read config.json
	configPath := filepath.Join(latestSnapshot, "config.json")
	if data, readErr := afs.ReadFile(configPath); readErr == nil {
		var cfg hfConfig
		if json.Unmarshal(data, &cfg) == nil {
			result.Family = cfg.ModelType
			if len(cfg.Architectures) > 0 {
				result.Architecture = cfg.Architectures[0]
			}
			if cfg.QuantizationConfig != nil {
				if qt, ok := cfg.QuantizationConfig["quant_method"].(string); ok && qt != "" {
					result.Quantization = qt
				}
			}
		}
	}

	// README frontmatter
	readme := parseHFReadmeFrontmatter(afs, latestSnapshot)
	result.License = readme.License
	result.Tags = readme.Tags
	if readme.PipelineTag != "" && len(result.Tags) == 0 {
		result.Tags = []string{readme.PipelineTag}
	}

	// Format detection
	if f := detectFormatInDir(afs, latestSnapshot); f != "" {
		result.Format = f
	} else {
		subdirs, _ := afs.ReadDir(latestSnapshot)
		for _, d := range subdirs {
			fullPath := filepath.Join(latestSnapshot, d.Name())
			info, err := afs.Stat(fullPath)
			if err != nil || !info.IsDir() {
				continue
			}
			if f := detectFormatInDir(afs, fullPath); f != "" {
				result.Format = f
				break
			}
		}
		if result.Format == "" {
			result.Format = "unknown"
		}
	}

	// Quantization fallback from filenames
	if result.Quantization == "" {
		files, _ := afs.ReadDir(latestSnapshot)
		for _, f := range files {
			if m := reQuantization.FindString(f.Name()); m != "" {
				result.Quantization = strings.ToUpper(m)
				break
			}
		}
	}

	return result
}

func parseHFReadmeFrontmatter(afs *afero.Afero, snapshotDir string) hfReadmeMeta {
	var meta hfReadmeMeta
	readmePath := filepath.Join(snapshotDir, "README.md")
	data, err := afs.ReadFile(readmePath)
	if err != nil {
		return meta
	}
	content := string(data)
	if !strings.HasPrefix(content, "---") {
		return meta
	}
	end := strings.Index(content[3:], "---")
	if end < 0 {
		return meta
	}
	frontmatter := content[3 : 3+end]
	_ = yaml.Unmarshal([]byte(frontmatter), &meta)
	return meta
}

func detectFormatInDir(afs *afero.Afero, dir string) string {
	files, err := afs.ReadDir(dir)
	if err != nil {
		return ""
	}
	for _, f := range files {
		switch strings.ToLower(filepath.Ext(f.Name())) {
		case ".safetensors":
			return "safetensors"
		case ".gguf":
			return "gguf"
		case ".onnx":
			return "onnx"
		case ".npz":
			return "mlx"
		}
	}
	for _, f := range files {
		if strings.ToLower(filepath.Ext(f.Name())) == ".bin" {
			return "pytorch"
		}
	}
	return ""
}

// --- LM Studio ---

func ScanLMStudio(afs *afero.Afero, home string) []ModelInfo {
	dirs := []string{
		filepath.Join(home, ".lmstudio", "models"),
		filepath.Join(home, ".cache", "lm-studio", "models"),
	}

	seen := map[string]bool{}
	var results []ModelInfo
	for _, modelsDir := range dirs {
		entries, err := afs.ReadDir(modelsDir)
		if err != nil {
			continue
		}
		for _, publisher := range entries {
			if !publisher.IsDir() {
				continue
			}
			publisherDir := filepath.Join(modelsDir, publisher.Name())
			repos, err := afs.ReadDir(publisherDir)
			if err != nil {
				continue
			}
			for _, repo := range repos {
				if !repo.IsDir() {
					continue
				}
				repoDir := filepath.Join(publisherDir, repo.Name())
				modelName := publisher.Name() + "/" + repo.Name()
				if seen[modelName] {
					continue
				}
				seen[modelName] = true

				ggufFiles := findGGUFFiles(afs, repoDir)
				for _, m := range ggufFiles {
					filename := filepath.Base(m.path)
					quant := ""
					if match := reQuantization.FindString(filename); match != "" {
						quant = strings.ToUpper(match)
					}
					paramSize := ""
					if pm := reParamSize.FindStringSubmatch(modelName); len(pm) > 1 {
						paramSize = pm[1] + "B"
					}

					results = append(results, ModelInfo{
						Name:          modelName + "/" + filename,
						Source:        "lmstudio",
						Vendor:        publisher.Name(),
						Path:          m.path,
						Size:          m.size,
						ModifiedAt:    m.modTime,
						Format:        "gguf",
						Quantization:  quant,
						ParameterSize: paramSize,
					})
				}
			}
		}
	}
	return results
}

type fileEntry struct {
	path    string
	size    int64
	modTime time.Time
}

func findGGUFFiles(afs *afero.Afero, dir string) []fileEntry {
	entries, err := afs.ReadDir(dir)
	if err != nil {
		return nil
	}
	var results []fileEntry
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if strings.HasSuffix(strings.ToLower(e.Name()), ".gguf") {
			results = append(results, fileEntry{
				path:    filepath.Join(dir, e.Name()),
				size:    e.Size(),
				modTime: e.ModTime(),
			})
		}
	}
	return results
}

// --- GPT4All ---

func ScanGPT4All(afs *afero.Afero, home string, osFamily string) []ModelInfo {
	dirs := gpt4allDirs(home, osFamily)
	seen := map[string]bool{}
	var results []ModelInfo

	for _, dir := range dirs {
		entries, err := afs.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			lower := strings.ToLower(e.Name())
			if !strings.HasSuffix(lower, ".gguf") && !strings.HasSuffix(lower, ".bin") {
				continue
			}
			if seen[e.Name()] {
				continue
			}
			seen[e.Name()] = true

			format := "gguf"
			if strings.HasSuffix(lower, ".bin") {
				format = "ggml"
			}

			quant := ""
			if match := reQuantization.FindString(e.Name()); match != "" {
				quant = strings.ToUpper(match)
			}
			paramSize := ""
			if pm := reParamSize.FindStringSubmatch(e.Name()); len(pm) > 1 {
				paramSize = pm[1] + "B"
			}

			results = append(results, ModelInfo{
				Name:          e.Name(),
				Source:        "gpt4all",
				Path:          filepath.Join(dir, e.Name()),
				Size:          e.Size(),
				ModifiedAt:    e.ModTime(),
				Format:        format,
				Quantization:  quant,
				ParameterSize: paramSize,
			})
		}
	}
	return results
}

func gpt4allDirs(home string, osFamily string) []string {
	dirs := []string{
		filepath.Join(home, ".cache", "gpt4all"),
	}
	switch osFamily {
	case "darwin":
		dirs = append(dirs, filepath.Join(home, "Library", "Application Support", "nomic.ai", "GPT4All"))
	case "linux":
		dirs = append(dirs, filepath.Join(home, ".local", "share", "nomic.ai", "GPT4All"))
	case "windows":
		dirs = append(dirs, filepath.Join(home, "AppData", "Local", "nomic.ai", "GPT4All"))
	}
	return dirs
}

// --- PyTorch Hub ---

func ScanPyTorchHub(afs *afero.Afero, home string) []ModelInfo {
	checkpointsDir := filepath.Join(home, ".cache", "torch", "hub", "checkpoints")
	entries, err := afs.ReadDir(checkpointsDir)
	if err != nil {
		return nil
	}

	var results []ModelInfo
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		lower := strings.ToLower(e.Name())
		if !strings.HasSuffix(lower, ".pth") && !strings.HasSuffix(lower, ".pt") {
			continue
		}

		name := strings.TrimSuffix(strings.TrimSuffix(e.Name(), ".pth"), ".pt")
		// Strip trailing hash: resnet50-0676ba61 -> resnet50
		if idx := strings.LastIndex(name, "-"); idx > 0 {
			candidate := name[idx+1:]
			if len(candidate) >= 6 && isHex(candidate) {
				name = name[:idx]
			}
		}

		results = append(results, ModelInfo{
			Name:       name,
			Source:     "pytorch",
			Path:       filepath.Join(checkpointsDir, e.Name()),
			Size:       e.Size(),
			ModifiedAt: e.ModTime(),
			Format:     "pytorch",
		})
	}
	return results
}

func isHex(s string) bool {
	for _, c := range s {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') && (c < 'A' || c > 'F') {
			return false
		}
	}
	return true
}

// --- Keras ---

func ScanKeras(afs *afero.Afero, home string) []ModelInfo {
	modelsDir := filepath.Join(home, ".keras", "models")
	entries, err := afs.ReadDir(modelsDir)
	if err != nil {
		return nil
	}

	var results []ModelInfo
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		lower := strings.ToLower(e.Name())
		if !strings.HasSuffix(lower, ".h5") && !strings.HasSuffix(lower, ".keras") {
			continue
		}

		format := "h5"
		if strings.HasSuffix(lower, ".keras") {
			format = "keras"
		}

		name := strings.TrimSuffix(strings.TrimSuffix(e.Name(), ".h5"), ".keras")

		results = append(results, ModelInfo{
			Name:       name,
			Source:     "keras",
			Path:       filepath.Join(modelsDir, e.Name()),
			Size:       e.Size(),
			ModifiedAt: e.ModTime(),
			Format:     format,
		})
	}
	return results
}

// --- TensorFlow Hub ---

func ScanTFHub(afs *afero.Afero, home string) []ModelInfo {
	modulesDir := filepath.Join(home, ".cache", "tfhub_modules")
	entries, err := afs.ReadDir(modulesDir)
	if err != nil {
		return nil
	}

	var results []ModelInfo
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		moduleDir := filepath.Join(modulesDir, e.Name())

		// TF Hub modules contain saved_model.pb
		savedModel := filepath.Join(moduleDir, "saved_model.pb")
		if exists, _ := afs.Exists(savedModel); !exists {
			continue
		}

		totalSize, modTime := dirSizeRecursive(afs, moduleDir)

		results = append(results, ModelInfo{
			Name:       e.Name(),
			Source:     "tfhub",
			Path:       moduleDir,
			Size:       totalSize,
			ModifiedAt: modTime,
			Format:     "savedmodel",
		})
	}
	return results
}

func dirSizeRecursive(afs *afero.Afero, dir string) (int64, time.Time) {
	var totalSize int64
	var latest time.Time
	_ = afero.Walk(afs, dir, func(_ string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		totalSize += info.Size()
		if info.ModTime().After(latest) {
			latest = info.ModTime()
		}
		return nil
	})
	return totalSize, latest
}

// --- Jan ---

type janModelMeta struct {
	Name        string            `json:"name"`
	ID          string            `json:"id"`
	Format      string            `json:"format"`
	Version     string            `json:"version"`
	Description string            `json:"description"`
	License     string            `json:"license"`
	Metadata    janModelMetadata  `json:"metadata"`
	Publisher   janModelPublisher `json:"publisher"`
}

type janModelMetadata struct {
	Author string   `json:"author"`
	Name   string   `json:"name"`
	Tags   []string `json:"tags"`
}

type janModelPublisher struct {
	Author string `json:"author"`
	Name   string `json:"name"`
}

func ScanJan(afs *afero.Afero, home string) []ModelInfo {
	modelsDir := filepath.Join(home, "jan", "models")
	entries, err := afs.ReadDir(modelsDir)
	if err != nil {
		return nil
	}

	var results []ModelInfo
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		modelDir := filepath.Join(modelsDir, e.Name())

		name := e.Name()
		format := "unknown"
		vendor := ""
		version := ""
		description := ""
		license := ""
		var tags []string

		metaPath := filepath.Join(modelDir, "model.json")
		if data, err := afs.ReadFile(metaPath); err == nil {
			var meta janModelMeta
			if json.Unmarshal(data, &meta) == nil {
				if meta.Name != "" {
					name = meta.Name
				} else if meta.ID != "" {
					name = meta.ID
				}
				if meta.Format != "" {
					format = strings.ToLower(meta.Format)
				}
				if meta.Publisher.Name != "" {
					vendor = meta.Publisher.Name
				} else if meta.Publisher.Author != "" {
					vendor = meta.Publisher.Author
				} else if meta.Metadata.Author != "" {
					vendor = meta.Metadata.Author
				}
				version = meta.Version
				description = meta.Description
				license = meta.License
				tags = meta.Metadata.Tags
			}
		}

		if format == "unknown" {
			format = detectDirModelFormat(afs, modelDir)
		}

		// Quantization from GGUF filenames in directory
		quant := ""
		dirEntries, _ := afs.ReadDir(modelDir)
		for _, f := range dirEntries {
			if match := reQuantization.FindString(f.Name()); match != "" {
				quant = strings.ToUpper(match)
				break
			}
		}

		paramSize := ""
		if pm := reParamSize.FindStringSubmatch(name); len(pm) > 1 {
			paramSize = pm[1] + "B"
		}

		totalSize, modTime := dirSizeRecursive(afs, modelDir)

		results = append(results, ModelInfo{
			Name:          name,
			Source:        "jan",
			Vendor:        vendor,
			Path:          modelDir,
			Size:          totalSize,
			ModifiedAt:    modTime,
			Format:        format,
			Version:       version,
			Quantization:  quant,
			ParameterSize: paramSize,
			License:       license,
			Tags:          tags,
			Description:   description,
		})
	}
	return results
}

func detectDirModelFormat(afs *afero.Afero, dir string) string {
	entries, err := afs.ReadDir(dir)
	if err != nil {
		return "unknown"
	}
	for _, f := range entries {
		switch strings.ToLower(filepath.Ext(f.Name())) {
		case ".gguf":
			return "gguf"
		case ".safetensors":
			return "safetensors"
		case ".onnx":
			return "onnx"
		}
	}
	return "unknown"
}

// --- Helpers ---

func dirSizeAndLatestMtime(afs *afero.Afero, dir string) (int64, time.Time) {
	var totalSize int64
	var latest time.Time
	entries, err := afs.ReadDir(dir)
	if err != nil {
		return 0, latest
	}
	for _, e := range entries {
		if e.IsDir() || strings.HasSuffix(e.Name(), ".lock") {
			continue
		}
		totalSize += e.Size()
		if e.ModTime().After(latest) {
			latest = e.ModTime()
		}
	}
	return totalSize, latest
}
