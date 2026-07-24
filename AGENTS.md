# AGENTS.md

Behavioral guidelines to reduce common LLM coding mistakes. Merge with project-specific instructions as needed.

**Tradeoff:** These guidelines bias toward caution over speed. For trivial tasks, use judgment.

## Communication

- Reply to the user in Chinese unless the user explicitly asks for another language.

## 1. Think Before Coding

**Don't assume. Don't hide confusion. Surface tradeoffs.**

Before implementing:
- State your assumptions explicitly. If uncertain, ask.
- If multiple interpretations exist, present them - don't pick silently.
- If a simpler approach exists, say so. Push back when warranted.
- If something is unclear, stop. Name what's confusing. Ask.

## 2. Simplicity First

**Minimum code that solves the problem. Nothing speculative.**

- No features beyond what was asked.
- No abstractions for single-use code.
- No "flexibility" or "configurability" that wasn't requested.
- No error handling for impossible scenarios.
- If you write 200 lines and it could be 50, rewrite it.

Ask yourself: "Would a senior engineer say this is overcomplicated?" If yes, simplify.

## 3. Surgical Changes

**Touch only what you must. Clean up only your own mess.**

When editing existing code:
- Don't "improve" adjacent code, comments, or formatting.
- Don't refactor things that aren't broken.
- Match existing style, even if you'd do it differently.
- If you notice unrelated dead code, mention it - don't delete it.
- Do not add descriptive text to pages, such as feature explanations, usage instructions, or decorative copy, unless explicitly requested.
- Do not put all application code in `App.tsx`; keep `App.tsx` as a composition shell and move feature UI, window chrome, animations, and reusable logic into focused files such as `src/components/*` when they grow beyond a few lines.
- Use Tailwind CSS for styling.

When your changes create orphans:
- Remove imports/variables/functions that YOUR changes made unused.
- Don't remove pre-existing dead code unless asked.

The test: Every changed line should trace directly to the user's request.

## 4. Global UI Design Specification

**Do not invent UI from scratch when a project component or trusted registry component already fits.**

For all UI work in this project:
- Treat the existing app components, CSS files, and design tokens as the local source of truth.
- Reuse existing project components when they fit the job. Build custom primitives when they better match the product interaction or visual direction.
- Before building a custom primitive, check whether a suitable component already exists under `src/components/*`.
- Use Tailwind utility classes for layout and spacing. Prefer semantic tokens such as `bg-background`, `text-foreground`, `text-muted-foreground`, `border-border`, `bg-primary`, and `bg-accent` over one-off raw color utilities.
- Derive surfaces, borders, text, accents, and states from the app's semantic color tokens instead of custom one-off colors.

Visual direction:
- The product should feel like a polished, cinematic desktop music player: dark glass, restrained neon accents, depth, glow, and smooth 3D-like layering.
- Keep colors unified and musical. Reuse the current palette direction: near-black surfaces, warm coral, amber/gold, soft periwinkle, and muted off-white. If a new color is needed, add a named CSS variable in `src/App.css` and reuse it consistently.
- Avoid random gradients, unrelated accent colors, single-use palettes, and decorative copy. Visual drama should come from lighting, depth, motion, typography, and composition rather than extra explanatory text.
- Keep text readable over visual effects. Do not let glow, blur, particles, canvas layers, or 3D elements obscure labels, controls, or hit targets.

Motion and 3D:
- React Bits components may be used for animated text, backgrounds, spotlight/glow effects, interaction effects, and other high-impact motion pieces when they improve the product feel.
- Treat React Bits as an enhancement layer, not a replacement for accessible, maintainable project UI primitives.
- When React Bits or another registry adds files, review the added files before wiring them into the app.
- For real 3D scenes, prefer the existing `three` and `@react-three/fiber` stack. Keep 3D scenes full-bleed or integrated into the app surface, not trapped in decorative preview cards.
- Respect `prefers-reduced-motion`. Motion should guide attention, not create layout shift, hide controls, or block keyboard/mouse use.

Implementation boundaries:
- Keep `App.tsx` as a composition shell. Put feature UI, window chrome, 3D scenes, React Bits wrappers, and reusable logic in focused files under `src/components/*` or `src/lib/*`.
- Use lucide icons inside icon buttons when a matching icon exists. Prefer icons over text-only tool buttons when the action is familiar.
- Do not add in-app feature explanations, usage instructions, or decorative marketing copy unless explicitly requested.
- For custom title bars and draggable window areas, call the Tauri native window dragging API (`getCurrentWindow().startDragging()`) instead of using CSS drag regions such as `-webkit-app-region: drag` or `data-tauri-drag-region`.
- Every visual change must work at desktop and narrow widths, with stable dimensions for fixed-format controls such as docks, title bars, playback controls, sliders, and toolbars.

## 5. Goal-Driven Execution

**Define success criteria. Loop until verified.**

Transform tasks into verifiable goals:
- "Add validation" → "Write tests for invalid inputs, then make them pass"
- "Fix the bug" → "Write a test that reproduces it, then make it pass"
- "Refactor X" → "Ensure tests pass before and after"

For multi-step tasks, state a brief plan:
```
1. [Step] → verify: [check]
2. [Step] → verify: [check]
3. [Step] → verify: [check]
```

Strong success criteria let you loop independently. Weak criteria ("make it work") require constant clarification.

## 6. Completion Check

After finishing implementation work in this project:
- Only run the compile/build check needed to confirm the app compiles.
- Do not run tests unless the user explicitly asks for tests.
- Do not start Vite for browser rendering checks.

---

**These guidelines are working if:** fewer unnecessary changes in diffs, fewer rewrites due to overcomplication, and clarifying questions come before implementation rather than after mistakes.
