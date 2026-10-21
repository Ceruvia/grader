package metrics

import (
	"context"
	"fmt"
	"time"

	"github.com/Ceruvia/grader/internal/config"
	"github.com/Ceruvia/grader/internal/pool"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
	log "github.com/sirupsen/logrus"
)

func RunMetricsPusher(cfg *config.ServerConfig) {
	url := cfg.MonitoringCfg.InfluxURL
	token := cfg.MonitoringCfg.InfluxToken
	org := cfg.MonitoringCfg.InfluxOrganization
	bucket := cfg.MonitoringCfg.InfluxBucket

	// Create the InfluxDB client and blocking write API
	client := influxdb2.NewClient(url, token)
	writeAPI := client.WriteAPIBlocking(org, bucket)
	defer client.Close()

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		// Collect metrics
		idle := float64(pool.Pool.IdleCount())
		busy := float64(pool.Pool.BusyCount())

		var cpuPct float64
		if pcts, err := cpu.Percent(0, false); err == nil && len(pcts) > 0 {
			cpuPct = pcts[0]
		}

		var memPct float64
		if vm, err := mem.VirtualMemory(); err == nil {
			memPct = vm.UsedPercent
		}

		// Build the point
		p := influxdb2.NewPoint(
			"grader_metrics", // measurement name
			map[string]string{ // tags
				"grader_name":        cfg.GraderName,
				"grader_environment": cfg.GraderEnv,
			},
			map[string]interface{}{ // fields
				"idle_workers":       idle,
				"busy_workers":       busy,
				"system_cpu_percent": cpuPct,
				"system_mem_percent": memPct,
			},
			time.Now(),
		)

		if err := writeAPI.WritePoint(context.Background(), p); err != nil {
			log.WithError(err).Error("failed to write metrics to InfluxDB")
		} else {
			fmt.Printf("pushed metrics:idle=%.0f;busy=%.0f;cpu=%.1f%%;mem=%.1f%%\n",
				idle, busy, cpuPct, memPct)
		}
	}
}
