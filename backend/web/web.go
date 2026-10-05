// Package web serves the built frontend out of the binary, so one artifact
// and one origin serve both the API and the app. That is not merely
// convenient: the session cookie is SameSite=Lax and the fetch client sends
// no cross-origin credentials, so splitting the two across origins would
// break sign-in. Same-origin is what the rest of the code already assumes.
//
// dist/ is produced by `npm run build` and copied here by the build (see the
// root Dockerfile and `make build-web`). Only a .gitkeep is committed, so a
// real build is never checked in and `go build` still works in a fresh
// checkout — in that case this serves a short notice instead of the app, and
// local development goes through the Vite dev server as it always has.
package web

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"

	"campaigntrackerpro/platform/httpx"
)

//go:embed all:dist
var embedded embed.FS

// notice is served when no frontend build is embedded. Having the binary
// explain itself beats a blank page or a 500, which is what "the API works
// but every route is empty" otherwise looks like.
const notice = `<!doctype html>
<meta charset="utf-8">
<title>Campaign Tracker Pro — frontend not built</title>
<style>body{font:15px/1.6 system-ui,sans-serif;max-width:34rem;margin:15vh auto;padding:0 1.5rem}
code{background:#eef0f6;padding:.15em .4em;border-radius:4px}</style>
<h1>No frontend build embedded</h1>
<p>The API is running, but this binary carries no compiled frontend.</p>
<p>For local development run <code>make dev</code> and use the Vite dev server
on <code>:5173</code>, which proxies <code>/api</code> here. To serve the real
app from this port instead, run <code>make build-web</code> and restart.</p>
`

// Assets is the built frontend, rooted so "index.html" is at the top.
func Assets() (fs.FS, error) { return fs.Sub(embedded, "dist") }

// Built reports whether a real frontend build is embedded. The composition
// root logs it at startup so a misbuilt image is obvious from the first line
// of logs rather than from a blank browser tab.
func Built() bool {
	assets, err := Assets()
	if err != nil {
		return false
	}
	_, err = fs.Stat(assets, "index.html")
	return err == nil
}

// Handler serves the SPA. A request matching a real file gets that file;
// anything else gets index.html, so a refresh on /campaigns/cred is routed by
// the client instead of 404ing.
//
// apiPrefixes are the paths that must never fall back to the shell: an
// unmatched API route has to stay a JSON 404, or a typo in a fetch call
// returns an HTML page and the client reports "unexpected token <" instead of
// the actual problem.
func Handler(apiPrefixes ...string) http.Handler {
	assets, err := Assets()
	if err != nil {
		return placeholder()
	}
	index, err := fs.ReadFile(assets, "index.html")
	if err != nil {
		return placeholder()
	}
	files := http.FileServer(http.FS(assets))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for _, p := range apiPrefixes {
			if r.URL.Path == p || strings.HasPrefix(r.URL.Path, p+"/") {
				// JSON, not net/http's text/plain, so an unmatched API path
				// looks like every other API error to the client.
				httpx.WriteNotFound(w, "not found")
				return
			}
		}

		if name := strings.TrimPrefix(r.URL.Path, "/"); name != "" {
			if f, ferr := assets.Open(name); ferr == nil {
				info, serr := f.Stat()
				_ = f.Close()
				if serr == nil && !info.IsDir() {
					// Vite fingerprints everything under assets/, so those are
					// safe to cache hard. Anything else may be replaced in
					// place by the next deploy.
					if strings.HasPrefix(name, "assets/") {
						w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
					}
					files.ServeHTTP(w, r)
					return
				}
			}
		}

		// The shell must never be cached, or a deploy leaves browsers asking
		// for asset filenames that no longer exist.
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(index)
	})
}

func placeholder() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(notice))
	})
}
