# Ollama

Checked **2026-11-10**.

Facet supports a locally administered Ollama provider. It defaults to `http://localhost:11434`, uses `/api/generate` for prompts and `/api/tags` for model discovery, and defaults to model `llama3.2` when none is configured. Prompt requests disable streaming and set `num_ctx` to 8192.

No Ollama server or model version is pinned by this repository, and no local server was inspected for this inventory. Compatibility and model lifecycle therefore depend on the owner's installation. Preserve the base-URL fallback to `http://ollama:11434` used by Docker-oriented configurations when changing discovery code.

Sources: [`backend/services/ai.go`](../backend/services/ai.go), [`backend/hooks/ai.go`](../backend/hooks/ai.go), [`backend/hooks/ai_ollama_test.go`](../backend/hooks/ai_ollama_test.go), [official Ollama API documentation](https://docs.ollama.com/api), all checked 2026-11-10.
