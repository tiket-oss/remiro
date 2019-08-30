package remiro

import (
	"context"
	"time"

	"go.opencensus.io/stats"
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
)

func sinceInMs(startTime time.Time) float64 {
	return float64(time.Since(startTime).Nanoseconds()) / 1e6
}

func recordRedisCmd(target, command string) {
	ctx, _ := tag.New(context.Background(), tag.Insert(keyTarget, target), tag.Insert(keyCommand, command))
	stats.Record(ctx, redisCmdCount.M(1))
}
