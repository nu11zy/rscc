package http

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"rscc/internal/common/scriptgen"
	"rscc/internal/common/validators"
	"rscc/internal/database/ent"
	"strings"
	"time"
)

// TODO: Improve logging
func (p *Protocol) RequestHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/" {
		p.ServeDefaultPage(w, r)
		return
	}

	// If ends with .sh, .ps1, .bat return script
	if strings.HasSuffix(r.URL.Path, ".sh") || strings.HasSuffix(r.URL.Path, ".ps1") || strings.HasSuffix(r.URL.Path, ".py") {
		err := p.DownloadScript(w, r)
		if err != nil {
			p.lg.Errorf("Failed to download script: %v", err)
			p.ServeDefaultPage(w, r)
			return
		}
		return
	}

	// Download agent
	if err := p.DownloadAgent(w, r); err != nil {
		p.lg.Errorf("Failed to download agent: %v", err)
		p.ServeDefaultPage(w, r)
		return
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

func (p *Protocol) DownloadAgent(w http.ResponseWriter, r *http.Request) error {
	agent, err := p.db.GetAgentByURL(r.Context(), r.URL.Path)
	if err != nil {
		if ent.IsNotFound(err) {
			p.lg.Debugf("Agent not found: %v", err)
			p.ServeDefaultPage(w, r)
			return nil
		}
		p.lg.Errorf("Failed to get agent by URL: %v", err)
		p.ServeDefaultPage(w, r)
		return fmt.Errorf("failed to get agent by URL: %v", err)
	}

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

	updateCtx, updateCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer updateCancel()
	if err := p.db.UpdateAgentDownloads(updateCtx, agent.ID); err != nil {
		p.lg.Errorf("Failed to update agent downloads: %v", err)
	}

	p.lg.Infof("Agent '%s' (%s) downloaded by %s (%s)", agent.Name, agent.ID, r.RemoteAddr, r.URL.Path)
	return nil
}

func (p *Protocol) DownloadScript(w http.ResponseWriter, r *http.Request) error {
	isSh := strings.HasSuffix(r.URL.Path, ".sh")
	isPs1 := strings.HasSuffix(r.URL.Path, ".ps1")
	isPy := strings.HasSuffix(r.URL.Path, ".py")

	var agentURL string
	switch {
	case isSh:
		agentURL = strings.TrimSuffix(r.URL.Path, ".sh")
	case isPs1:
		agentURL = strings.TrimSuffix(r.URL.Path, ".ps1")
	case isPy:
		agentURL = strings.TrimSuffix(r.URL.Path, ".py")
	}

	agent, err := p.db.GetAgentByURL(r.Context(), agentURL)
	if err != nil {
		if ent.IsNotFound(err) {
			p.lg.Debugf("Agent not found: %v", err)
			p.ServeDefaultPage(w, r)
			return nil
		}
		p.lg.Errorf("Failed to get agent by URL: %v", err)
		p.ServeDefaultPage(w, r)
		return fmt.Errorf("failed to get agent by URL: %v", err)
	}

	if !validators.ValidateFileExists(agent.Path) {
		return fmt.Errorf("agent file not found: %s", agent.Path)
	}

	var script string
	switch {
	case isSh:
		script, err = scriptgen.GenerateShScript(agent)
		if err != nil {
			return fmt.Errorf("failed to generate sh script: %w", err)
		}
	case isPy:
		script, err = scriptgen.GeneratePyScript(agent)
		if err != nil {
			return fmt.Errorf("failed to generate py script: %w", err)
		}
	case isPs1:
		script, err = scriptgen.GeneratePs1Script(agent)
		if err != nil {
			return fmt.Errorf("failed to generate ps1 script: %w", err)
		}
	default:
		return fmt.Errorf("unsupported script type: %s", r.URL.Path)
	}

	// return raw script
	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(script)))
	w.Write([]byte(script))

	p.lg.Infof("Agent script '%s' (%s) downloaded by %s (%s)", agent.Name, agent.ID, r.RemoteAddr, r.URL.Path)
	return nil
}
