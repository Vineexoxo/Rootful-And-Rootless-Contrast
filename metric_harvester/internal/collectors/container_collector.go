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
// Example: "CONTAINER ID   NAME               CPU %     MEM USAGE / LIMIT   MEM %     NET I/O           BLOCK I/O         PIDS"
func (c *ContainerCollector) parseContainerStats(output, runtime string) error {
	lines := strings.Split(output, "\n")

	for i, line := range lines {
		if i == 0 || strings.TrimSpace(line) == "" {
			continue // Skip header and empty lines
		}

		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}

		containerName := fields[0]

		// Parse CPU usage (e.g., "15.30%")
		if cpuStr := strings.TrimSuffix(fields[1], "%"); cpuStr != fields[1] {
			if cpu, err := strconv.ParseFloat(cpuStr, 64); err == nil {
				c.containerCPU.WithLabelValues(containerName, runtime).Set(cpu)
				c.containerStatus.WithLabelValues(containerName, runtime).Set(1) // Running
			}
		}

		// Parse memory usage (e.g., "1.5GiB / 8GiB")
		if len(fields) >= 3 {
			memParts := strings.Split(fields[2], " / ")
			if len(memParts) == 2 {
				used := parseMemoryValue(strings.TrimSpace(memParts[0]))
				limit := parseMemoryValue(strings.TrimSpace(memParts[1]))

				if used > 0 {
					c.containerMemory.WithLabelValues(containerName, runtime, "used").Set(used)
				}
				if limit > 0 {
					c.containerMemory.WithLabelValues(containerName, runtime, "limit").Set(limit)
				}
			}
		}

		// Parse network I/O (e.g., "1.2MB / 850kB")
		if len(fields) >= 4 {
			netParts := strings.Split(fields[3], " / ")
			if len(netParts) == 2 {
				rx := parseByteValue(strings.TrimSpace(netParts[0]))
				tx := parseByteValue(strings.TrimSpace(netParts[1]))

				if rx > 0 {
					c.containerNetIO.WithLabelValues(containerName, runtime, "rx").Set(rx)
				}
				if tx > 0 {
					c.containerNetIO.WithLabelValues(containerName, runtime, "tx").Set(tx)
				}
			}
		}

		// Parse block I/O (e.g., "0B / 0B")
		if len(fields) >= 5 {
			blockParts := strings.Split(fields[4], " / ")
			if len(blockParts) == 2 {
				read := parseByteValue(strings.TrimSpace(blockParts[0]))
				write := parseByteValue(strings.TrimSpace(blockParts[1]))

				if read > 0 {
					c.containerBlockIO.WithLabelValues(containerName, runtime, "read").Set(read)
				}
				if write > 0 {
					c.containerBlockIO.WithLabelValues(containerName, runtime, "write").Set(write)
				}
			}
		}
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

// parseByteValue converts byte strings like "1.2MB", "850kB" to bytes
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
