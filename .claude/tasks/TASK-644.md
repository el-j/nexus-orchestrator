---
id: TASK-644
planId: PLAN-077
title: 'Frontend lint and the nilerr design'
role: quality
status: done
createdAt: 2026-10-31
---

# TASK-644 — Frontend lint and the nilerr design

Source: [docs/audit-2026-10-31.md](../../docs/audit-2026-10-31.md).

Add an ESLint flat config for frontend and docs (vue + typescript-eslint, no-explicit-any as error).
Replace the six `nilerr` returns in activity_network/reader.go with a typed result so "no activity"
and "unreachable" differ.

Done: ESLint flat configs (`eslint.config.js`: typescript-eslint recommended, eslint-plugin-vue essential, `no-explicit-any` and unused variables as errors) for `frontend` and `docs`, `npm run lint` in both, and a lint step for each in `ci.yml`; the dozen findings (escaped quotes in specs, unused catch variables, one `this` alias) are fixed.

Decided against: a typed result for `activity_network`'s best-effort probes. The reader's port already returns an error and the activity service only reads what comes back; nothing shows reader health, so a result type that tells "no activity" from "unreachable" would have no consumer. The best-effort behaviour is documented on each probe. The linter has no `nilerr` check enabled.
