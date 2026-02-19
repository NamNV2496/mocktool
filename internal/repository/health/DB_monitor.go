package health

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

func StartMongoMonitor(
	client *mongo.Client,
	latencyThreshold time.Duration,
) {
	go func() {
		const windowSize = 5
		var latencies []time.Duration
		for {
			start := time.Now()
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			err := client.Ping(ctx, readpref.Primary())
			cancel()
			latency := time.Since(start)
			if err != nil {
				SetOverloaded(true)
				time.Sleep(5 * time.Second)
				continue
			}
			// Sliding window latency
			latencies = append(latencies, latency)
			if len(latencies) < windowSize {
				continue
			}
			var avg time.Duration
			for _, l := range latencies {
				avg += l
			}
			avg /= time.Duration(len(latencies))
			if avg > latencyThreshold {
				SetLatencyOverloaded(true, int64(avg.Milliseconds()))
			} else if avg < latencyThreshold {
				SetOverloaded(false)
			}
			time.Sleep(5 * time.Second)
		}
	}()
}
