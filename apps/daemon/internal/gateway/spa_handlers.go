package gateway

import (
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"
)

func (s *Server) handleStaticSPA(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/api/") {
		http.NotFound(w, r)
		return
	}

	if s.webFS == nil {
		serveControlPlaneBanner(w)
		return
	}

	cleanPath := path.Clean(strings.TrimPrefix(r.URL.Path, "/"))
	if cleanPath == "." || cleanPath == "" {
		cleanPath = "index.html"
	}

	f, err := s.webFS.Open(cleanPath)
	if err == nil {
		defer f.Close()
		stat, err := f.Stat()
		if err == nil && !stat.IsDir() {
			http.FileServer(http.FS(s.webFS)).ServeHTTP(w, r)
			return
		}
	}

	fallback, err := s.webFS.Open("index.html")
	if err != nil {
		serveControlPlaneBanner(w)
		return
	}
	defer fallback.Close()

	stat, err := fallback.Stat()
	if err != nil {
		serveControlPlaneBanner(w)
		return
	}

	seeker, ok := fallback.(io.ReadSeeker)
	if !ok {
		serveControlPlaneBanner(w)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	http.ServeContent(w, r, "index.html", stat.ModTime(), seeker)
}

func serveControlPlaneBanner(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	banner := `<!DOCTYPE html>
<html>
<head><title>zenf</title></head>
<body style="font-family:sans-serif;background:#09090b;color:#e4e4e7;padding:40px;">
  <h2>zenf Control Plane Active</h2>
  <p>Status: Healthy (Headless/API mode)</p>
</body>
</html>`
	fmt.Fprint(w, banner)
}
