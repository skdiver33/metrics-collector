package grpcserver

import (
	"context"
	"log"
	"net"

	pb "github.com/skdiver33/metrics-collector/internal/proto"
	"github.com/skdiver33/metrics-collector/internal/server"
	"github.com/skdiver33/metrics-collector/internal/store"
	"github.com/skdiver33/metrics-collector/models"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type MetricsServer struct {
	Config  *server.ServerConfig
	Storage store.StorageInterface
	grpcSrv *grpc.Server
	pb.UnimplementedMetricsServer
}

func NewMetricsServer(conf *server.ServerConfig, store store.StorageInterface) *MetricsServer {
	return &MetricsServer{Config: conf, Storage: store}
}

func (ms *MetricsServer) ConvertMetrics(pm []*pb.Metric) *[]models.Metrics {

	bunch := make([]models.Metrics, 0)
	for _, m := range pm {
		newMetr := models.Metrics{}
		newMetr.ID = m.GetId()
		switch m.GetType() {
		case pb.Metric_GAUGE:
			newMetr.MType = models.Gauge
			val := m.GetValue()
			newMetr.Value = &val
		case pb.Metric_COUNTER:
			newMetr.MType = models.Counter
			delta := m.GetDelta()
			newMetr.Delta = &delta
		}
		bunch = append(bunch, newMetr)
	}
	return &bunch
}

func (ms MetricsServer) UpdateMetrics(ctx context.Context, in *pb.UpdateMetricsRequest) (*pb.UpdateMetricsResponse, error) {
	log.Println("rpcserver call update metrics")

	m := in.GetMetrics()
	newMetrics := ms.ConvertMetrics(m)
	err := ms.Storage.UpdateAllMetrics(ctx, newMetrics)
	if err != nil {
		log.Println("grpc server error update bunch metrics")
	}
	return &pb.UpdateMetricsResponse{}, nil
}

func (ms *MetricsServer) NetworkTrustInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {

	var ip string
	if ms.Config.TrustedSubnet != "" {
		_, ipNet, err := net.ParseCIDR(ms.Config.TrustedSubnet)
		if err != nil {
			return nil, err
		}

		md, ok := metadata.FromIncomingContext(ctx)
		if ok {
			values := md.Get("x-real-ip")
			if len(values) > 0 {
				// ключ содержит слайс строк, получаем первую строку
				ip = values[0]

				if !ipNet.Contains(net.ParseIP(ip)) {
					log.Printf("agent ip not in trusted network. Request forrbiden.")
					return &pb.UpdateMetricsResponse{}, status.Error(codes.PermissionDenied, "ip in not trusted network")
				}
			}

		}
	}
	// Call the actual RPC method
	resp, err := handler(ctx, req)

	return resp, err
}

func (ms *MetricsServer) Run() error {
	listener, err := net.Listen("tcp", ms.Config.GRPCListenAddress)
	if err != nil {
		log.Println("error open listener for grpc server")
		return err
	}

	s := grpc.NewServer(grpc.UnaryInterceptor(ms.NetworkTrustInterceptor))
	ms.grpcSrv = s
	pb.RegisterMetricsServer(s, ms)

	log.Println("Running grpc server")

	if err := s.Serve(listener); err != nil {
		log.Println("error run grpc server ")
		return err
	}
	return nil

}

func (ms *MetricsServer) Stop() {
	log.Println("grpc server stoped")
	ms.grpcSrv.GracefulStop()
}
