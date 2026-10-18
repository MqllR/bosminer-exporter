package main

import (
	"flag"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/mqllr/bosminer-exporter/internal/cgminer"
	"github.com/mqllr/bosminer-exporter/internal/collector"
)

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func main() {
	listen := flag.String("web.listen-address", envOr("BOSMINER_EXPORTER_LISTEN_ADDRESS", ":9928"), "Address to expose metrics on.")
	miner := flag.String("miner.address", envOr("BOSMINER_EXPORTER_MINER_ADDRESS", "localhost:4028"), "host:port of the BOSminer API.")
	timeoutStr := flag.String("miner.timeout", envOr("BOSMINER_EXPORTER_MINER_TIMEOUT", "5s"), "Timeout per miner API command.")
	flag.Parse()

	log := slog.New(slog.NewTextHandler(os.Stderr, nil))
	timeout, err := time.ParseDuration(*timeoutStr)
	if err != nil {
		log.Error("invalid --miner.timeout", "err", err)
		os.Exit(2)
	}

	reg := prometheus.NewRegistry()
	reg.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		collector.New(&cgminer.Client{Address: *miner, Timeout: timeout}, log),
	)

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("ok\n")) })
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(`<html><body><h1>BOSminer exporter</h1><a href="/metrics">Metrics</a></body></html>`))
	})

	log.Info("starting", "listen", *listen, "miner", *miner)
	srv := &http.Server{Addr: *listen, Handler: mux, ReadHeaderTimeout: 10 * time.Second}
	if err := srv.ListenAndServe(); err != nil {
		log.Error("server failed", "err", err)
		os.Exit(1)
	}
}
