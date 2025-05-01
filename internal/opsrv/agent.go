package opsrv

import (
	"archive/zip"
	"bytes"
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

const agentDir = "agents"

// agent list, agent generate, agent remove
func (s *OperatorServer) NewAgentCommand() *cobra.Command {
	agentCmd := &cobra.Command{
		Use:   "agent",
		Short: "Agent management",
		Args:  cobra.NoArgs,
	}

	// List agents
	agentListCmd := &cobra.Command{
		Use:   "list",
		Short: "List agents",
		Args:  cobra.NoArgs,
		RunE:  s.agentList,
	}

	// Generate agent
	agentNewCmd := &cobra.Command{
		Use:   "new",
		Short: "Generate new agent",
		Args:  cobra.NoArgs,
		RunE:  s.agentNew,
	}
	agentNewCmd.Flags().StringP("name", "n", utils.GetRandomName(), "Agent name (without extension)")
	agentNewCmd.Flags().StringP("os", "o", runtime.GOOS, "OS (windows, linux, darwin)")
	agentNewCmd.Flags().StringP("arch", "a", runtime.GOARCH, "Architecture (amd64, arm64)")
	agentNewCmd.Flags().String("addr", "", "Server address (e.g. 127.0.0.1:8080)")
	agentNewCmd.MarkFlagRequired("addr")

	// Remove agent
	agentRemoveCmd := &cobra.Command{
		Use:   "remove",
		Short: "Remove agent",
		Args:  cobra.NoArgs,
		RunE:  s.agentRemove,
	}

	agentCmd.AddCommand(agentListCmd)
	agentCmd.AddCommand(agentNewCmd)
	agentCmd.AddCommand(agentRemoveCmd)

	return agentCmd
}

func (s *OperatorServer) agentList(cmd *cobra.Command, args []string) error {
	// Get agents
	agents, err := s.db.GetAllAgents(cmd.Context())
	if err != nil {
		return fmt.Errorf("failed to get agents: %w", err)
	}
	if len(agents) == 0 {
		cmd.Println(pprint.Info("No agents found"))
		return nil
	}

	rows := make([][]string, len(agents))
	for i, agent := range agents {
		// Check if agent file exists
		agentBytes, err := os.ReadFile(filepath.Join(constants.AgentDir, agent.Name))
		if err != nil {
			if os.IsNotExist(err) {
				rows[i] = []string{agent.ID, pprint.ErrorColor.Sprint(agent.Name), agent.Os, agent.Arch, agent.Addr}
				continue
			} else {
				return fmt.Errorf("failed to read agent file %s: %w", agent.ID, err)
			}
		}

		// Check if agent file is modified
		agentHash := strconv.FormatUint(xxhash.Sum64(agentBytes), 10)
		if agentHash != agent.Xxhash {
			rows[i] = []string{agent.ID, pprint.WarnColor.Sprint(agent.Name), agent.Os, agent.Arch, agent.Addr}
		} else {
			rows[i] = []string{agent.ID, agent.Name, agent.Os, agent.Arch, agent.Addr}
		}
	}

	cmd.Println(pprint.Table([]string{"ID", "Name", "OS", "Arch", "Addr"}, rows))
	cmd.Printf("[%s] - deleted; [%s] - modified\n", pprint.ErrorColor.Sprint("*"), pprint.WarnColor.Sprint("*"))
	return nil
}

func (s *OperatorServer) agentRemove(cmd *cobra.Command, args []string) error {
	return nil
}

func (s *OperatorServer) agentNew(cmd *cobra.Command, args []string) error {
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
	addr, err := cmd.Flags().GetString("addr")
	if err != nil {
		return err
	}

	// Validate arguments
	if !validators.ValidateGOOS(goos) {
		return fmt.Errorf("invalid os: %s", goos)
	}
	if !validators.ValidateGOARCH(goarch) {
		return fmt.Errorf("invalid arch: %s", goarch)
	}
	if !validators.ValidateAddr(addr) {
		return fmt.Errorf("invalid addr: %s", addr)
	}

	// Check if agent with same name already exists
	name = strings.ReplaceAll(name, " ", "-")

	// Check db
	agent, err := s.db.GetAgentByName(cmd.Context(), name)
	if err == nil && agent != nil {
		s.lg.Warnf("Agent `%s` [id: %s] already exists", name, agent.ID)
		return fmt.Errorf("agent `%s` [id: %s] already exists", name, agent.ID)
	}

	// Check if agent with same name already exists
	if _, err := os.Stat(filepath.Join(agentDir, name)); !os.IsNotExist(err) {
		s.lg.Warnf("Agent with name `%s` not found in database. Replacing file `%s`", name, filepath.Join(agentDir, name))
		cmd.Println(pprint.Warn("Agent with name `%s` not found in database. Replacing file `%s`", name, filepath.Join(agentDir, name)))
	}
	s.lg.Infof("Generating agent `%s` for %s/%s (listener: %s)", name, goos, goarch, addr)
	cmd.Println(pprint.Info("Generating agent `%s` for %s/%s (listener: %s)", name, goos, goarch, addr))

	// Generate keys
	privKey, err := sshd.GeneratePrivateKey()
	if err != nil {
		return fmt.Errorf("failed to generate private key: %w", err)
	}
	pubKey, err := sshd.GeneratePublicKey(privKey)
	if err != nil {
		return fmt.Errorf("failed to generate public key: %w", err)
	}

	// Unzip agent
	tmpDir, err := s.unzipAgent()
	if err != nil {
		return fmt.Errorf("failed to unzip agent: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Build agent
	s.lg.Infof("Building agent")
	err = s.buildAgent(tmpDir, name, goos, goarch, addr, privKey, pubKey)
	if err != nil {
		return fmt.Errorf("failed to build agent: %w", err)
	}

	// Get agent hash
	agentBytes, err := os.ReadFile(filepath.Join(agentDir, name))
	if err != nil {
		return fmt.Errorf("failed to read agent: %w", err)
	}
	agentHash := strconv.FormatUint(xxhash.Sum64(agentBytes), 10)

	// Add agent to db
	agent, err = s.db.CreateAgent(cmd.Context(), name, goos, goarch, addr, agentHash, pubKey)
	if err != nil {
		return fmt.Errorf("failed to add agent to db: %w", err)
	}

	cmd.Println(pprint.Success("Agent `%s` [id: %s] generated (%s)", agent.Name, agent.ID, filepath.Join("./agents", name)))
	return nil
}

func (s *OperatorServer) unzipAgent() (string, error) {
	agentDir, err := os.MkdirTemp("", "agent-src")
	if err != nil {
		return "", fmt.Errorf("create temp dir: %w", err)
	}

	zipReader, err := zip.NewReader(bytes.NewReader(rscc.ZipAgentSource), int64(len(rscc.ZipAgentSource)))
	if err != nil {
		return "", fmt.Errorf("create zip reader: %w", err)
	}

	for _, file := range zipReader.File {
		s.lg.Debugf("Unzipping %s", file.Name)

		rc, err := file.Open()
		if err != nil {
			return "", fmt.Errorf("failed to open file: %w", err)
		}
		defer rc.Close()
		filePath := filepath.Join(agentDir, file.Name)

		if file.FileInfo().IsDir() {
			err = os.MkdirAll(filePath, 0777)
			if err != nil {
				return "", fmt.Errorf("failed to create dir: %w", err)
			}
			continue
		}

		unzippedFile, err := os.Create(filePath)
		if err != nil {
			return "", fmt.Errorf("failed to create file: %w", err)
		}
		defer unzippedFile.Close()

		_, err = io.Copy(unzippedFile, rc)
		if err != nil {
			return "", fmt.Errorf("failed to copy file: %w", err)
		}
	}

	return agentDir, nil
}

func (s *OperatorServer) buildAgent(agentDir, name, goos, goarch, addr string, privKey, pubKey []byte) error {
	currentDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current dir: %w", err)
	}

	// Set ldflags
	ldflags := "-s -w"
	if goos == "windows" {
		ldflags = fmt.Sprintf("%s -H windowsgui", ldflags)
	}
	ldflags = fmt.Sprintf("%s -X main.serverAddress=%s", ldflags, addr)
	ldflags = fmt.Sprintf("%s -buildid=", ldflags)

	// Build agent
	cmd := exec.Command(
		"go",
		"build",
		"-C",
		agentDir,
		"-o",
		filepath.Join(currentDir, "agents", name),
		"-mod=vendor",
		"-trimpath",
		fmt.Sprintf("-ldflags=%s", ldflags),
		"cmd/agent/main.go",
	)
	// Set environment variables
	cmd.Env = append(os.Environ(), "GOOS="+goos, "GOARCH="+goarch)

	// Build agent
	s.lg.Debugf("Running command: %s", cmd.String())
	output, err := cmd.CombinedOutput()
	if err != nil {
		err = fmt.Errorf("failed to run build: %w", err)
		if len(output) > 0 {
			err = fmt.Errorf("%w:\n%s", err, string(output))
		}
		return err
	}
	if len(output) > 0 {
		s.lg.Infof("Build output: %s", string(output))
	}

	return nil
}
