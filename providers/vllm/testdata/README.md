# vLLM Provider Testing

Test the mql vLLM provider against a live vLLM inference server.

## Setup (macOS, Apple Silicon)

vLLM runs natively on macOS with CPU-only mode. Use Python 3.12 or 3.13 (3.14+ is not yet supported).

### 1. Install vLLM

```bash
python3.12 -m venv /tmp/vllm-env
/tmp/vllm-env/bin/pip install vllm
```

### 2. Start the server

```bash
VLLM_CPU_KVCACHE_SPACE=1 /tmp/vllm-env/bin/vllm serve sshleifer/tiny-gpt2 \
  --host 0.0.0.0 --port 8000
```

The `sshleifer/tiny-gpt2` model is ~2 MB and loads in seconds. Wait for `Application startup complete.` in the output.

### 3. Build and install the provider

```bash
make providers/build/vllm && make providers/install/vllm
```

### 4. Run the test suite

```bash
VLLM_IP=localhost ./providers/vllm/testdata/verify.sh
```

## Ad-hoc queries

```bash
mql run vllm http://localhost:8000 -c "vllm.server { baseUrl reachable version }"
mql run vllm http://localhost:8000 -c "vllm.models { id root maxModelLen ownedBy }"
mql run vllm http://localhost:8000 -c "vllm.endpoints { method path present anonymousAccessible }"
```

## Environment Variables

| Variable | Default | Description |
|---|---|---|
| `VLLM_IP` | `localhost` | IP or hostname of the vLLM server |
| `VLLM_PORT` | `8000` | Port the vLLM server listens on |

## Testing with a Remote Server

Point the test suite at any reachable vLLM instance:

```bash
VLLM_IP=vllm.example.com VLLM_PORT=8000 ./providers/vllm/testdata/verify.sh
```

## Notes

- vLLM on macOS uses CPU-only mode (no GPU required)
- Python 3.12 or 3.13 is recommended; 3.14+ is not yet supported by vLLM
- The `VLLM_CPU_KVCACHE_SPACE=1` env var limits KV cache to 1 GB of RAM
- On Linux, vLLM requires a CUDA-capable GPU
