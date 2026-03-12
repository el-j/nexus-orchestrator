---
id: TASK-140
title: Add VS Code extension section to DownloadsView.vue
role: frontend
planId: PLAN-019
status: todo
dependencies: []
createdAt: 2026-03-11T18:00:00.000Z
---

## Context

`docs/src/views/DownloadsView.vue` has sections for Desktop App, CLI+Daemon, Quick
Install, Verify, System Requirements, and "What's Included". The VS Code extension is
not mentioned anywhere. Users arriving on the downloads page cannot discover or download
the `.vsix`.

## Files to Read

- `docs/src/views/DownloadsView.vue` (full file)

## Implementation Steps

1. **Add a "VS Code Extension" download section** between the "CLI + Daemon" section and
   the "Quick Install" section. The section should:
   - Use the same card style as the `cliCards` section (smaller border, `border-white/8`)
   - Display a single download card for the `.vsix` file
   - Download URL: `https://github.com/el-j/nexus-orchestrator/releases/latest/download/nexus-orchestrator-vscode.vsix`
   - Show the file extension `.vsix` on the button
   - Icon: `🔌` (puzzle piece / extension icon)
   - Include a short description: "Install in VS Code 1.85+ via Extensions sidebar or `code --install-extension`"
   
   Suggested markup (add after the `</section>` closing the CLI+Daemon section):
   ```html
   <!-- VS Code Extension Section -->
   <section class="mb-16">
     <div class="flex items-center gap-3 mb-6 reveal">
       <span class="text-2xl">🔌</span>
       <h2 class="text-2xl font-black">VS Code Extension</h2>
       <span class="px-2 py-0.5 text-xs rounded-full bg-cyan-600/20 border border-cyan-500/30 text-cyan-300">New</span>
     </div>
     <p class="text-slate-500 mb-8 reveal">
       Submit tasks, monitor the queue, and switch providers without leaving your editor.
       Requires VS Code 1.85+ and a running nexus daemon.
     </p>
     <div class="max-w-xs reveal">
       <div class="rounded-xl border border-white/8 bg-[#0d0d14] hover:border-cyan-500/30 p-6 text-center transition-all">
         <div class="text-3xl mb-3">🔌</div>
         <div class="font-bold text-white mb-1">VS Code</div>
         <div class="text-xs text-slate-500 mb-1">1.85+</div>
         <div class="text-xs text-slate-600 mb-4">~2 MB</div>
         <a
           :href="vsixURL"
           class="inline-flex items-center justify-center gap-1.5 w-full py-2 rounded-lg border border-cyan-500/40 text-cyan-300 hover:bg-cyan-600/10 text-sm font-semibold transition-colors"
         >
           <i class="pi pi-download text-xs"></i>
           .vsix
         </a>
       </div>
     </div>
     <p class="text-xs text-slate-600 mt-4 reveal">
       Or install from the command line: <code class="text-slate-400">code --install-extension nexus-orchestrator-vscode.vsix</code>
     </p>
   </section>
   ```

2. **Add a fourth entry to the `packages` array** (the "What's Included" section) in the
   `<script setup>` block:
   ```ts
   {
     icon: '🔌',
     title: 'VS Code Extension',
     items: [
       'Submit tasks directly from the editor',
       'Task queue tree view sidebar',
       'Status bar with live daemon health',
       'Provider &amp; model picker command',
       'Connects to daemon at <code class="text-slate-300">127.0.0.1:9999</code>',
     ],
   },
   ```
   Append this after the existing `CLI (nexus-cli)` entry in `packages`.

3. **Add `vsixURL` constant** in the `<script setup>` block (near `const baseURL`):
   ```ts
   const vsixURL = `${baseURL}/nexus-orchestrator-vscode.vsix`
   ```

4. **Update the "What's Included" grid** from `lg:grid-cols-3` to `lg:grid-cols-4`
   to accommodate the new fourth card:
   Change: `class="grid grid-cols-1 lg:grid-cols-3 gap-4"`
   To:     `class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4"`

## Acceptance Criteria

- [ ] A "VS Code Extension" section appears between "CLI + Daemon" and "Quick Install"
- [ ] The `.vsix` download button links to `nexus-orchestrator-vscode.vsix` on the release
- [ ] "What's Included" has 4 cards, the 4th being the VS Code Extension
- [ ] `npm run build` in `docs/` succeeds with no TypeScript errors

## Anti-patterns to Avoid

- Do not change existing `desktopCards`, `cliCards`, or `packages` entries
- Do not add OS-detection logic for the extension (it's platform-independent)
- Do not add a "Recommended" badge to the extension card
