---
id: TASK-478
plan: PLAN-062
status: done
wave: 3
priority: 3
---

# TASK-478: Fix AIActivityCard — undefined :class binding

**Problem:** `frontend/src/components/AIActivityCard.vue` defines `emoji()`, `borderClass()`, and `accentClass()` as computed functions with `switch` statements that have no `default` case. When an unknown or future `status`/`type` value is encountered, these functions implicitly return `undefined`. Vue binds `:class="borderClass()"` and emits a runtime warning; the `undefined` is silently dropped but the warning pollutes the console and can cause style regressions.

**Fix:**

1. Add `default: return ''` to the `switch` statement in `emoji()` — return a safe empty string or a neutral fallback emoji (e.g., `'⚙️'`)
2. Add `default: return ''` to the `switch` statement in `borderClass()` — return a neutral Tailwind class (e.g., `'border-gray-300'`)
3. Add `default: return ''` to the `switch` statement in `accentClass()` — return a neutral class (e.g., `'text-gray-500'`)
4. Update return type annotations from `string | undefined` (or implicit) to `string` on all three functions
5. Verify no Vue `:class` / `:style` binding sites in this component receive `undefined` (check all computed/method return types)

**Files:**

- `frontend/src/components/AIActivityCard.vue`

**Acceptance criteria:**

- No Vue runtime warning `[Vue warn]: Invalid prop: type check failed for prop "class"` in browser console
- All three functions have explicit `default` cases and return `string`
- `vue-tsc --noEmit` zero errors (return types are `string`)
