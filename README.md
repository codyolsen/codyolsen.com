# codyolsen.com

The game is served at `/game/`. The homepage redirects there.

## Game files

Edit the source files in `static/game/`; Hugo copies this directory to
`public/game/` when building the site.

```text
static/game/
├── index.html
├── css/game.css
├── js/game.js
└── assets/audio/
    ├── boom.m4a
    ├── boom.mp3
    └── laugh.mp3
```

Preview the game using `make run` (requires Go; no Hugo build needed),
then visit `http://localhost:8000/game/`. Stop the server with Ctrl+C.
Use `make run PORT=8080` to choose another port. Refresh the browser after edits.

The game selects a mobile performance profile for devices with a coarse primary
pointer. That profile limits canvas resolution, simultaneous sounds, balls and
effects, and omits launch reverb. Desktop keeps its original effect intensity,
20-ball limit, 60-point trails and full launch audio. Both profiles reuse audio
buffers, disconnect finished sounds and pause rendering/audio in background tabs.

Run the game regression checks with `node --test tests/game.test.cjs` (Node.js).
These simulate browser APIs to check resource bounds, input cancellation,
background/resume and refresh-rate-independent physics; validate visual performance
and sound on real mobile devices before release.

Build the full site using `hugo --gc --minify -b https://codyolsen.com/`.
The `public/` directory is generated output and is not tracked by Git.

The CloudFront function in `infra/infra.go` maps `/game` and `/game/` to
`/game/index.html` for the private S3 origin. Deploy the infrastructure change
alongside the site when first moving the game; the regular GitHub Actions workflow
only uploads site files and invalidates the cache.
