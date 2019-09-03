package remiro

import (
	"context"
	"time"

	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
)

var (
	// reqLatencyMs records the time it took for request to be served
	reqLatencyMs = stats.Float64("request/latency", "Request serving latency", "ms")

	// redisCmdCount records the count of any request towards backing redis server
	redisCmdCount = stats.Int64("cmd/count", "Redis request count", "requests")

	// keyTarget tag the backing Redis target in a request
	keyTarget, _ = tag.NewKey("target")

	// keyCommand tag the command sent to a backing Redis
	keyCommand, _ = tag.NewKey("command")

	// cmdCountView provides view for Redis command count
	cmdCountView = &view.View{
		Name:        "command/count",
		Measure:     redisCmdCount,
		Description: "The count of outbound request to Redis instances",
		Aggregation: view.Count(),
		TagKeys:     []tag.Key{keyTarget, keyCommand},
	}

	// reqLatencyView provides view for request latency
	reqLatencyView = &view.View{
		Name:        "request/latency",
		Measure:     reqLatencyMs,
		Description: "The latency distribution of requests",

		// Latency in buckets:
		// [>=0ms, >=10ms, >=25ms, >=50ms, >=75ms, >=100ms, >=150ms >=200ms]
		Aggregation: view.Distribution(0, 10, 25, 50, 75, 100, 150, 200),
	}

	views = []*view.View{cmdCountView, reqLatencyView}
)

func sinceInMs(startTime time.Time) float64 {
	return float64(time.Since(startTime).Nanoseconds()) / 1e6
}

func recordRedisCmd(target, command string) {
	ctx, _ := tag.New(context.Background(), tag.Insert(keyTarget, target), tag.Insert(keyCommand, command))
	stats.Record(ctx, redisCmdCount.M(1))
}
