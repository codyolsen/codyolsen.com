---
name: verify
description: Verify the homepage canvas animation (static/index.html) in a headless browser — perf, visuals, and interaction.
---

# Verifying the canvas animation

The site is a single-page Canvas2D animation. Source of truth is `static/index.html`;
after editing, `cp static/index.html layouts/index.html` and re-template the one
divergent line: the build-stamp console.log (Hugo `now.Format` template in layouts,
"dev copy" placeholder in static). Hugo renders the homepage from layouts/, so the
layouts copy is what ships; public/ is CI build output (gitignored).

## Handle

```bash
# serve the page (no build step needed)
cd static && python3 -m http.server 8077 --bind 127.0.0.1 &

# headless browser: install playwright + chromium into the scratchpad dir
npm install playwright && npx playwright install chromium --with-deps
```

## Drive

- Load `http://127.0.0.1:8077/index.html`, wait ~2.2s (3 seed balls spawn by 1.6s).
- Launch balls with mouse drags (mouse events work even under touch emulation —
  the page has both handler sets).
- Mobile perf check: `devices['Pixel 5']` context + CDP
  `Emulation.setCPUThrottlingRate {rate: 6}`. Measure fps with a rAF counter via
  `page.evaluate`. Baselines (2026-07): pre-optimization 10fps, post 44fps; desktop 60fps.
- The `#debug` HUD shows fps/pos/vel — a frozen HUD means the rAF loop died
  (uncaught exception before the trailing `requestAnimationFrame`).
- Mega-drag corner-to-corner (dist >~1400px → power >75000) exercises the
  freeze + shockwave path.
- Collect `page.on('pageerror')` — the animation has no error handling; any
  exception in `loop()` kills the animation permanently.

## Gotchas

- Timing race worth re-testing after edits to the freeze path: rAF timestamps can
  lag `performance.now()` captured in event handlers. Deterministic repro: init
  script `performance.now = () => orig() + 50` then mega-drag.
- `boom.m4a` 404s when serving only static/ — harmless (play() is caught).
