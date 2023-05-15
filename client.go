package client

import (
	"context"
	"fmt"
	"os"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"

	config "github.com/comfforts/comff-config"
	"github.com/comfforts/logger"

	api "github.com/comfforts/comff-offers/api/v1"
)

const DEFAULT_SERVICE_PORT = "57051"
const DEFAULT_SERVICE_HOST = "127.0.0.1"

type ContextKey string

func (c ContextKey) String() string {
	return string(c)
}

var (
	defaultDialTimeout      = 5 * time.Second
	defaultKeepAlive        = 30 * time.Second
	defaultKeepAliveTimeout = 10 * time.Second
)

const OffersClientContextKey = ContextKey("offers-client")
const DefaultClientName = "comfforts-offers-client"

type Client interface {
	GetOfferStatuses(ctx context.Context, req *api.OfferStatusesRequest, opts ...grpc.CallOption) (*api.OfferStatusesResponse, error)
	GetOfferTypes(ctx context.Context, req *api.OfferTypesRequest, opts ...grpc.CallOption) (*api.OfferTypesResponse, error)
	CreateOffer(ctx context.Context, req *api.CreateOfferRequest, opts ...grpc.CallOption) (*api.OfferResponse, error)
	UpdateOffer(ctx context.Context, req *api.UpdateOfferRequest, opts ...grpc.CallOption) (*api.OfferResponse, error)
	GetOffer(ctx context.Context, req *api.GetOfferRequest, opts ...grpc.CallOption) (*api.OfferResponse, error)
	GetOffers(ctx context.Context, req *api.GetOffersRequest, opts ...grpc.CallOption) (*api.OffersResponse, error)
	DeleteOffer(ctx context.Context, req *api.DeleteOfferRequest, opts ...grpc.CallOption) (*api.DeleteResponse, error)
	Close() error
}

type ClientOption struct {
	DialTimeout      time.Duration
	KeepAlive        time.Duration
	KeepAliveTimeout time.Duration
	Caller           string
}

func NewDefaultClientOption() *ClientOption {
	return &ClientOption{
		DialTimeout:      defaultDialTimeout,
		KeepAlive:        defaultKeepAlive,
		KeepAliveTimeout: defaultKeepAliveTimeout,
	}
}

type offersClient struct {
	logger logger.AppLogger
	client api.OffersClient
	conn   *grpc.ClientConn
	opts   *ClientOption
}

func NewClient(
	logger logger.AppLogger,
	clientOpts *ClientOption,
) (*offersClient, error) {
	if clientOpts.Caller == "" {
		clientOpts.Caller = DefaultClientName
	}

	tlsConfig, err := config.SetupTLSConfig(&config.ConfigOpts{
		Target: config.OFFERS_CLIENT,
	})
	if err != nil {
		logger.Error("error setting offers service client TLS", zap.Error(err))
		return nil, err
	}
	tlsCreds := credentials.NewTLS(tlsConfig)
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(tlsCreds),
	}

	servicePort := os.Getenv("OFFERS_SERVICE_PORT")
	if servicePort == "" {
		servicePort = DEFAULT_SERVICE_PORT
	}
	serviceHost := os.Getenv("OFFERS_SERVICE_HOST")
	if serviceHost == "" {
		serviceHost = DEFAULT_SERVICE_HOST
	}
	serviceAddr := fmt.Sprintf("%s:%s", serviceHost, servicePort)
	// with load balancer
	// serviceAddr = fmt.Sprintf("%s:///%s", loadbalance.ShopResolverName, serviceAddr)
	// serviceAddr = fmt.Sprintf("%s:///%s", "shops", serviceAddr)

	conn, err := grpc.Dial(serviceAddr, opts...)
	if err != nil {
		logger.Error("offers client failed to connect", zap.Error(err))
		return nil, err
	}

	client := api.NewOffersClient(conn)
	logger.Info("offers client connected", zap.String("host", serviceHost), zap.String("port", servicePort))
	return &offersClient{
		client: client,
		logger: logger,
		conn:   conn,
		opts:   clientOpts,
	}, nil
}

func (ofc *offersClient) GetOfferStatuses(
	ctx context.Context,
	req *api.OfferStatusesRequest,
	opts ...grpc.CallOption,
) (*api.OfferStatusesResponse, error) {
	ctx, cancel := ofc.contextWithOptions(ctx, ofc.opts)
	defer cancel()

	return ofc.client.GetOfferStatuses(ctx, req)
}

func (ofc *offersClient) GetOfferTypes(
	ctx context.Context,
	req *api.OfferTypesRequest,
	opts ...grpc.CallOption,
) (*api.OfferTypesResponse, error) {
	ctx, cancel := ofc.contextWithOptions(ctx, ofc.opts)
	defer cancel()

	return ofc.client.GetOfferTypes(ctx, req)
}

func (ofc *offersClient) CreateOffer(
	ctx context.Context,
	req *api.CreateOfferRequest,
	opts ...grpc.CallOption,
) (*api.OfferResponse, error) {
	ctx, cancel := ofc.contextWithOptions(ctx, ofc.opts)
	defer cancel()

	return ofc.client.CreateOffer(ctx, req)
}

func (ofc *offersClient) UpdateOffer(
	ctx context.Context,
	req *api.UpdateOfferRequest,
	opts ...grpc.CallOption,
) (*api.OfferResponse, error) {
	ctx, cancel := ofc.contextWithOptions(ctx, ofc.opts)
	defer cancel()

	return ofc.client.UpdateOffer(ctx, req)
}

func (ofc *offersClient) GetOffer(
	ctx context.Context,
	req *api.GetOfferRequest,
	opts ...grpc.CallOption,
) (*api.OfferResponse, error) {
	ctx, cancel := ofc.contextWithOptions(ctx, ofc.opts)
	defer cancel()

	return ofc.client.GetOffer(ctx, req)
}

func (ofc *offersClient) GetOffers(
	ctx context.Context,
	req *api.GetOffersRequest,
	opts ...grpc.CallOption,
) (*api.OffersResponse, error) {
	ctx, cancel := ofc.contextWithOptions(ctx, ofc.opts)
	defer cancel()

	return ofc.client.GetOffers(ctx, req)
}

func (ofc *offersClient) DeleteOffer(
	ctx context.Context,
	req *api.DeleteOfferRequest,
	opts ...grpc.CallOption,
) (*api.DeleteResponse, error) {
	ctx, cancel := ofc.contextWithOptions(ctx, ofc.opts)
	defer cancel()

	return ofc.client.DeleteOffer(ctx, req)
}

func (ofc *offersClient) Close() error {
	if err := ofc.conn.Close(); err != nil {
		ofc.logger.Error("error closing offers client connection", zap.Error(err))
		return err
	}
	return nil
}

func (ofc *offersClient) contextWithOptions(ctx context.Context, opts *ClientOption) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(ctx, ofc.opts.DialTimeout)
	if ofc.opts.Caller != "" {
		md := metadata.New(map[string]string{"service-client": ofc.opts.Caller})
		ctx = metadata.NewOutgoingContext(ctx, md)
	}

	return ctx, cancel
}
