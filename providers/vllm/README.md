# vLLM Provider

The vLLM provider probes the externally observable HTTP posture of a vLLM
inference server. It checks selected documentation, metadata, OpenAI-compatible,
metrics, utility, and operational endpoints from the URL supplied to the
connector.

## Usage

```bash
mql shell vllm http://localhost:8000
mql shell vllm https://vllm.example.com --api-key <token>
```

Use `--insecure` only for lab or test endpoints where TLS certificate
verification is intentionally disabled.

## Authentication

Authenticated comparison probes use an API key from `--api-key`, inventory
credentials, the `api-key` option, or `VLLM_API_KEY`. Explicit credentials in
the connection configuration take precedence over environment variables.

## Security Considerations

Only point this provider at vLLM endpoints you are authorized to assess. The
connector sends HTTP requests to the configured URL. If policies or automation
build that URL from untrusted asset metadata, the scan can become a server-side
request forgery path to internal addresses such as loopback, link-local
metadata services, or private network hosts.

Redirects are not followed. This keeps endpoint posture tied to the probed path
and avoids forwarding bearer tokens to redirect targets.
