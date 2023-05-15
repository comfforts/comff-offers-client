package client

import (
	"time"

	"google.golang.org/grpc"

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

const DeliveryClientContextKey = ContextKey("offers-client")
const DefaultClientName = "comfforts-offers-client"

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
