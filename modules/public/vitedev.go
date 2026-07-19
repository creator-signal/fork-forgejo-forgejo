// Copyright 2026 The Forgejo Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package public

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"forgejo.org/modules/log"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/web/routing"
)

const viteDevPortFile = "public/assets/.vite/dev-port"

// classicEntryVirtualPaths maps each non-module (classic <script>, or `new SharedWorker(...)`)
// build entry's unhashed output name to the IN-MEMORY BUNDLED virtual path that
// `classicEntryPlugin.configureServer` in vite.config.js serves for it.
//
// This must NOT point at the raw source file (e.g. "web_src/js/iife.js"): Vite's dev server always
// serves raw source through its native-ESM transform pipeline (adding `import`/`export`), which a
// classic (non-`type="module"`) <script> tag cannot parse. classicEntryPlugin instead runs a real
// `build()` in iife format for these entries and serves the bundled result from a virtual path, so
// the classic <script> tag keeps working exactly as it does in production.
var classicEntryVirtualPaths = map[string]string{
	"js/iife.js":                     "web_src/js/__vite_classic_iife.js",
	"js/webcomponents.js":            "web_src/js/__vite_classic_webcomponents.js",
	"js/swagger.js":                  "web_src/js/__vite_classic_swagger.js",
	"js/forgejoswagger.js":           "web_src/js/__vite_classic_forgejoswagger.js",
	"js/eventsource.sharedworker.js": "web_src/js/__vite_classic_eventsource.sharedworker.js",
}

var viteDevProxy atomic.Pointer[httputil.ReverseProxy]

func getViteDevProxy() *httputil.ReverseProxy {
	if proxy := viteDevProxy.Load(); proxy != nil {
		return proxy
	}

	portFile := filepath.Join(setting.StaticRootPath, viteDevPortFile)
	data, err := os.ReadFile(portFile)
	if err != nil {
		return nil
	}
	port := strings.TrimSpace(string(data))
	if port == "" {
		return nil
	}

	target, err := url.Parse("http://localhost:" + port)
	if err != nil {
		log.Error("Failed to parse Vite dev server URL: %v", err)
		return nil
	}

	transport := &http.Transport{
		IdleConnTimeout:       5 * time.Second,
		ResponseHeaderTimeout: 5 * time.Second,
	}
	log.Info("Proxying Vite dev server requests to %s", target)
	proxy := &httputil.ReverseProxy{
		Transport: transport,
		Rewrite: func(r *httputil.ProxyRequest) {
			r.SetURL(target)
			r.Out.Host = target.Host
		},
		ModifyResponse: func(resp *http.Response) error {
			// add a header to indicate the Vite dev server port,
			// make developers know that this request is proxied to Vite dev server and which port it is
			resp.Header.Add("X-Forgejo-Vite-Port", port)
			return nil
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			log.Error("Error proxying to Vite dev server: %v", err)
			http.Error(w, "Error proxying to Vite dev server: "+err.Error(), http.StatusBadGateway)
		},
	}
	viteDevProxy.Store(proxy)
	return proxy
}

// ViteDevMiddleware proxies matching requests to the Vite dev server.
// It is registered as middleware in non-production mode and lazily discovers
// the Vite dev server port from the port file written by the viteDevServerPortPlugin.
func ViteDevMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
		if !isViteDevRequest(req) {
			next.ServeHTTP(resp, req)
			return
		}
		proxy := getViteDevProxy()
		if proxy == nil {
			next.ServeHTTP(resp, req)
			return
		}
		routing.MarkLongPolling(resp, req)
		proxy.ServeHTTP(resp, req)
	})
}

// isViteDevMode returns true if the Vite dev server port file exists.
// In production mode, the result is cached after the first check.
func isViteDevMode() bool {
	if setting.IsProd {
		return false
	}
	portFile := filepath.Join(setting.StaticRootPath, viteDevPortFile)
	_, err := os.Stat(portFile)
	return err == nil
}

func viteDevSourceURL(name string) string {
	if !isViteDevMode() {
		return ""
	}
	if strings.HasPrefix(name, "css/theme-") {
		// Only redirect built-in themes to Vite source; custom themes are served from custom/public/assets/css/
		themeFile := strings.TrimPrefix(name, "css/")
		srcPath := filepath.Join(setting.StaticRootPath, "web_src/css/themes", themeFile)
		if _, err := os.Stat(srcPath); err == nil {
			return setting.AppSubURL + "/web_src/css/themes/" + themeFile
		}
		return ""
	}
	if name == "css/swagger.css" {
		return setting.AppSubURL + "/web_src/css/standalone/swagger.css"
	}
	if name == "css/index.css" {
		return setting.AppSubURL + "/web_src/css/index.css"
	}
	if src, ok := classicEntryVirtualPaths[name]; ok {
		return setting.AppSubURL + "/" + src
	}
	if name == "js/index.js" {
		return setting.AppSubURL + "/web_src/js/index.js"
	}
	return ""
}

// isViteDevRequest returns true if the request should be proxied to the Vite dev server.
// Ref: Vite source packages/vite/src/node/constants.ts and packages/vite/src/shared/constants.ts
func isViteDevRequest(req *http.Request) bool {
	if req.Header.Get("Upgrade") == "websocket" {
		wsProtocol := req.Header.Get("Sec-WebSocket-Protocol")
		return wsProtocol == "vite-hmr" || wsProtocol == "vite-ping"
	}
	path := req.URL.Path

	// vite internal requests
	if strings.HasPrefix(path, "/@vite/") /* HMR client */ ||
		strings.HasPrefix(path, "/@fs/") /* out-of-root file access, see vite.config.js: server.fs.allow */ ||
		strings.HasPrefix(path, "/@id/") /* virtual modules */ {
		return true
	}

	// local source requests (VITE-DEV-SERVER-SECURITY: don't serve sensitive files outside the allowed paths)
	if strings.HasPrefix(path, "/node_modules/") ||
		strings.HasPrefix(path, "/public/assets/") ||
		strings.HasPrefix(path, "/web_src/") {
		return true
	}

	// Vite uses a path relative to project root and adds "?import" to non-JS/CSS asset imports:
	// - {WebSite}/public/assets/... (e.g. SVG icons from "{RepoRoot}/public/assets/img/svg/")
	// - {WebSite}/assets/emoji.json: it is an exception for the frontend assets, it is imported by JS code, but:
	//   - KEEP IN MIND: all static frontend assets are served from "{AssetFS}/assets" to "{WebSite}/assets" by Forgejo Web Server
	//   - "{AssetFS}" is a layered filesystem from "{RepoRoot}/public" or embedded assets, and user's custom files in "{CustomPath}/public"
	//   - "{RepoRoot}/assets/emoji.json" just happens to have the dir name "assets", it is not related to frontend assets
	//   - BAD DESIGN: indeed it is a "conflicted and polluted name" sample
	if strings.HasPrefix(path, "/assets/img/svg/") {
		return true
	}
	if path == "/assets/emoji.json" {
		return true
	}
	return false
}
