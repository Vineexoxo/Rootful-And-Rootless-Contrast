package utils

import (
	"context"
	"os/exec"
	"strconv"
	"strings"

	"go.uber.org/zap"
)

type CommandExecutor interface {
	Execute(ctx context.Context, command string, args ...string) ([]byte, error)

	// System metrics methods
	GetCPUUsage(ctx context.Context) ([]byte, error)
	GetMemoryUsage(ctx context.Context) ([]byte, error)
	GetDiskUsage(ctx context.Context, path string) ([]byte, error)
	GetNetworkStats(ctx context.Context) ([]byte, error)
	GetSystemUptime(ctx context.Context) ([]byte, error)

	// Container metrics methods
	GetDockerStats(ctx context.Context, containerName string) ([]byte, error)
	GetPodmanStats(ctx context.Context, containerName string) ([]byte, error)

	// Network testing methods
	PingHost(ctx context.Context, host string, count int) ([]byte, error)
	GetProcessInfo(ctx context.Context, pid string) ([]byte, error)
}

type SystemCommandExecutor struct {
	logger *zap.Logger
}

func NewSystemCommandExecutor(logger *zap.Logger) *SystemCommandExecutor {
	return &SystemCommandExecutor{
		logger: logger,
	}
}

// Execute executes a command and returns the output
// Args:
// - ctx: context.Context
// - command: string
// - args: []string
// Returns:
// - []byte: output of the command
// - error: error if the command fails
func (e *SystemCommandExecutor) Execute(ctx context.Context, command string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, command, args...)

	e.logger.Debug("Executing command",
		zap.String("command", command),
		zap.Strings("args", args),
	)

	output, err := cmd.Output()
	if err != nil {
		e.logger.Error("Command execution failed",
			zap.String("command", command),
			zap.Strings("args", args),
			zap.Error(err),
		)
		return nil, err
	}

	return output, nil
}

// Helper functions for common system commands

// GetCPUUsage gets CPU usage on Linux
// The command it runs is:
// - top -bn1
func (e *SystemCommandExecutor) GetCPUUsage(ctx context.Context) ([]byte, error) {
	// Use top command to get CPU usage on Linux
	return e.Execute(ctx, "top", "-bn1")
}

// GetMemoryUsage gets memory usage on Linux
// The command it runs is:
// - free -b
func (e *SystemCommandExecutor) GetMemoryUsage(ctx context.Context) ([]byte, error) {
	// Use free command on Linux
	return e.Execute(ctx, "free", "-b")
}

// GetDockerStats gets Docker stats
// The command it runs is:
// - docker stats --no-stream --format "table {{.Container}}\\t{{.CPUPerc}}\\t{{.MemUsage}}\\t{{.NetIO}}\\t{{.BlockIO}}"
func (e *SystemCommandExecutor) GetDockerStats(ctx context.Context, containerName string) ([]byte, error) {
	if containerName == "" {
		return e.Execute(ctx, "docker", "stats", "--no-stream", "--format", "table {{.Container}}\\t{{.CPUPerc}}\\t{{.MemUsage}}\\t{{.NetIO}}\\t{{.BlockIO}}")
	}
	return e.Execute(ctx, "docker", "stats", "--no-stream", "--format", "table {{.Container}}\\t{{.CPUPerc}}\\t{{.MemUsage}}\\t{{.NetIO}}\\t{{.BlockIO}}", containerName)
}

// GetPodmanStats gets Podman stats
// The command it runs is:
// - podman stats --no-stream --format "table {{.Name}}\\t{{.CPUPerc}}\\t{{.MemUsage}}\\t{{.NetIO}}\\t{{.BlockIO}}"
func (e *SystemCommandExecutor) GetPodmanStats(ctx context.Context, containerName string) ([]byte, error) {
	if containerName == "" {
		return e.Execute(ctx, "podman", "stats", "--no-stream", "--format", "table {{.Name}}\\t{{.CPUPerc}}\\t{{.MemUsage}}\\t{{.NetIO}}\\t{{.BlockIO}}")
	}
	return e.Execute(ctx, "podman", "stats", "--no-stream", "--format", "table {{.Name}}\\t{{.CPUPerc}}\\t{{.MemUsage}}\\t{{.NetIO}}\\t{{.BlockIO}}", containerName)
}

// GetNetworkStats gets network stats
// The command it runs is:
// - netstat -i
func (e *SystemCommandExecutor) GetNetworkStats(ctx context.Context) ([]byte, error) {
	// Get network interface statistics
	return e.Execute(ctx, "netstat", "-i")
}

// PingHost pings a host
// The command it runs is:
// - ping -c count host
func (e *SystemCommandExecutor) PingHost(ctx context.Context, host string, count int) ([]byte, error) {
	return e.Execute(ctx, "ping", "-c", strconv.Itoa(count), host)
}

// GetProcessInfo gets process info
// The command it runs is:
// - ps -p pid -o pid,ppid,user,cpu,mem,command
func (e *SystemCommandExecutor) GetProcessInfo(ctx context.Context, pid string) ([]byte, error) {
	return e.Execute(ctx, "ps", "-p", pid, "-o", "pid,ppid,user,cpu,mem,command")
}

// GetSystemUptime gets system uptime
// The command it runs is:
// - uptime
func (e *SystemCommandExecutor) GetSystemUptime(ctx context.Context) ([]byte, error) {
	return e.Execute(ctx, "uptime")
}

// GetDiskUsage gets disk usage
// The command it runs is:
// - df -h /
func (e *SystemCommandExecutor) GetDiskUsage(ctx context.Context, path string) ([]byte, error) {
	if path == "" {
		path = "/"
	}
	return e.Execute(ctx, "df", "-h", path)
}

// ParseCommandOutput provides utilities to parse common command outputs
func ParseCommandOutput(output []byte, delimiter string) []string {
	lines := strings.Split(string(output), "\n")
	var result []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			result = append(result, line)
		}
	}
	return result
}
