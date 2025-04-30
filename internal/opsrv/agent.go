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
	"rscc/internal/common/utils"
	"runtime"

	"github.com/spf13/cobra"
)

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
	agentNewCmd.Flags().StringP("name", "n", utils.GetRandomName(0), "Agent name")
	agentNewCmd.Flags().StringP("os", "o", runtime.GOOS, "OS")
	agentNewCmd.Flags().StringP("arch", "a", runtime.GOARCH, "Architecture")
	agentNewCmd.Flags().String("addr", "", "Address")
	// agentNewCmd.MarkFlagRequired("addr")

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
	operationSystem, err := cmd.Flags().GetString("os")
	if err != nil {
		return err
	}
	arch, err := cmd.Flags().GetString("arch")
	if err != nil {
		return err
	}
	addr, err := cmd.Flags().GetString("addr")
	if err != nil {
		return err
	}

	s.lg.Infof("Generating agent `%s` for %s/%s (addr: %s)", name, operationSystem, arch, addr)

	// Unzip agent
	agentDir, err := s.unzipAgent()
	if err != nil {
		return fmt.Errorf("unzip agent: %w", err)
	}
	// defer os.RemoveAll(agentDir)

	// Build agent
	s.lg.Infof("Building agent")
	err = s.buildAgent(agentDir)
	if err != nil {
		return fmt.Errorf("build agent: %w", err)
	}

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

func (s *OperatorServer) buildAgent(agentDir string) error {
	currentDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current dir: %w", err)
	}

	s.lg.Debugf("Building %s", filepath.Join(agentDir, "cmd/agent/main.go"))
	s.lg.Debugf("Agent output dir %s", filepath.Join(currentDir, "agent"))

	cmd := exec.Command("go", "build", "-C", agentDir, "-o", filepath.Join(currentDir, "agent"), "-mod=vendor", "cmd/agent/main.go")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to run build: %w (%s)", err, string(output))
	}
	s.lg.Debugf("Build output: %s", string(output))

	return nil
}
