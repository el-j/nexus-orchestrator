---
id: TASK-647
planId: PLAN-078
title: 'Tokenized natural-language FTS5 BM25 search'
role: search
status: done
createdAt: 2026-11-18
---

# TASK-647 — Tokenized natural-language FTS5 BM25 search

Source: PLAN-078.

Problem: `SearchFTS` wraps the entire query in quotes (`"how do we handle rate limits"`), which requires an exact literal substring match and returns nothing for natural language questions.

Goal:

1. Tokenize query strings into valid FTS5 tokens, stripping operators/punctuation that break FTS5 syntax.
2. Build ranked FTS5 expressions combining multi-term prefix and OR matches.
3. Weight `topic` over `content` via `bm25(project_knowledge_fts, 5.0, 1.0)`.
4. Ensure 100% test coverage and doc comments.
