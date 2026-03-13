# TASK-268 — Extend Ollama probe to call /api/ps

**Plan**: PLAN-041  
**Status**: done

## What
In sys_scanner, after the port probe for Ollama succeeds, make an additional HTTP GET
to `http://127.0.0.1:11434/api/ps`. Parse the response to extract:
- `loadedModels []string` — model names currently loaded into VRAM
- `generating bool` — true if any model has `size_vram > 0` or `expires_at` is in the future

Populate the `ActiveModels` and `Generating` fields on the returned DiscoveredProvider.
