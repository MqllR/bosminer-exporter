// Package collector exposes BOSminer API data as Prometheus metrics.
package collector

import (
	"context"
	"log/slog"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/mqllr/bosminer-exporter/internal/cgminer"
)

const namespace = "bosminer"

// Querier is satisfied by *cgminer.Client.
type Querier interface {
	Query(ctx context.Context, command string) (*cgminer.Response, error)
}

// Collector queries the miner on every scrape.
type Collector struct {
	client Querier
	log    *slog.Logger

	up, scrapeDuration, info                                                *prometheus.Desc
	fanRPM, fanSpeed                                                        *prometheus.Desc
	tempBoard, tempChip                                                     *prometheus.Desc
	tempTarget, tempHot, tempDangerous                                      *prometheus.Desc
	minerPower, chainPower, chainPowerLimit                                 *prometheus.Desc
	hashrate, chainHashrate, accepted, rejected, hwErrors                   *prometheus.Desc
	chainChips, chainCores, chainFrequency, chainVoltage                    *prometheus.Desc
	poolUp, poolActive, poolPriority, poolAccepted, poolRejected, poolStale *prometheus.Desc
	poolDiffAccepted, poolDiffRejected, poolStratumDiff, poolLastShare      *prometheus.Desc
}

func desc(name, help string, labels ...string) *prometheus.Desc {
	return prometheus.NewDesc(prometheus.BuildFQName(namespace, "", name), help, labels, nil)
}

// New returns a Collector using the given client.
func New(client Querier, log *slog.Logger) *Collector {
	return &Collector{
		client:           client,
		log:              log,
		up:               desc("up", "1 if the last scrape of the miner API succeeded."),
		scrapeDuration:   desc("scrape_duration_seconds", "Duration of the miner API scrape."),
		info:             desc("info", "BOSminer version information.", "version"),
		fanRPM:           desc("fan_rpm", "Fan rotation speed in RPM.", "fan"),
		fanSpeed:         desc("fan_speed_percent", "Fan speed in percent.", "fan"),
		tempBoard:        desc("board_temperature_celsius", "Board temperature.", "chain"),
		tempChip:         desc("chip_temperature_celsius", "Chip temperature.", "chain"),
		tempTarget:       desc("temp_target_celsius", "Temperature control target."),
		tempHot:          desc("temp_hot_celsius", "Temperature control hot threshold."),
		tempDangerous:    desc("temp_dangerous_celsius", "Temperature control dangerous threshold."),
		minerPower:       desc("power_watts", "Approximate miner power consumption."),
		chainPower:       desc("chain_power_watts", "Approximate chain power consumption.", "chain"),
		chainPowerLimit:  desc("chain_power_limit_watts", "Chain power limit.", "chain"),
		hashrate:         desc("hashrate_ghs", "Miner hashrate in GH/s over the given window (summary).", "window"),
		chainHashrate:    desc("chain_hashrate_ghs", "Chain hashrate in GH/s over the given window (devs).", "chain", "window"),
		chainChips:       desc("chain_chips", "Number of chips on the chain (devdetails).", "chain", "model"),
		chainCores:       desc("chain_cores", "Number of cores on the chain (devdetails).", "chain", "model"),
		chainFrequency:   desc("chain_frequency_mhz", "Chain chip frequency in MHz (devdetails).", "chain", "model"),
		chainVoltage:     desc("chain_voltage_volts", "Chain voltage (devdetails).", "chain", "model"),
		poolUp:           desc("pool_up", "1 if the pool status is Alive.", "pool", "url"),
		poolActive:       desc("pool_stratum_active", "1 if the stratum connection of the pool is active.", "pool", "url"),
		poolPriority:     desc("pool_priority", "Pool priority (0 is highest).", "pool", "url"),
		poolAccepted:     desc("pool_shares_accepted_total", "Shares accepted by the pool.", "pool", "url"),
		poolRejected:     desc("pool_shares_rejected_total", "Shares rejected by the pool.", "pool", "url"),
		poolStale:        desc("pool_shares_stale_total", "Stale shares for the pool.", "pool", "url"),
		poolDiffAccepted: desc("pool_difficulty_accepted_total", "Accepted share difficulty sum.", "pool", "url"),
		poolDiffRejected: desc("pool_difficulty_rejected_total", "Rejected share difficulty sum.", "pool", "url"),
		poolStratumDiff:  desc("pool_stratum_difficulty", "Current stratum difficulty.", "pool", "url"),
		poolLastShare:    desc("pool_last_share_timestamp_seconds", "Unix time of the last share submitted to the pool.", "pool", "url"),
		accepted:         desc("shares_accepted_total", "Accepted shares (summary)."),
		rejected:         desc("shares_rejected_total", "Rejected shares (summary)."),
		hwErrors:         desc("hardware_errors_total", "Hardware errors (summary)."),
	}
}

// Describe implements prometheus.Collector.
func (c *Collector) Describe(ch chan<- *prometheus.Desc) {
	for _, d := range []*prometheus.Desc{
		c.up, c.scrapeDuration, c.info, c.fanRPM, c.fanSpeed, c.tempBoard, c.tempChip,
		c.tempTarget, c.tempHot, c.tempDangerous, c.minerPower, c.chainPower,
		c.chainPowerLimit, c.chainChips, c.chainCores, c.chainFrequency, c.chainVoltage,
		c.poolUp, c.poolActive, c.poolPriority, c.poolAccepted, c.poolRejected, c.poolStale,
		c.poolDiffAccepted, c.poolDiffRejected, c.poolStratumDiff, c.poolLastShare,
		c.hashrate, c.chainHashrate, c.accepted, c.rejected, c.hwErrors,
	} {
		ch <- d
	}
}

// Collect implements prometheus.Collector.
func (c *Collector) Collect(ch chan<- prometheus.Metric) {
	start := time.Now()
	ctx := context.Background()
	ok := 1.0

	// Each command is independent: a failure in one must not hide the others.
	for name, fn := range map[string]func(context.Context, chan<- prometheus.Metric) error{
		"version":     c.version,
		"summary":     c.summary,
		"fans":        c.fans,
		"temps":       c.temps,
		"devdetails":  c.devdetails,
		"pools":       c.pools,
		"devs":        c.devs,
		"tempctrl":    c.tempctrl,
		"tunerstatus": c.tunerstatus,
	} {
		if err := fn(ctx, ch); err != nil {
			c.log.Warn("command failed", "command", name, "err", err)
			ok = 0
		}
	}

	ch <- prometheus.MustNewConstMetric(c.up, prometheus.GaugeValue, ok)
	ch <- prometheus.MustNewConstMetric(c.scrapeDuration, prometheus.GaugeValue, time.Since(start).Seconds())
}

func (c *Collector) gauge(ch chan<- prometheus.Metric, d *prometheus.Desc, row map[string]any, field string, labels ...string) {
	if v, ok := cgminer.Float(row, field); ok {
		ch <- prometheus.MustNewConstMetric(d, prometheus.GaugeValue, v, labels...)
	}
}

func (c *Collector) version(ctx context.Context, ch chan<- prometheus.Metric) error {
	resp, err := c.client.Query(ctx, "version")
	if err != nil {
		return err
	}
	version := ""
	if len(resp.Status) > 0 {
		version = resp.Status[0].Description
	}
	if rows := resp.Data["VERSION"]; len(rows) > 0 {
		if v, ok := cgminer.String(rows[0], "BOSminer"); ok {
			version = v
		}
	}
	ch <- prometheus.MustNewConstMetric(c.info, prometheus.GaugeValue, 1, version)
	return nil
}

var hashrateWindows = []string{"5s", "1m", "5m", "15m", "24h", "av"}

func (c *Collector) devs(ctx context.Context, ch chan<- prometheus.Metric) error {
	resp, err := c.client.Query(ctx, "devs")
	if err != nil {
		return err
	}
	for i, r := range resp.Data["DEVS"] {
		id := idOf(r, "ID", i)
		for _, w := range hashrateWindows {
			if v, ok := cgminer.Float(r, "MHS "+w); ok {
				ch <- prometheus.MustNewConstMetric(c.chainHashrate, prometheus.GaugeValue, v/1000, id, w)
			}
		}
	}
	return nil
}

func (c *Collector) summary(ctx context.Context, ch chan<- prometheus.Metric) error {
	resp, err := c.client.Query(ctx, "summary")
	if err != nil {
		return err
	}
	rows := resp.Data["SUMMARY"]
	if len(rows) == 0 {
		return nil
	}
	r := rows[0]
	for _, w := range hashrateWindows {
		if v, ok := cgminer.Float(r, "MHS "+w); ok {
			ch <- prometheus.MustNewConstMetric(c.hashrate, prometheus.GaugeValue, v/1000, w)
		}
	}
	for field, d := range map[string]*prometheus.Desc{
		"Accepted": c.accepted, "Rejected": c.rejected, "Hardware Errors": c.hwErrors,
	} {
		if v, ok := cgminer.Float(r, field); ok {
			ch <- prometheus.MustNewConstMetric(d, prometheus.CounterValue, v)
		}
	}
	return nil
}

func (c *Collector) fans(ctx context.Context, ch chan<- prometheus.Metric) error {
	resp, err := c.client.Query(ctx, "fans")
	if err != nil {
		return err
	}
	for i, r := range resp.Data["FANS"] {
		id := idOf(r, "ID", i)
		c.gauge(ch, c.fanRPM, r, "RPM", id)
		c.gauge(ch, c.fanSpeed, r, "Speed", id)
	}
	return nil
}

func (c *Collector) temps(ctx context.Context, ch chan<- prometheus.Metric) error {
	resp, err := c.client.Query(ctx, "temps")
	if err != nil {
		return err
	}
	for i, r := range resp.Data["TEMPS"] {
		id := idOf(r, "ID", i)
		c.gauge(ch, c.tempBoard, r, "Board", id)
		c.gauge(ch, c.tempChip, r, "Chip", id)
	}
	return nil
}

func (c *Collector) tempctrl(ctx context.Context, ch chan<- prometheus.Metric) error {
	resp, err := c.client.Query(ctx, "tempctrl")
	if err != nil {
		return err
	}
	if rows := resp.Data["TEMPCTRL"]; len(rows) > 0 {
		c.gauge(ch, c.tempTarget, rows[0], "Target")
		c.gauge(ch, c.tempHot, rows[0], "Hot")
		c.gauge(ch, c.tempDangerous, rows[0], "Dangerous")
	}
	return nil
}

func (c *Collector) tunerstatus(ctx context.Context, ch chan<- prometheus.Metric) error {
	resp, err := c.client.Query(ctx, "tunerstatus")
	if err != nil {
		return err
	}
	rows := resp.Data["TUNERSTATUS"]
	if len(rows) == 0 {
		return nil
	}
	c.gauge(ch, c.minerPower, rows[0], "ApproximateMinerPowerConsumption")
	chains, _ := rows[0]["TunerChainStatus"].([]any)
	for i, item := range chains {
		r, ok := item.(map[string]any)
		if !ok {
			continue
		}
		id := idOf(r, "HashchainIndex", i)
		c.gauge(ch, c.chainPower, r, "ApproximatePowerConsumptionWatt", id)
		c.gauge(ch, c.chainPowerLimit, r, "PowerLimitWatt", id)
	}
	return nil
}

// idOf returns the numeric field as a label value, falling back to the index.
func idOf(row map[string]any, field string, fallback int) string {
	if v, ok := cgminer.Float(row, field); ok {
		return strconv.Itoa(int(v))
	}
	return strconv.Itoa(fallback)
}

func (c *Collector) devdetails(ctx context.Context, ch chan<- prometheus.Metric) error {
	resp, err := c.client.Query(ctx, "devdetails")
	if err != nil {
		return err
	}
	for i, r := range resp.Data["DEVDETAILS"] {
		id := idOf(r, "ID", i)
		model, _ := cgminer.String(r, "Model")
		c.gauge(ch, c.chainChips, r, "Chips", id, model)
		c.gauge(ch, c.chainCores, r, "Cores", id, model)
		c.gauge(ch, c.chainFrequency, r, "Frequency", id, model)
		c.gauge(ch, c.chainVoltage, r, "Voltage", id, model)
	}
	return nil
}

func (c *Collector) pools(ctx context.Context, ch chan<- prometheus.Metric) error {
	resp, err := c.client.Query(ctx, "pools")
	if err != nil {
		return err
	}
	for i, r := range resp.Data["POOLS"] {
		id := idOf(r, "POOL", i)
		url, _ := cgminer.String(r, "URL")
		if status, ok := cgminer.String(r, "Status"); ok {
			up := 0.0
			if status == "Alive" {
				up = 1
			}
			ch <- prometheus.MustNewConstMetric(c.poolUp, prometheus.GaugeValue, up, id, url)
		}
		if active, ok := r["Stratum Active"].(bool); ok {
			v := 0.0
			if active {
				v = 1
			}
			ch <- prometheus.MustNewConstMetric(c.poolActive, prometheus.GaugeValue, v, id, url)
		}
		c.gauge(ch, c.poolPriority, r, "Priority", id, url)
		c.gauge(ch, c.poolStratumDiff, r, "Stratum Difficulty", id, url)
		c.gauge(ch, c.poolLastShare, r, "Last Share Time", id, url)
		for field, d := range map[string]*prometheus.Desc{
			"Accepted": c.poolAccepted, "Rejected": c.poolRejected, "Stale": c.poolStale,
			"Difficulty Accepted": c.poolDiffAccepted, "Difficulty Rejected": c.poolDiffRejected,
		} {
			if v, ok := cgminer.Float(r, field); ok {
				ch <- prometheus.MustNewConstMetric(d, prometheus.CounterValue, v, id, url)
			}
		}
	}
	return nil
}
