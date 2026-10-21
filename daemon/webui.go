package daemon

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed webui
var webUIFiles embed.FS

// handleWebUI serves the web interface at /ui/. It is a single page that
// talks to the API on the same address, so it needs no CORS setup.
func handleWebUI(m *http.ServeMux) {
	files, err := fs.Sub(webUIFiles, "webui")
	if err != nil {
		panic(err)
	}

	m.Handle("GET /ui/", http.StripPrefix("/ui/", http.FileServerFS(files)))
	m.Handle("GET /ui", http.RedirectHandler("/ui/", http.StatusMovedPermanently))
}
