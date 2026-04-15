---
id: TASK-507
plan: PLAN-065
status: done
wave: 3
priority: 2
---

# TASK-507: Fix GitHub Pages download architecture recommendation

## Description

Apple Silicon detection in the downloads page can incorrectly highlight Intel builds. This task implements robust architecture detection logic with safer fallback behavior for uncertain macOS architecture reporting.

## Checklist

- [ ] Improve detection logic in docs Vue downloads page
- [ ] Update legacy/static docs downloads page detection script to match
- [ ] Avoid false Intel recommendation on uncertain macOS detection

## Files

- `docs/src/views/DownloadsView.vue`
- `docs/downloads.md`

## Acceptance Criteria

- Apple Silicon devices are recommended ARM64 builds in common browser cases
- Unknown macOS architecture does not incorrectly force Intel recommendation
- Docs build remains valid
