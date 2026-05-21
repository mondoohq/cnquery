// Copyright Mondoo, Inc. 2024, 2026
// SPDX-License-Identifier: BUSL-1.1

package aimodel

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestAfs() (*afero.Afero, afero.Fs) {
	fs := afero.NewMemMapFs()
	return &afero.Afero{Fs: fs}, fs
}

func writeJSON(t *testing.T, fs afero.Fs, path string, v any) {
	t.Helper()
	data, err := json.Marshal(v)
	require.NoError(t, err)
	require.NoError(t, afero.WriteFile(fs, path, data, 0644))
}

func TestScanOllama(t *testing.T) {
	afs, fs := newTestAfs()
	home := "/home/testuser"

	manifest := ollamaManifest{
		Config: ollamaDescriptor{Digest: "sha256:abc123"},
		Layers: []ollamaLayer{
			{MediaType: "application/vnd.ollama.image.model", Digest: "sha256:model1", Size: 1000},
			{MediaType: "application/vnd.ollama.image.template", Digest: "sha256:tmpl1", Size: 200},
		},
	}
	manifestPath := filepath.Join(home, ".ollama/models/manifests/registry.ollama.ai/library/llama3/latest")
	writeJSON(t, fs, manifestPath, manifest)

	configBlob := ollamaConfig{ModelFamily: "llama"}
	configPath := filepath.Join(home, ".ollama/models/blobs/sha256-abc123")
	writeJSON(t, fs, configPath, configBlob)

	results := ScanOllama(afs, home)
	require.Len(t, results, 1)

	m := results[0]
	assert.Equal(t, "llama3:latest", m.Name)
	assert.Equal(t, "ollama", m.Source)
	assert.Equal(t, "Meta", m.Vendor)
	assert.Equal(t, "llama", m.Family)
	assert.Equal(t, int64(1200), m.Size)
	assert.Equal(t, "gguf", m.Format)
}

func TestScanOllama_MultipleModels(t *testing.T) {
	afs, fs := newTestAfs()
	home := "/home/testuser"

	for _, tc := range []struct {
		model, tag, family, vendor string
	}{
		{"gemma", "7b", "gemma", "Google"},
		{"deepseek-r1", "latest", "deepseek2", "DeepSeek"},
		{"qwen", "4b", "qwen2", "Alibaba"},
	} {
		manifest := ollamaManifest{
			Config: ollamaDescriptor{Digest: "sha256:" + tc.model},
			Layers: []ollamaLayer{{Size: 500}},
		}
		manifestPath := filepath.Join(home, ".ollama/models/manifests/registry.ollama.ai/library", tc.model, tc.tag)
		writeJSON(t, fs, manifestPath, manifest)

		configPath := filepath.Join(home, ".ollama/models/blobs/sha256-"+tc.model)
		writeJSON(t, fs, configPath, ollamaConfig{ModelFamily: tc.family})
	}

	results := ScanOllama(afs, home)
	require.Len(t, results, 3)

	vendors := map[string]string{}
	families := map[string]string{}
	for _, m := range results {
		vendors[m.Name] = m.Vendor
		families[m.Name] = m.Family
	}

	assert.Equal(t, "Google", vendors["gemma:7b"])
	assert.Equal(t, "DeepSeek", vendors["deepseek-r1:latest"])
	assert.Equal(t, "Alibaba", vendors["qwen:4b"])
	assert.Equal(t, "gemma", families["gemma:7b"])
	assert.Equal(t, "deepseek2", families["deepseek-r1:latest"])
	assert.Equal(t, "qwen2", families["qwen:4b"])
}

func TestScanOllama_MalformedManifest(t *testing.T) {
	afs, fs := newTestAfs()
	home := "/home/testuser"

	manifestPath := filepath.Join(home, ".ollama/models/manifests/registry.ollama.ai/library/badmodel/latest")
	require.NoError(t, afero.WriteFile(fs, manifestPath, []byte("not json"), 0644))

	results := ScanOllama(afs, home)
	assert.Empty(t, results)
}

func TestScanOllama_EmptyLayers(t *testing.T) {
	afs, fs := newTestAfs()
	home := "/home/testuser"

	manifest := ollamaManifest{
		Config: ollamaDescriptor{Digest: "sha256:abc"},
		Layers: []ollamaLayer{},
	}
	manifestPath := filepath.Join(home, ".ollama/models/manifests/registry.ollama.ai/library/empty/latest")
	writeJSON(t, fs, manifestPath, manifest)

	results := ScanOllama(afs, home)
	assert.Empty(t, results)
}

func TestScanOllama_MissingConfigBlob(t *testing.T) {
	afs, fs := newTestAfs()
	home := "/home/testuser"

	manifest := ollamaManifest{
		Config: ollamaDescriptor{Digest: "sha256:missing"},
		Layers: []ollamaLayer{{Size: 100}},
	}
	manifestPath := filepath.Join(home, ".ollama/models/manifests/registry.ollama.ai/library/phi/latest")
	writeJSON(t, fs, manifestPath, manifest)

	results := ScanOllama(afs, home)
	require.Len(t, results, 1)
	assert.Equal(t, "Microsoft", results[0].Vendor)
	assert.Equal(t, "", results[0].Family)
}

func TestScanOllama_NoManifestsDir(t *testing.T) {
	afs, _ := newTestAfs()
	results := ScanOllama(afs, "/nonexistent")
	assert.Nil(t, results)
}

func TestOllamaVendor(t *testing.T) {
	tests := []struct {
		modelBase string
		want      string
	}{
		{"llama3", "Meta"},
		{"codellama", "Meta"},
		{"gemma", "Google"},
		{"deepseek-r1", "DeepSeek"},
		{"qwen", "Alibaba"},
		{"phi", "Microsoft"},
		{"mistral", "Mistral AI"},
		{"mixtral", "Mistral AI"},
		{"glm", "Z.AI"},
		{"grok", "xAI"},
		{"jamba", "AI21 Labs"},
		{"baichuan", "Baichuan"},
		{"rwkv", "RWKV Foundation"},
		{"unknown-model", ""},
		{"", ""},
	}
	for _, tt := range tests {
		t.Run(tt.modelBase, func(t *testing.T) {
			assert.Equal(t, tt.want, ollamaVendor(tt.modelBase))
		})
	}
}

func TestScanHuggingFace(t *testing.T) {
	afs, fs := newTestAfs()
	home := "/home/testuser"

	modelDir := filepath.Join(home, ".cache/huggingface/hub/models--meta-llama--Llama-2-7b-hf")
	snapshotDir := filepath.Join(modelDir, "snapshots/abc123def")

	require.NoError(t, afero.WriteFile(fs, filepath.Join(snapshotDir, "model.safetensors"), make([]byte, 5000), 0644))

	cfg := hfConfig{ModelType: "llama"}
	writeJSON(t, fs, filepath.Join(snapshotDir, "config.json"), cfg)

	require.NoError(t, afero.WriteFile(fs, filepath.Join(modelDir, "blobs/sha256abc"), make([]byte, 4000), 0644))
	require.NoError(t, afero.WriteFile(fs, filepath.Join(modelDir, "blobs/sha256def"), make([]byte, 1000), 0644))

	results := ScanHuggingFace(afs, home)
	require.Len(t, results, 1)

	m := results[0]
	assert.Equal(t, "meta-llama/Llama-2-7b-hf", m.Name)
	assert.Equal(t, "huggingface", m.Source)
	assert.Equal(t, "meta-llama", m.Vendor)
	assert.Equal(t, "llama", m.Family)
	assert.Equal(t, int64(5000), m.Size)
	assert.Equal(t, "safetensors", m.Format)
}

func TestScanHuggingFace_GGUFFormat(t *testing.T) {
	afs, fs := newTestAfs()
	home := "/home/testuser"

	modelDir := filepath.Join(home, ".cache/huggingface/hub/models--TheBloke--Llama-2-7B-GGUF")
	snapshotDir := filepath.Join(modelDir, "snapshots/rev1")
	require.NoError(t, afero.WriteFile(fs, filepath.Join(snapshotDir, "llama-2-7b.Q4_K_M.gguf"), make([]byte, 3000), 0644))
	require.NoError(t, afero.WriteFile(fs, filepath.Join(modelDir, "blobs/b1"), make([]byte, 3000), 0644))

	results := ScanHuggingFace(afs, home)
	require.Len(t, results, 1)
	assert.Equal(t, "gguf", results[0].Format)
}

func TestScanHuggingFace_ONNXInSubdir(t *testing.T) {
	afs, fs := newTestAfs()
	home := "/home/testuser"

	modelDir := filepath.Join(home, ".cache/huggingface/hub/models--sentence-transformers--all-MiniLM-L6-v2")
	snapshotDir := filepath.Join(modelDir, "snapshots/rev1")
	require.NoError(t, afero.WriteFile(fs, filepath.Join(snapshotDir, "onnx/model.onnx"), make([]byte, 2000), 0644))
	require.NoError(t, afero.WriteFile(fs, filepath.Join(modelDir, "blobs/b1"), make([]byte, 2000), 0644))

	results := ScanHuggingFace(afs, home)
	require.Len(t, results, 1)
	assert.Equal(t, "onnx", results[0].Format)
}

func TestScanHuggingFace_EmptySnapshotsDir(t *testing.T) {
	afs, fs := newTestAfs()
	home := "/home/testuser"

	modelDir := filepath.Join(home, ".cache/huggingface/hub/models--org--repo")
	require.NoError(t, fs.MkdirAll(filepath.Join(modelDir, "snapshots"), 0755))
	require.NoError(t, afero.WriteFile(fs, filepath.Join(modelDir, "blobs/b1"), make([]byte, 100), 0644))

	results := ScanHuggingFace(afs, home)
	require.Len(t, results, 1)
	assert.Equal(t, "unknown", results[0].Format)
	assert.Equal(t, "", results[0].Family)
}

func TestScanHuggingFace_LockFilesExcluded(t *testing.T) {
	afs, fs := newTestAfs()
	home := "/home/testuser"

	modelDir := filepath.Join(home, ".cache/huggingface/hub/models--org--model")
	snapshotDir := filepath.Join(modelDir, "snapshots/rev1")
	require.NoError(t, afero.WriteFile(fs, filepath.Join(snapshotDir, "model.safetensors"), make([]byte, 100), 0644))

	require.NoError(t, afero.WriteFile(fs, filepath.Join(modelDir, "blobs/sha1"), make([]byte, 1000), 0644))
	require.NoError(t, afero.WriteFile(fs, filepath.Join(modelDir, "blobs/sha1.lock"), make([]byte, 500), 0644))

	results := ScanHuggingFace(afs, home)
	require.Len(t, results, 1)
	assert.Equal(t, int64(1000), results[0].Size)
}

func TestScanHuggingFace_NoHubDir(t *testing.T) {
	afs, _ := newTestAfs()
	results := ScanHuggingFace(afs, "/nonexistent")
	assert.Nil(t, results)
}

func TestScanHuggingFace_SkipsNonModelDirs(t *testing.T) {
	afs, fs := newTestAfs()
	home := "/home/testuser"

	hubDir := filepath.Join(home, ".cache/huggingface/hub")
	require.NoError(t, fs.MkdirAll(filepath.Join(hubDir, "datasets--org--data"), 0755))
	require.NoError(t, afero.WriteFile(fs, filepath.Join(hubDir, ".locks"), []byte("lock"), 0644))

	results := ScanHuggingFace(afs, home)
	assert.Empty(t, results)
}

func TestScanAll(t *testing.T) {
	afs, fs := newTestAfs()
	home := "/home/testuser"

	manifest := ollamaManifest{
		Config: ollamaDescriptor{Digest: "sha256:cfg1"},
		Layers: []ollamaLayer{{Size: 100}},
	}
	writeJSON(t, fs, filepath.Join(home, ".ollama/models/manifests/registry.ollama.ai/library/llama3/latest"), manifest)
	writeJSON(t, fs, filepath.Join(home, ".ollama/models/blobs/sha256-cfg1"), ollamaConfig{ModelFamily: "llama"})

	modelDir := filepath.Join(home, ".cache/huggingface/hub/models--google--bert-base")
	snapshotDir := filepath.Join(modelDir, "snapshots/rev1")
	require.NoError(t, afero.WriteFile(fs, filepath.Join(snapshotDir, "model.safetensors"), make([]byte, 200), 0644))
	require.NoError(t, afero.WriteFile(fs, filepath.Join(modelDir, "blobs/b1"), make([]byte, 200), 0644))

	results := ScanAll(afs, home, "linux")
	assert.GreaterOrEqual(t, len(results), 2)

	sources := map[string]bool{}
	for _, m := range results {
		sources[m.Source] = true
	}
	assert.True(t, sources["ollama"])
	assert.True(t, sources["huggingface"])
}

func TestDetectFormatInDir(t *testing.T) {
	tests := []struct {
		name     string
		files    []string
		expected string
	}{
		{"safetensors", []string{"model.safetensors"}, "safetensors"},
		{"gguf", []string{"model.gguf"}, "gguf"},
		{"onnx", []string{"model.onnx"}, "onnx"},
		{"mlx", []string{"weights.npz"}, "mlx"},
		{"pytorch_bin", []string{"pytorch_model.bin"}, "pytorch"},
		{"safetensors_over_bin", []string{"model.safetensors", "model.bin"}, "safetensors"},
		{"empty", []string{}, ""},
		{"unknown_ext", []string{"readme.txt", "config.json"}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			afs, fs := newTestAfs()
			dir := "/test"
			for _, f := range tt.files {
				require.NoError(t, afero.WriteFile(fs, filepath.Join(dir, f), []byte("data"), 0644))
			}
			assert.Equal(t, tt.expected, detectFormatInDir(afs, dir))
		})
	}
}
