package main

import (
	"context"
	"fmt"
	"github.com/aptible/mega-collector/api"
	"github.com/aptible/mega-collector/batch"
	"github.com/aptible/mega-collector/batcher"
	"github.com/aptible/mega-collector/emitter"
	"github.com/aptible/mega-collector/emitter/text"
	"github.com/aptible/mini-collector/tls"
	grpcLogrus "github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/reflection"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

const (
	port = ":8000"
)

var (
	requiredTags = []string{
		"environment",
		"service",
		"container",
		"host",
	}

	optionalTags = []string{
		"app",
		"database",
	}

	logger = logrus.WithFields(logrus.Fields{
		"source": "server",
	})
)

type server struct {
	batcher batcher.Batcher
}

func (s *server) Publish(ctx context.Context, line *api.PublishRequest) (*api.PublishResponse, error) {
	md, ok := metadata.FromIncomingContext(ctx)

	if !ok {
		return nil, fmt.Errorf("no metadata")
	}

	ts := time.Unix(line.UnixTime, 0)

	tags := map[string]string{}

	for _, k := range requiredTags {
		v, ok := md[k]
		if !ok {
			return nil, fmt.Errorf("missing required metadata key: %s", k)
		}
		tags[k] = v[0]
	}

	for _, k := range optionalTags {
		v, ok := md[k]
		if !ok {
			continue
		}
		tags[k] = v[0]
	}

	err := s.batcher.Ingest(ctx, &batch.Entry{
		Time:           ts,
		Tags:           tags,
		PublishRequest: *line,
	})

	if err != nil {
		logger.Warnf("Ingest failed: %v", err)
	}

	return &api.PublishResponse{}, nil
}


func getEmitter() (emitter.Emitter, func(), error) {
	_, ok := os.LookupEnv("AGGREGATOR_TEXT_CONFIGURATION")
	if ok {
		logger.Infof("using text emitter")
		em := text.Open()
		return em, em.Close, nil
	}

	return nil, nil, fmt.Errorf("no emitter configured")
}

func getBatcher(em emitter.Emitter) (batcher.Batcher, error) {
	minPublishFrequencyText, ok := os.LookupEnv("AGGREGATOR_MINIMUM_PUBLISH_FREQUENCY")
	if !ok {
		minPublishFrequencyText = "15s"
	}

	minPublishFrequency, err := time.ParseDuration(minPublishFrequencyText)
	if err != nil {
		return nil, fmt.Errorf("invalid minimum publish frequency (%s): %v", minPublishFrequencyText, err)
	}

	maxBatchSizeText, ok := os.LookupEnv("AGGREGATOR_MAX_BATCH_SIZE")
	if !ok {
		maxBatchSizeText = "1000"
	}

	maxBatchSize, err := strconv.Atoi(maxBatchSizeText)
	if err != nil {
		return nil, fmt.Errorf("invalid max batch size (%s): %v", maxBatchSizeText, err)
	}

	logger.Infof("minPublishFrequency: %v", minPublishFrequency)

	// TODO: Make batchsize configurable?
	return batcher.New(em, minPublishFrequency, maxBatchSize), nil

}

func main() {
	grpcLogrus.ReplaceGrpcLogger(logger)

	emitter, closeCallback, err := getEmitter()
	if err != nil {
		logger.Fatalf("getEmitterStack failed: %v", err)
	}
	defer closeCallback()

	batcher, err := getBatcher(emitter)
	if err != nil {
		logger.Fatalf("getBatcher failed: %v", err)
	}
	defer batcher.Close()

	lis, err := net.Listen("tcp", port)
	if err != nil {
		logger.Fatalf("failed to listen: %v", err)
	}
	logger.Infof("listening on: %s", port)

	var srv *grpc.Server

	_, enableTls := os.LookupEnv("AGGREGATOR_TLS")
	if enableTls {
		tlsConfig, err := tls.GetTlsConfig("AGGREGATOR")
		if err != nil {
			logger.Fatalf("failed to load tlsConfig: %v", err)
		}

		logger.Info("tls is enabled")
		srv = grpc.NewServer(grpc.Creds(credentials.NewTLS(tlsConfig)))
	} else {
		logger.Warn("tls is disabled")
		srv = grpc.NewServer()
	}

	api.RegisterAggregatorServer(srv, &server{
		batcher: batcher,
	})

	// Register reflection service on gRPC server.
	reflection.Register(srv)

	termChan := make(chan os.Signal, 1)
	signal.Notify(termChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		termSig := <-termChan
		logger.Infof("received %s, shutting down", termSig)
		srv.GracefulStop()
	}()

	if err := srv.Serve(lis); err != nil {
		logger.Fatalf("failed to serve: %v", err)
	}

	logger.Infof("server shutdown")
}
