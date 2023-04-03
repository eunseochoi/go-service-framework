package manager

import (
	"context"
	"fmt"
	"google.golang.org/grpc"
	"time"
)

func (m *Manager) metricsInterceptor(
	ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (interface{}, error) {
	start := time.Now()
	ret, err := handler(ctx, req)
	m.Metrics().Gauge(fmt.Sprintf("grpc.%s.time", info.FullMethod), float64(time.Since(start)), []string{}, 1.0)

	return ret, err
}
