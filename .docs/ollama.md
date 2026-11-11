# Ollama

Checked **2026-11-11**.

Facet supports a locally administered Ollama provider. It defaults to `http://localhost:11434`, uses `/api/generate` for prompts and `/api/tags` for model discovery, and defaults to model `llama3.2` when none is configured. Prompt requests disable streaming and set `num_ctx` to 8192. Both API endpoints remain documented upstream.

The latest upstream release is **v0.40.16**, published 2026-11-09. No Ollama server or model version is pinned by this repository, and no local server was inspected for this inventory. Owners should compare their installations with upstream. Compatibility and model lifecycle depend on the owner's installation. Preserve the base-URL fallback to `http://ollama:11434` used by Docker-oriented configurations when changing discovery code.

The maintainer's security-advisory page showed no published advisories when checked. That is a point-in-time repository check, not a general vulnerability clearance.

Sources: [`backend/services/ai.go`](../backend/services/ai.go), [`backend/hooks/ai.go`](../backend/hooks/ai.go), [`backend/hooks/ai_ollama_test.go`](../backend/hooks/ai_ollama_test.go), [official Ollama API documentation](https://docs.ollama.com/api), [v0.40.16 release](https://github.com/ollama/ollama/releases/tag/v0.40.16), [maintainer advisories](https://github.com/ollama/ollama/security/advisories), all checked 2026-11-11.
