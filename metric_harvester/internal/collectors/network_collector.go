package collectors

import (
	"context"
	"regexp"
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

// NetworkCollector collects network metrics like interface statistics and ping metrics
type NetworkCollector struct {
	deps *CollectorDependencies

	// Prometheus metrics for network interfaces
	interfaceRxBytes   *prometheus.GaugeVec
	interfaceTxBytes   *prometheus.GaugeVec
	interfaceRxPackets *prometheus.GaugeVec
	interfaceTxPackets *prometheus.GaugeVec
	interfaceRxErrors  *prometheus.GaugeVec
	interfaceTxErrors  *prometheus.GaugeVec
	interfaceRxDropped *prometheus.GaugeVec
	interfaceTxDropped *prometheus.GaugeVec
	interfaceUp        *prometheus.GaugeVec

	// Prometheus metrics for connectivity tests
	pingLatency    *prometheus.GaugeVec
	pingPacketLoss *prometheus.GaugeVec
	pingReachable  *prometheus.GaugeVec
}

// NewNetworkCollector creates a new NetworkCollector
// Args:
// - deps: CollectorDependencies
// Returns:
// - *NetworkCollector: new NetworkCollector instance
func NewNetworkCollector(deps *CollectorDependencies) *NetworkCollector {
	return &NetworkCollector{
		deps: deps,
		interfaceRxBytes: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "network_interface_rx_bytes_total",
				Help: "Total received bytes on network interface",
			},
			[]string{"interface"},
		),
		interfaceTxBytes: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "network_interface_tx_bytes_total",
				Help: "Total transmitted bytes on network interface",
			},
			[]string{"interface"},
		),
		interfaceRxPackets: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "network_interface_rx_packets_total",
				Help: "Total received packets on network interface",
			},
			[]string{"interface"},
		),
		interfaceTxPackets: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "network_interface_tx_packets_total",
				Help: "Total transmitted packets on network interface",
			},
			[]string{"interface"},
		),
		interfaceRxErrors: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "network_interface_rx_errors_total",
				Help: "Total receive errors on network interface",
			},
			[]string{"interface"},
		),
		interfaceTxErrors: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "network_interface_tx_errors_total",
				Help: "Total transmit errors on network interface",
			},
			[]string{"interface"},
		),
		interfaceRxDropped: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "network_interface_rx_dropped_total",
				Help: "Total dropped received packets on network interface",
			},
			[]string{"interface"},
		),
		interfaceTxDropped: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "network_interface_tx_dropped_total",
				Help: "Total dropped transmitted packets on network interface",
			},
			[]string{"interface"},
		),
		interfaceUp: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "network_interface_up",
				Help: "Network interface is up (1) or down (0)",
			},
			[]string{"interface"},
		),
		pingLatency: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "network_ping_latency_milliseconds",
				Help: "Ping latency to target host in milliseconds",
			},
			[]string{"target"},
		),
		pingPacketLoss: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "network_ping_packet_loss_percent",
				Help: "Ping packet loss percentage to target host",
			},
			[]string{"target"},
		),
		pingReachable: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "network_ping_reachable",
				Help: "Target host is reachable via ping (1) or not (0)",
			},
			[]string{"target"},
		),
	}
}

func (c *NetworkCollector) Name() string {
	return "network"
}

func (c *NetworkCollector) Describe(ch chan<- *prometheus.Desc) {
	c.interfaceRxBytes.Describe(ch)
	c.interfaceTxBytes.Describe(ch)
	c.interfaceRxPackets.Describe(ch)
	c.interfaceTxPackets.Describe(ch)
	c.interfaceRxErrors.Describe(ch)
	c.interfaceTxErrors.Describe(ch)
	c.interfaceRxDropped.Describe(ch)
	c.interfaceTxDropped.Describe(ch)
	c.interfaceUp.Describe(ch)
	c.pingLatency.Describe(ch)
	c.pingPacketLoss.Describe(ch)
	c.pingReachable.Describe(ch)
}

func (c *NetworkCollector) Collect(ch chan<- prometheus.Metric) {
	c.interfaceRxBytes.Collect(ch)
	c.interfaceTxBytes.Collect(ch)
	c.interfaceRxPackets.Collect(ch)
	c.interfaceTxPackets.Collect(ch)
	c.interfaceRxErrors.Collect(ch)
	c.interfaceTxErrors.Collect(ch)
	c.interfaceRxDropped.Collect(ch)
	c.interfaceTxDropped.Collect(ch)
	c.interfaceUp.Collect(ch)
	c.pingLatency.Collect(ch)
	c.pingPacketLoss.Collect(ch)
	c.pingReachable.Collect(ch)
}

// CollectMetrics collects network metrics
// This is the main function that collects all the network metrics
// The commands it runs are:
// - cat /proc/net/dev
// - ping -c 3 target
func (c *NetworkCollector) CollectMetrics(ctx context.Context) error {
	c.deps.Logger.Debug("Collecting network metrics")

	// Collect network interface statistics
	if err := c.collectInterfaceMetrics(ctx); err != nil {
		c.deps.Logger.Error("Failed to collect network interface metrics", zap.Error(err))
	}

	// Collect ping metrics for configured targets
	if err := c.collectPingMetrics(ctx); err != nil {
		c.deps.Logger.Error("Failed to collect ping metrics", zap.Error(err))
	}

	return nil
}

// collectInterfaceMetrics collects network interface statistics
// This is the main function that collects all the network interface statistics
// The command it runs is:
// - cat /proc/net/dev
func (c *NetworkCollector) collectInterfaceMetrics(ctx context.Context) error {
	// Get network interface statistics from /proc/net/dev on Linux
	output, err := c.deps.Executor.Execute(ctx, "cat", "/proc/net/dev")
	if err != nil {
		return err
	}

	return c.parseInterfaceStats(string(output))
}

// collectPingMetrics collects ping metrics
// This is the main function that collects all the ping metrics
// The commands it runs are:
// - ping -c 3 target
func (c *NetworkCollector) collectPingMetrics(ctx context.Context) error {
	// Default ping targets - these could be made configurable
	targets := []string{
		"8.8.8.8",    // Google DNS
		"1.1.1.1",    // Cloudflare DNS
		"google.com", // External connectivity test
	}

	// Add configured ping targets if available
	if len(c.deps.Config.Network.PingTargets) > 0 {
		targets = c.deps.Config.Network.PingTargets
	}

	for _, target := range targets {
		if err := c.collectPingMetricsForTarget(ctx, target); err != nil {
			c.deps.Logger.Warn("Failed to ping target",
				zap.String("target", target),
				zap.Error(err))
			// Mark as unreachable
			c.pingReachable.WithLabelValues(target).Set(0)
		}
	}

	return nil
}

// parseInterfaceStats parses network interface statistics
// This is the main function that parses the network interface statistics
// The command it runs is:
// - cat /proc/net/dev
// Parse received stats (first 8 fields)
// Example: "eth0: 1234567 8901 0 0 0 0 0 0 2345678 9012 0 0 0 0 0 0"
// fields[0] is the received bytes
// fields[1] is the received packets
// fields[2] is the received errors
// fields[3] is the received dropped
// fields[8] is the transmitted bytes
// fields[9] is the transmitted packets
// fields[10] is the transmitted errors
// fields[11] is the transmitted dropped
// fields[12] is the received errors
// fields[13] is the received dropped
// fields[14] is the transmitted errors
// fields[15] is the transmitted dropped
func (c *NetworkCollector) parseInterfaceStats(output string) error {
	lines := strings.Split(output, "\n")

	for i, line := range lines {
		// Skip first two header lines
		if i < 2 || strings.TrimSpace(line) == "" {
			continue
		}

		// Parse interface line: "eth0: 1234567 8901 0 0 0 0 0 0 2345678 9012 0 0 0 0 0 0"
		parts := strings.Split(line, ":")
		if len(parts) != 2 {
			continue
		}

		interfaceName := strings.TrimSpace(parts[0])
		statsStr := strings.TrimSpace(parts[1])
		fields := strings.Fields(statsStr)

		if len(fields) < 16 {
			continue
		}

		// Skip loopback interface unless specifically configured
		if interfaceName == "lo" && !c.deps.Config.Network.MonitorLoopback {
			continue
		}

		// Skip ignored interfaces
		if c.isInterfaceIgnored(interfaceName) {
			continue
		}

		if rxBytes, err := strconv.ParseFloat(fields[0], 64); err == nil {
			c.interfaceRxBytes.WithLabelValues(interfaceName).Set(rxBytes)
		}
		if rxPackets, err := strconv.ParseFloat(fields[1], 64); err == nil {
			c.interfaceRxPackets.WithLabelValues(interfaceName).Set(rxPackets)
		}
		if rxErrors, err := strconv.ParseFloat(fields[2], 64); err == nil {
			c.interfaceRxErrors.WithLabelValues(interfaceName).Set(rxErrors)
		}
		if rxDropped, err := strconv.ParseFloat(fields[3], 64); err == nil {
			c.interfaceRxDropped.WithLabelValues(interfaceName).Set(rxDropped)
		}

		// Parse transmitted stats (fields 8-15)
		if txBytes, err := strconv.ParseFloat(fields[8], 64); err == nil {
			c.interfaceTxBytes.WithLabelValues(interfaceName).Set(txBytes)
		}
		if txPackets, err := strconv.ParseFloat(fields[9], 64); err == nil {
			c.interfaceTxPackets.WithLabelValues(interfaceName).Set(txPackets)
		}
		if txErrors, err := strconv.ParseFloat(fields[10], 64); err == nil {
			c.interfaceTxErrors.WithLabelValues(interfaceName).Set(txErrors)
		}
		if txDropped, err := strconv.ParseFloat(fields[11], 64); err == nil {
			c.interfaceTxDropped.WithLabelValues(interfaceName).Set(txDropped)
		}

		// Check if interface is up by checking if it has any activity
		isUp := 0.0
		if rxBytes, _ := strconv.ParseFloat(fields[0], 64); rxBytes > 0 {
			isUp = 1.0
		} else if txBytes, _ := strconv.ParseFloat(fields[8], 64); txBytes > 0 {
			isUp = 1.0
		}
		c.interfaceUp.WithLabelValues(interfaceName).Set(isUp)
	}

	return nil
}

// collectPingMetricsForTarget collects ping metrics for a target
// This is the main function that collects all the ping metrics for a target
// The command it runs is:
// - ping -c 3 target
// Example: "64 bytes from 8.8.8.8: icmp_seq=1 ttl=118 time=12.3 ms"
func (c *NetworkCollector) collectPingMetricsForTarget(ctx context.Context, target string) error {
	output, err := c.deps.Executor.PingHost(ctx, target, 3) // Send 3 pings
	if err != nil {
		return err
	}

	return c.parsePingOutput(string(output), target)
}

// parsePingOutput parses ping output
// This is the main function that parses the ping output
// Example: "64 bytes from 8.8.8.8: icmp_seq=1 ttl=118 time=12.3 ms"
func (c *NetworkCollector) parsePingOutput(output, target string) error {
	lines := strings.Split(output, "\n")

	var latencies []float64
	var packetsSent, packetsReceived int

	// Parse ping output
	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Parse individual ping lines: "64 bytes from 8.8.8.8: icmp_seq=1 ttl=118 time=12.3 ms"
		if strings.Contains(line, "time=") {
			timeRegex := regexp.MustCompile(`time=([\d.]+)\s*ms`)
			matches := timeRegex.FindStringSubmatch(line)
			if len(matches) == 2 {
				if latency, err := strconv.ParseFloat(matches[1], 64); err == nil {
					latencies = append(latencies, latency)
				}
			}
			packetsReceived++
		}

		// Parse summary line: "3 packets transmitted, 3 received, 0% packet loss, time 2002ms"
		if strings.Contains(line, "packets transmitted") {
			summaryRegex := regexp.MustCompile(`(\d+) packets transmitted, (\d+) received`)
			matches := summaryRegex.FindStringSubmatch(line)
			if len(matches) == 3 {
				packetsSent, _ = strconv.Atoi(matches[1])
				packetsReceived, _ = strconv.Atoi(matches[2])
			}
		}
	}

	// Calculate metrics
	if len(latencies) > 0 {
		// Use average latency
		var totalLatency float64
		for _, lat := range latencies {
			totalLatency += lat
		}
		avgLatency := totalLatency / float64(len(latencies))
		c.pingLatency.WithLabelValues(target).Set(avgLatency)

		// Host is reachable
		c.pingReachable.WithLabelValues(target).Set(1)
	} else {
		// No successful pings
		c.pingReachable.WithLabelValues(target).Set(0)
	}

	// Calculate packet loss
	if packetsSent > 0 {
		packetLoss := float64(packetsSent-packetsReceived) / float64(packetsSent) * 100
		c.pingPacketLoss.WithLabelValues(target).Set(packetLoss)
	}

	return nil
}

func (c *NetworkCollector) isInterfaceIgnored(interfaceName string) bool {
	for _, ignored := range c.deps.Config.Network.IgnoredInterfaces {
		if interfaceName == ignored {
			return true
		}
	}
	return false
}
