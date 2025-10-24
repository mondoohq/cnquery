# VirusTotal provider (PoC)

This provider integrates [VirusTotal](https://www.virustotal.com/) data into cnquery. It currently supports
domain and IP lookups using the official [`vt-go`](https://github.com/VirusTotal/vt-go) client library.

## Usage

```bash
cnquery shell virustotal --api-key $VT_API_KEY
```

Once connected you can explore resources such as:

```mql
virustotal.domain("example.com") {
  reputation
  categories
  lastAnalysisStats
}

virustotal.ip("8.8.8.8") {
  reputation
  country
  lastAnalysisStats
}
```

> **Note:** This is a proof-of-concept. The resources and fields are subject to change as the provider evolves.

