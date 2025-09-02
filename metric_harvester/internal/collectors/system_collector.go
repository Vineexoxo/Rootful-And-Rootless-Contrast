package collectors

import (
	"context"
	"regexp"
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

// SystemCollector collects system metrics like CPU, memory, disk, and uptime
type SystemCollector struct {
	deps *CollectorDependencies

	// Prometheus metrics
	// cpuUsage: system CPU usage percentage
	// memoryUsage: system memory usage in bytes
	// diskUsage: system disk usage in bytes
	// systemUptime: system uptime in seconds. Can be used to calculate system age in days.
	cpuUsage     *prometheus.GaugeVec
	memoryUsage  *prometheus.GaugeVec
	diskUsage    *prometheus.GaugeVec
	systemUptime prometheus.Gauge
}

// NewSystemCollector creates a new SystemCollector
// Args:
// - deps: CollectorDependencies
// Returns:
// - *SystemCollector: new SystemCollector instance
func NewSystemCollector(deps *CollectorDependencies) *SystemCollector {
	return &SystemCollector{
		deps: deps,
		cpuUsage: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "system_cpu_usage_percent",
				Help: "System CPU usage percentage",
			},
			[]string{"type"}, // user, system, idle
		),
		memoryUsage: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "system_memory_usage_bytes",
				Help: "System memory usage in bytes",
			},
			[]string{"type"}, // total, used, free, cached
		),
		diskUsage: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "system_disk_usage_bytes",
				Help: "System disk usage in bytes",
			},
			[]string{"device", "type"}, // used, available, total
		),
		systemUptime: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "system_uptime_seconds",
				Help: "System uptime in seconds",
			},
		),
	}
}

func (c *SystemCollector) Name() string {
	return "system"
}

// Following are methods that need to be implemented for the Prometheus server to know which metrics are being collected
// which are implicitly implemented by the prometheus.systemCollector interface

// Describe implements the prometheus.systemCollector interface
// Needs to be implemented for the Prometheus server to know which metrics are being collected
func (c *SystemCollector) Describe(ch chan<- *prometheus.Desc) {
	c.cpuUsage.Describe(ch)
	c.memoryUsage.Describe(ch)
	c.diskUsage.Describe(ch)
	c.systemUptime.Describe(ch)
}

// Collect implements the prometheus.systemCollector interface
// It sends the collected metrics to the Prometheus server
func (c *SystemCollector) Collect(ch chan<- prometheus.Metric) {
	c.cpuUsage.Collect(ch)
	c.memoryUsage.Collect(ch)
	c.diskUsage.Collect(ch)
	c.systemUptime.Collect(ch)
}

// CollectMetrics collects system metrics
// This is the main function that collects all the system metrics
func (c *SystemCollector) CollectMetrics(ctx context.Context) error {
	c.deps.Logger.Debug("Collecting system metrics")

	// Collect CPU metrics
	if err := c.collectCPUMetrics(ctx); err != nil {
		c.deps.Logger.Error("Failed to collect CPU metrics", zap.Error(err))
	}

	// Collect memory metrics
	if err := c.collectMemoryMetrics(ctx); err != nil {
		c.deps.Logger.Error("Failed to collect memory metrics", zap.Error(err))
	}

	// Collect disk metrics
	if err := c.collectDiskMetrics(ctx); err != nil {
		c.deps.Logger.Error("Failed to collect disk metrics", zap.Error(err))
	}

	// Collect uptime
	if err := c.collectUptimeMetrics(ctx); err != nil {
		c.deps.Logger.Error("Failed to collect uptime metrics", zap.Error(err))
	}

	return nil
}

// collectCPUMetrics collects CPU metrics
// This is the main function that collects all the CPU metrics
// The commands it runs are:
// - top -l 1 -n 0
func (c *SystemCollector) collectCPUMetrics(ctx context.Context) error {
	output, err := c.deps.Executor.GetCPUUsage(ctx)
	if err != nil {
		return err
	}

	// Parse top -bn1 output for Linux
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "%Cpu(s):") {
			// Linux format: "%Cpu(s):  3.2 us,  1.1 sy,  0.0 ni, 95.6 id,  0.0 wa,  0.0 hi,  0.1 si,  0.0 st"
			re := regexp.MustCompile(`(\d+\.?\d*)\s+(\w+)`)
			matches := re.FindAllStringSubmatch(line, -1)

			for _, match := range matches {
				if len(match) == 3 {
					value, err := strconv.ParseFloat(match[1], 64)
					if err == nil {
						switch match[2] {
						case "us":
							c.cpuUsage.WithLabelValues("user").Set(value)
						case "sy":
							c.cpuUsage.WithLabelValues("system").Set(value)
						case "id":
							c.cpuUsage.WithLabelValues("idle").Set(value)
						}
					}
				}
			}
			break
		}
	}

	return nil
}

// collectMemoryMetrics collects memory metrics
// This is the main function that collects all the memory metrics
// The command it runs is:
// - vm_stat
func (c *SystemCollector) collectMemoryMetrics(ctx context.Context) error {
	output, err := c.deps.Executor.GetMemoryUsage(ctx)
	if err != nil {
		return err
	}

	// Parse free -b output for Linux
	lines := strings.Split(string(output), "\n")
	for i, line := range lines {
		if i == 0 || strings.TrimSpace(line) == "" {
			continue // Skip header and empty lines
		}

		fields := strings.Fields(line)
		if len(fields) >= 7 && i == 1 { // Memory line (skip header)
			// Format: "Mem: 16384000 8192000 4096000 4096000 4096000 12288000"
			if total, err := strconv.ParseFloat(fields[1], 64); err == nil {
				c.memoryUsage.WithLabelValues("total").Set(total)
			}
			if used, err := strconv.ParseFloat(fields[2], 64); err == nil {
				c.memoryUsage.WithLabelValues("used").Set(used)
			}
			if free, err := strconv.ParseFloat(fields[3], 64); err == nil {
				c.memoryUsage.WithLabelValues("free").Set(free)
			}
			if available, err := strconv.ParseFloat(fields[6], 64); err == nil {
				c.memoryUsage.WithLabelValues("available").Set(available)
			}
		}
	}

	return nil
}

// collectDiskMetrics collects disk metrics
// This is the main function that collects all the disk metrics
// The command it runs is:
// - df -h /
func (c *SystemCollector) collectDiskMetrics(ctx context.Context) error {
	output, err := c.deps.Executor.GetDiskUsage(ctx, "/")
	if err != nil {
		return err
	}

	// Parse df output
	lines := strings.Split(string(output), "\n")
	for i, line := range lines {
		if i == 0 || strings.TrimSpace(line) == "" {
			continue // Skip header and empty lines
		}

		fields := strings.Fields(line)
		if len(fields) >= 6 {
			device := fields[0]

			// Convert sizes from KB to bytes (df typically shows 1K blocks)
			if total, err := strconv.ParseFloat(fields[1], 64); err == nil {
				c.diskUsage.WithLabelValues(device, "total").Set(total * 1024)
			}
			if used, err := strconv.ParseFloat(fields[2], 64); err == nil {
				c.diskUsage.WithLabelValues(device, "used").Set(used * 1024)
			}
			if available, err := strconv.ParseFloat(fields[3], 64); err == nil {
				c.diskUsage.WithLabelValues(device, "available").Set(available * 1024)
			}
		}
	}

	return nil
}

// collectUptimeMetrics collects uptime metrics
// This is the main function that collects all the uptime metrics
// The command it runs is:
// - uptime
func (c *SystemCollector) collectUptimeMetrics(ctx context.Context) error {
	output, err := c.deps.Executor.GetSystemUptime(ctx)
	if err != nil {
		return err
	}

	// Parse uptime output
	uptimeStr := string(output)

	// Extract uptime in seconds from uptime command output
	// Example: "up 2 days, 10:30" or "up 10:30"
	re := regexp.MustCompile(`up\s+(?:(\d+)\s+days?,\s+)?(\d+):(\d+)`)
	if matches := re.FindStringSubmatch(uptimeStr); len(matches) >= 4 {
		days, _ := strconv.ParseFloat(matches[1], 64)
		hours, _ := strconv.ParseFloat(matches[2], 64)
		minutes, _ := strconv.ParseFloat(matches[3], 64)

		totalSeconds := days*24*3600 + hours*3600 + minutes*60
		c.systemUptime.Set(totalSeconds)
	}

	return nil
}
