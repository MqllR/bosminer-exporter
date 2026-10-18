package collector

import (
	"encoding/json"
	"io"
	"log/slog"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus/testutil"

	"github.com/mqllr/bosminer-exporter/internal/cgminer"
)

var replies = map[string]string{
	"version":     `{"STATUS":[{"STATUS":"S","Code":22,"Msg":"BOSminer","Description":"BOSminer+ 0.2.0"}],"VERSION":[{"BOSminer":"0.2.0"}],"id":1}`,
	"summary":     `{"STATUS":[{"STATUS":"S"}],"SUMMARY":[{"MHS av":13500500.0,"Accepted":10,"Rejected":1,"Hardware Errors":2}],"id":1}`,
	"devs":        `{"STATUS":[{"STATUS":"S"}],"DEVS":[{"ID":6,"MHS av":2000000.0}],"id":1}`,
	"pools":       `{"STATUS":[{"STATUS":"S"}],"POOLS":[{"POOL":0,"URL":"stratum+tcp://p:3333","Status":"Alive","Stratum Active":true,"Accepted":5,"Rejected":1,"Stale":0,"Priority":0}],"id":1}`,
	"devdetails":  `{"STATUS":[{"STATUS":"S"}],"DEVDETAILS":[{"ID":6,"Model":"Antminer S9","Chips":63,"Cores":7182,"Frequency":282,"Voltage":7.9}],"id":1}`,
	"fans":        `{"STATUS":[{"STATUS":"S","Code":202}],"FANS":[{"FAN":0,"ID":0,"RPM":3000,"Speed":50}],"id":1}`,
	"temps":       `{"STATUS":[{"STATUS":"S","Code":201}],"TEMPS":[{"ID":6,"Board":60,"Chip":70,"TEMP":0}],"id":1}`,
	"tempctrl":    `{"STATUS":[{"STATUS":"S","Code":200}],"TEMPCTRL":[{"Mode":"Automatic","Target":75,"Hot":90,"Dangerous":95}],"id":1}`,
	"tunerstatus": `{"STATUS":[{"STATUS":"S","Code":203}],"TUNERSTATUS":[{"ApproximateMinerPowerConsumption":1200,"TunerChainStatus":[{"HashchainIndex":6,"ApproximatePowerConsumptionWatt":400,"PowerLimitWatt":450}]}],"id":1}`,
}

func fakeMiner(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { l.Close() })
	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				return
			}
			go func() {
				defer conn.Close()
				buf := make([]byte, 1024)
				n, _ := conn.Read(buf)
				var req struct{ Command string }
				_ = json.Unmarshal(buf[:n], &req)
				_, _ = io.WriteString(conn, replies[req.Command]+"\x00")
			}()
		}
	}()
	return l.Addr().String()
}

func TestCollect(t *testing.T) {
	c := New(&cgminer.Client{Address: fakeMiner(t), Timeout: 2 * time.Second}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	expected := `
# HELP bosminer_up 1 if the last scrape of the miner API succeeded.
# TYPE bosminer_up gauge
bosminer_up 1
# HELP bosminer_pool_up 1 if the pool status is Alive.
# TYPE bosminer_pool_up gauge
bosminer_pool_up{pool="0",url="stratum+tcp://p:3333"} 1
# HELP bosminer_chain_chips Number of chips on the chain (devdetails).
# TYPE bosminer_chain_chips gauge
bosminer_chain_chips{chain="6",model="Antminer S9"} 63
# HELP bosminer_fan_rpm Fan rotation speed in RPM.
# TYPE bosminer_fan_rpm gauge
bosminer_fan_rpm{fan="0"} 3000
# HELP bosminer_board_temperature_celsius Board temperature.
# TYPE bosminer_board_temperature_celsius gauge
bosminer_board_temperature_celsius{chain="6"} 60
# HELP bosminer_power_watts Approximate miner power consumption.
# TYPE bosminer_power_watts gauge
bosminer_power_watts 1200
# HELP bosminer_hashrate_ghs Miner hashrate in GH/s over the given window (summary).
# TYPE bosminer_hashrate_ghs gauge
bosminer_hashrate_ghs{window="av"} 13500.5
`
	if err := testutil.CollectAndCompare(c, strings.NewReader(expected),
		"bosminer_up", "bosminer_pool_up", "bosminer_chain_chips", "bosminer_fan_rpm", "bosminer_board_temperature_celsius", "bosminer_power_watts", "bosminer_hashrate_ghs"); err != nil {
		t.Fatal(err)
	}
}

func TestCollectMinerDown(t *testing.T) {
	c := New(&cgminer.Client{Address: "127.0.0.1:1", Timeout: time.Second}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	expected := `
# HELP bosminer_up 1 if the last scrape of the miner API succeeded.
# TYPE bosminer_up gauge
bosminer_up 0
`
	if err := testutil.CollectAndCompare(c, strings.NewReader(expected), "bosminer_up"); err != nil {
		t.Fatal(err)
	}
}
