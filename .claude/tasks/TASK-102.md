---
id: TASK-102
title: Add license and version badges to README
role: devops
planId: PLAN-012
status: todo
dependencies: [TASK-098]
createdAt: 2026-03-10T20:00:00.000Z
---

## Context
README should show license badge (MIT) and optionally a GitHub Release version badge for professional appearance.

## Files to Read
- `README.md`
- `LICENSE` (created in TASK-098)

## Implementation Steps
1. Add badges at the top of README.md, below the `# nexusOrchestrator` heading:
   - MIT License badge: `[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)`
   - GitHub Release badge: `[![GitHub Release](https://img.shields.io/github/v/release/el-j/nexusOrchestrator)](https://github.com/el-j/nexusOrchestrator/releases/latest)`
   - Go Report Card (optional): `[![Go Report Card](https://goreportcard.com/badge/github.com/el-j/nexusOrchestrator)](https://goreportcard.com/report/github.com/el-j/nexusOrchestrator)`
2. Add a "License" section at the bottom of README with: `This project is licensed under the MIT License — see the [LICENSE](LICENSE) file for details.`

## Acceptance Criteria
- [ ] README has MIT license badge
- [ ] README has GitHub Release version badge
- [ ] README has License section at bottom
- [ ] Badge URLs use correct repo `el-j/nexusOrchestrator`

## Anti-patterns to Avoid
- Do NOT add too many badges — keep it clean (max 3-4)
- Do NOT hardcode a version number in the badge
