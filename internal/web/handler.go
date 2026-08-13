package web

import (
	"io/fs"
	"net/http"
	"strings"
)

const ContentSecurityPolicy = "default-src 'self'; base-uri 'none'; object-src 'none'; frame-ancestors 'none'; form-action 'self'; script-src 'self' 'wasm-unsafe-eval'; style-src 'self'; style-src-attr 'unsafe-inline'; style-src-elem 'self' 'unsafe-inline'; img-src 'self' data:; font-src 'self'; connect-src 'self'"

type httpHandler = http.Handler

type handler struct {
	files   fs.FS
	catalog *assetCatalog
	index   []byte
}

func NewHandler(files fs.FS) (http.Handler, error) {
	catalog, err := loadDistribution(files)
	if err != nil {
		return nil, err
	}
	index, err := fs.ReadFile(files, "index.html")
	if err != nil {
		return nil, err
	}
	return &handler{files: files, catalog: catalog, index: index}, nil
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	setSecurityHeaders(w.Header())
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.NotFound(w, r)
		return
	}
	if r.URL.Path == "/" {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Location", "/reviews")
		w.WriteHeader(http.StatusTemporaryRedirect)
		return
	}

	assetPath, valid := requestAssetPath(r.URL.Path)
	if valid {
		if data, err := fs.ReadFile(h.files, assetPath); err == nil {
			contentType, known := fixedContentType(assetPath)
			if !known {
				http.NotFound(w, r)
				return
			}
			w.Header().Set("Content-Type", contentType)
			if _, immutable := h.catalog.immutable[assetPath]; immutable {
				w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			} else {
				w.Header().Set("Cache-Control", "no-cache")
			}
			w.WriteHeader(http.StatusOK)
			if r.Method == http.MethodGet {
				_, _ = w.Write(data)
			}
			return
		}
	}

	if valid && acceptsHTML(r) && isNavigationPath(r.URL.Path) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		w.WriteHeader(http.StatusOK)
		if r.Method == http.MethodGet {
			_, _ = w.Write(h.index)
		}
		return
	}
	http.NotFound(w, r)
}

func setSecurityHeaders(header http.Header) {
	header.Set("Content-Security-Policy", ContentSecurityPolicy)
	header.Set("X-Content-Type-Options", "nosniff")
	header.Set("Referrer-Policy", "no-referrer")
}

func requestAssetPath(requestPath string) (string, bool) {
	if requestPath == "" || !strings.HasPrefix(requestPath, "/") || strings.ContainsAny(requestPath, "\\\x00") {
		return "", false
	}
	assetPath := strings.TrimPrefix(requestPath, "/")
	if assetPath == "" || !fs.ValidPath(assetPath) {
		return "", false
	}
	for segment := range strings.SplitSeq(assetPath, "/") {
		if segment == "." || segment == ".." || strings.HasPrefix(segment, ".") {
			return "", false
		}
	}
	return assetPath, true
}

func acceptsHTML(r *http.Request) bool {
	return strings.Contains(strings.ToLower(r.Header.Get("Accept")), "text/html")
}

func isNavigationPath(requestPath string) bool {
	if requestPath != "/reviews" && !strings.HasPrefix(requestPath, "/reviews/") && requestPath != "/analytics" {
		return false
	}
	for segment := range strings.SplitSeq(strings.TrimPrefix(requestPath, "/"), "/") {
		if strings.Contains(segment, ".") {
			return false
		}
	}
	return true
}

func fixedContentType(assetPath string) (string, bool) {
	extensions := map[string]string{
		".css":   "text/css; charset=utf-8",
		".html":  "text/html; charset=utf-8",
		".js":    "text/javascript; charset=utf-8",
		".json":  "application/json; charset=utf-8",
		".png":   "image/png",
		".svg":   "image/svg+xml",
		".wasm":  "application/wasm",
		".woff2": "font/woff2",
	}
	for extension, contentType := range extensions {
		if strings.HasSuffix(strings.ToLower(assetPath), extension) {
			return contentType, true
		}
	}
	return "", false
}
