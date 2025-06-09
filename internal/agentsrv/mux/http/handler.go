package http

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"rscc/internal/common/validators"
	"rscc/internal/database/ent"
)

// TODO: Improve logging
func (p *Protocol) RequestHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/" {
		p.ServeDefaultPage(w, r)
		return
	}

	// Get agent by URL
	agent, err := p.db.GetAgentByURL(r.Context(), r.URL.Path)
	if err != nil {
		if ent.IsNotFound(err) {
			p.lg.Debugf("Agent not found: %v", err)
			p.ServeDefaultPage(w, r)
			return
		}
		p.lg.Errorf("Failed to get agent by URL: %v", err)
		p.ServeDefaultPage(w, r)
		return
	}

	// Download agent
	if err := p.DownloadAgent(w, r, agent); err != nil {
		p.lg.Errorf("Failed to download agent: %v", err)
		p.ServeDefaultPage(w, r)
		return
	}

	// Update agent downloads
	if err := p.db.UpdateAgentDownloads(r.Context(), agent.ID); err != nil {
		p.lg.Errorf("Failed to update agent downloads: %v", err)
	}
}

func (p *Protocol) ServeDefaultPage(w http.ResponseWriter, r *http.Request) {
	p.lg.Infof("Serving default page for %s (%s)", r.RemoteAddr, r.URL.Path)
	if p.htmlPagePath == "" {
		w.WriteHeader(http.StatusBadGateway)
		w.Write([]byte(`
			<html>
				<head><title>502 Bad Gateway</title></head>
				<body>
					<center><h1>502 Bad Gateway</h1></center>
					<hr><center>nginx</center>
				</body>
			</html>
			<!-- a padding to disable MSIE and Chrome friendly error page -->
			<!-- a padding to disable MSIE and Chrome friendly error page -->
			<!-- a padding to disable MSIE and Chrome friendly error page -->
			<!-- a padding to disable MSIE and Chrome friendly error page -->
			<!-- a padding to disable MSIE and Chrome friendly error page -->
			<!-- a padding to disable MSIE and Chrome friendly error page -->
		`))
	} else {
		p.fileServer.ServeHTTP(w, r)
	}
}

func (p *Protocol) DownloadAgent(w http.ResponseWriter, r *http.Request, agent *ent.Agent) error {
	p.lg.Infof("Downloading agent for %s (%s)", r.RemoteAddr, r.URL.Path) // TODO: add user-agent, remote-addr, agent name, agent id

	if !validators.ValidateFileExists(agent.Path) {
		return fmt.Errorf("agent file not found: %s", agent.Path)
	}

	file, err := os.Open(agent.Path)
	if err != nil {
		return fmt.Errorf("failed to open agent file: %w", err)
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to get file info: %w", err)
	}

	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", agent.Name))
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", fileInfo.Size()))

	if _, err := io.Copy(w, file); err != nil {
		return fmt.Errorf("failed to send file: %w", err)
	}

	return nil
}
