package main

import "embed"

// dist is the web app as Vite builds it: `npm run build` in web/ writes
// web/dist, and the Go build embeds it, so the plugin's binary — and so the
// release tera plugin pack writes — carries the app it serves. Build the app
// before the binary: the release workflow does, in that order.
//
// tera plugin new wrote a placeholder web/dist/index.html so the plugin
// builds before the app first does; the app's build replaces it. web/dist is
// a build, never committed (.gitignore), so a fresh checkout builds the app
// first.
//
//go:embed all:web/dist
var dist embed.FS
