---
id: TASK-650
planId: PLAN-078
title: 'Optional hybrid vector search & RRF ranking'
role: search
status: done
createdAt: 2026-11-18
---

# TASK-650 — Optional hybrid vector search & RRF ranking

Source: PLAN-078.

Problem: Pure lexical search misses synonyms and semantic concepts when terminology diverges.

Goal:

1. Optional embedding generation using available local or API embedding models (Ollama/LM Studio/OpenAI/Gemini).
2. Store vector embeddings in SQLite for knowledge items.
3. Combine lexical BM25 and vector cosine similarity ranks using Reciprocal Rank Fusion (RRF).
4. Gracefully fall back to pure BM25 when no embedding provider is available.
