package collectors

import (
	"context"
	"regexp"
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

type ContainerCollector struct {
	deps *CollectorDependencies

	// Prometheus metrics
	containerCPU     *prometheus.GaugeVec
	containerMemory  *prometheus.GaugeVec
	containerNetIO   *prometheus.GaugeVec
	containerBlockIO *prometheus.GaugeVec
	containerStatus  *prometheus.GaugeVec
}

// NewContainerCollector creates a new ContainerCollector
// Args:
// - deps: CollectorDependencies
// Returns:
// - *ContainerCollector: new ContainerCollector instance
func NewContainerCollector(deps *CollectorDependencies) *ContainerCollector {
	return &ContainerCollector{
		deps: deps,
		containerCPU: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "container_cpu_usage_percent",
				Help: "Container CPU usage percentage",
			},
			[]string{"container", "runtime"}, // container name, docker/podman
		),
		containerMemory: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "container_memory_usage_bytes",
				Help: "Container memory usage in bytes",
			},
			[]string{"container", "runtime", "type"}, // used, limit
		),
		containerNetIO: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "container_network_io_bytes",
				Help: "Container network I/O in bytes",
			},
			[]string{"container", "runtime", "direction"}, // rx, tx
		),
		containerBlockIO: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "container_block_io_bytes",
				Help: "Container block I/O in bytes",
			},
			[]string{"container", "runtime", "direction"}, // read, write
		),
		containerStatus: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "container_running",
				Help: "Container running status (1 for running, 0 for stopped)",
			},
			[]string{"container", "runtime"},
		),
	}
}

func (c *ContainerCollector) Name() string {
	return "container"
}

func (c *ContainerCollector) Describe(ch chan<- *prometheus.Desc) {
	c.containerCPU.Describe(ch)
	c.containerMemory.Describe(ch)
	c.containerNetIO.Describe(ch)
	c.containerBlockIO.Describe(ch)
	c.containerStatus.Describe(ch)
}

func (c *ContainerCollector) Collect(ch chan<- prometheus.Metric) {
	c.containerCPU.Collect(ch)
	c.containerMemory.Collect(ch)
	c.containerNetIO.Collect(ch)
	c.containerBlockIO.Collect(ch)
	c.containerStatus.Collect(ch)
}

// CollectMetrics collects container metrics
// This is the main function that collects all the container metrics
// The commands it runs are:
// - docker stats --no-stream --format "table {{.Container}}\\t{{.CPUPerc}}\\t{{.MemUsage}}\\t{{.NetIO}}\\t{{.BlockIO}}"
// - podman stats --no-stream --format "table {{.Name}}\\t{{.CPUPerc}}\\t{{.MemUsage}}\\t{{.NetIO}}\\t{{.BlockIO}}"
func (c *ContainerCollector) CollectMetrics(ctx context.Context) error {
	c.deps.Logger.Debug("Collecting container metrics")

	// Collect Docker metrics if enabled
	if c.deps.Config.Containers.DockerEnabled {
		if err := c.collectDockerMetrics(ctx); err != nil {
			c.deps.Logger.Error("Failed to collect Docker metrics", zap.Error(err))
		}
	}

	// Collect Podman metrics if enabled
	if c.deps.Config.Containers.PodmanEnabled {
		if err := c.collectPodmanMetrics(ctx); err != nil {
			c.deps.Logger.Error("Failed to collect Podman metrics", zap.Error(err))
		}
	}

	return nil
}

// collectDockerMetrics collects Docker metrics
// If MonitoredNames is specified, it gets stats only for those containers
// Otherwise, it gets stats for all containers
func (c *ContainerCollector) collectDockerMetrics(ctx context.Context) error {
	// If specific containers are configured, get stats for each one
	if len(c.deps.Config.Containers.MonitoredNames) > 0 {
		for _, containerName := range c.deps.Config.Containers.MonitoredNames {
			// Skip ignored containers
			if c.isContainerIgnored(containerName) {
				continue
			}

			output, err := c.deps.Executor.GetDockerStats(ctx, containerName)
			if err != nil {
				c.deps.Logger.Warn("Failed to get stats for container",
					zap.String("container", containerName),
					zap.Error(err))
				continue
			}

			if err := c.parseContainerStats(string(output), "docker"); err != nil {
				c.deps.Logger.Warn("Failed to parse stats for container",
					zap.String("container", containerName),
					zap.Error(err))
			}
		}
		return nil
	}

	// Get stats for all containers
	output, err := c.deps.Executor.GetDockerStats(ctx, "")
	if err != nil {
		return err
	}

	return c.parseContainerStats(string(output), "docker")
}

// collectPodmanMetrics collects Podman metrics
// If MonitoredNames is specified, it gets stats only for those containers
// Otherwise, it gets stats for all containers
func (c *ContainerCollector) collectPodmanMetrics(ctx context.Context) error {
	// If specific containers are configured, get stats for each one
	if len(c.deps.Config.Containers.MonitoredNames) > 0 {
		for _, containerName := range c.deps.Config.Containers.MonitoredNames {
			// Skip ignored containers
			if c.isContainerIgnored(containerName) {
				continue
			}

			output, err := c.deps.Executor.GetPodmanStats(ctx, containerName)
			if err != nil {
				c.deps.Logger.Warn("Failed to get stats for container",
					zap.String("container", containerName),
					zap.Error(err))
				continue
			}

			if err := c.parseContainerStats(string(output), "podman"); err != nil {
				c.deps.Logger.Warn("Failed to parse stats for container",
					zap.String("container", containerName),
					zap.Error(err))
			}
		}
		return nil
	}

	// Get stats for all containers
	output, err := c.deps.Executor.GetPodmanStats(ctx, "")
	if err != nil {
		return err
	}

	return c.parseContainerStats(string(output), "podman")
}

// parseContainerStats parses container stats
// This is the main function that parses the container stats
// Example: "artisan-agent-api   1.24%     601.9MiB / 7.654GiB   12.9kB / 6.34kB   164MB / 0B"
func (c *ContainerCollector) parseContainerStats(output, runtime string) error {
	c.deps.Logger.Debug("Parsing container stats",
		zap.String("runtime", runtime),
		zap.String("output", output))

	lines := strings.Split(output, "\n")

	for i, line := range lines {
		if i == 0 || strings.TrimSpace(line) == "" {
			c.deps.Logger.Debug("Skipping line",
				zap.Int("line_number", i),
				zap.String("line", line),
				zap.String("reason", "header or empty"))
			continue // Skip header and empty lines
		}
 
		c.deps.Logger.Debug("Processing container stats line",
			zap.Int("line_number", i),
			zap.String("line", line))

		// Use regex to parse the line properly, handling spaces within fields
		// Format: CONTAINER   CPU%   MEM_USAGE / MEM_LIMIT   NET_RX / NET_TX   BLOCK_READ / BLOCK_WRITE
		re := regexp.MustCompile(`^(\S+)\s+([\d.]+%)\s+([\d.]+\w+)\s+/\s+([\d.]+\w+)\s+([\d.]+\w+)\s+/\s+([\d.]+\w+)\s+([\d.]+\w+)\s+/\s+([\d.]+\w+)`)
		matches := re.FindStringSubmatch(strings.TrimSpace(line))

		c.deps.Logger.Debug("Regex parsing result",
			zap.String("line", strings.TrimSpace(line)),
			zap.Int("matches_count", len(matches)),
			zap.Strings("matches", matches))

		if len(matches) != 9 {
			c.deps.Logger.Warn("Failed to parse container stats line",
				zap.String("line", line),
				zap.Int("expected_matches", 9),
				zap.Int("actual_matches", len(matches)),
				zap.Strings("matches", matches))
			continue
		}

		containerName := matches[1]
		cpuStr := strings.TrimSuffix(matches[2], "%")
		memUsed := matches[3]
		memLimit := matches[4]
		netRx := matches[5]
		netTx := matches[6]
		blockRead := matches[7]
		blockWrite := matches[8]

		c.deps.Logger.Debug("Parsed container data",
			zap.String("container", containerName),
			zap.String("cpu_str", cpuStr),
			zap.String("mem_used", memUsed),
			zap.String("mem_limit", memLimit),
			zap.String("net_rx", netRx),
			zap.String("net_tx", netTx),
			zap.String("block_read", blockRead),
			zap.String("block_write", blockWrite))

		// Parse CPU usage
		if cpu, err := strconv.ParseFloat(cpuStr, 64); err == nil {
			c.deps.Logger.Debug("Setting CPU metric",
				zap.String("container", containerName),
				zap.Float64("cpu", cpu))
			c.containerCPU.WithLabelValues(containerName, runtime).Set(cpu)
			c.containerStatus.WithLabelValues(containerName, runtime).Set(1) // Running
		} else {
			c.deps.Logger.Error("Failed to parse CPU value",
				zap.String("cpu_str", cpuStr),
				zap.Error(err))
		}

		// Parse memory usage
		used := parseMemoryValue(memUsed)
		limit := parseMemoryValue(memLimit)
		c.deps.Logger.Debug("Setting memory metrics",
			zap.String("container", containerName),
			zap.String("mem_used_str", memUsed),
			zap.Float64("mem_used_bytes", used),
			zap.String("mem_limit_str", memLimit),
			zap.Float64("mem_limit_bytes", limit))
		c.containerMemory.WithLabelValues(containerName, runtime, "used").Set(used)
		c.containerMemory.WithLabelValues(containerName, runtime, "limit").Set(limit)

		// Parse network I/O
		rx := parseNetworkValue(netRx)
		tx := parseNetworkValue(netTx)
		c.deps.Logger.Debug("Setting network I/O metrics",
			zap.String("container", containerName),
			zap.String("net_rx_str", netRx),
			zap.Float64("net_rx_bytes", rx),
			zap.String("net_tx_str", netTx),
			zap.Float64("net_tx_bytes", tx))
		c.containerNetIO.WithLabelValues(containerName, runtime, "rx").Set(rx)
		c.containerNetIO.WithLabelValues(containerName, runtime, "tx").Set(tx)

		// Parse block I/O
		read := parseByteValue(blockRead)
		write := parseByteValue(blockWrite)
		c.deps.Logger.Debug("Setting block I/O metrics",
			zap.String("container", containerName),
			zap.String("block_read_str", blockRead),
			zap.Float64("block_read_bytes", read),
			zap.String("block_write_str", blockWrite),
			zap.Float64("block_write_bytes", write))
		c.containerBlockIO.WithLabelValues(containerName, runtime, "read").Set(read)
		c.containerBlockIO.WithLabelValues(containerName, runtime, "write").Set(write)
	}

	return nil
}

// isContainerIgnored checks if a container should be ignored
func (c *ContainerCollector) isContainerIgnored(containerName string) bool {
	for _, ignored := range c.deps.Config.Containers.IgnoredNames {
		if containerName == ignored {
			return true
		}
	}
	return false
}

// parseMemoryValue converts memory strings like "1.5GiB", "512MiB" to bytes
func parseMemoryValue(memStr string) float64 {
	re := regexp.MustCompile(`^([\d.]+)([KMGT]i?B?)$`)
	matches := re.FindStringSubmatch(memStr)

	if len(matches) != 3 {
		return 0
	}

	value, err := strconv.ParseFloat(matches[1], 64)
	if err != nil {
		return 0
	}

	unit := strings.ToUpper(matches[2])

	switch unit {
	case "B":
		return value
	case "KB", "KIB":
		return value * 1024
	case "MB", "MIB":
		return value * 1024 * 1024
	case "GB", "GIB":
		return value * 1024 * 1024 * 1024
	case "TB", "TIB":
		return value * 1024 * 1024 * 1024 * 1024
	default:
		return 0
	}
}

// parseByteValue converts byte strings like "164MB", "0B" to bytes (for block I/O)
func parseByteValue(byteStr string) float64 {
	re := regexp.MustCompile(`^([\d.]+)([KMGT]?B)$`)
	matches := re.FindStringSubmatch(byteStr)

	if len(matches) != 3 {
		return 0
	}

	value, err := strconv.ParseFloat(matches[1], 64)
	if err != nil {
		return 0
	}

	unit := strings.ToUpper(matches[2])

	switch unit {
	case "B":
		return value
	case "KB":
		return value * 1000
	case "MB":
		return value * 1000 * 1000
	case "GB":
		return value * 1000 * 1000 * 1000
	case "TB":
		return value * 1000 * 1000 * 1000 * 1000
	default:
		return 0
	}
}

// parseNetworkValue converts network I/O strings like "12.9kB", "6.34kB" to bytes
func parseNetworkValue(netStr string) float64 {
	// Handle case-insensitive units for network I/O (Docker uses lowercase)
	re := regexp.MustCompile(`^([\d.]+)([kmgtKMGT]?[bB])$`)
	matches := re.FindStringSubmatch(netStr)

	if len(matches) != 3 {
		return 0
	}

	value, err := strconv.ParseFloat(matches[1], 64)
	if err != nil {
		return 0
	}

	unit := strings.ToUpper(matches[2])
	var result float64

	switch unit {
	case "B":
		result = value
	case "KB":
		result = value * 1000
	case "MB":
		result = value * 1000 * 1000
	case "GB":
		result = value * 1000 * 1000 * 1000
	case "TB":
		result = value * 1000 * 1000 * 1000 * 1000
	default:
		result = 0
	}

	return result
}
