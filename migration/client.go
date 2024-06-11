package migration

import (
	"cloud.google.com/go/spanner"
	databasev1 "cloud.google.com/go/spanner/admin/database/apiv1"
	"cloud.google.com/go/spanner/admin/database/apiv1/databasepb"
	"context"
	"github.com/googleapis/gax-go/v2"
	"google.golang.org/api/option"
	"time"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -generate

//counterfeiter:generate . SpannerClient
type SpannerClient interface {
	ReadWriteTransaction(ctx context.Context, f func(context.Context, *spanner.ReadWriteTransaction) error) (commitTimestamp time.Time, err error)
	Single() *spanner.ReadOnlyTransaction
	UpdateDatabaseDdl(ctx context.Context, req *databasepb.UpdateDatabaseDdlRequest, opts ...gax.CallOption) (*databasev1.UpdateDatabaseDdlOperation, error)
}

type Client struct {
	client      *spanner.Client
	adminClient *databasev1.DatabaseAdminClient
}

func NewClient(ctx context.Context, config Config) (*Client, error) {
	var opts []option.ClientOption
	if config.CredentialsFile != "" {
		opts = append(opts, option.WithCredentialsFile(config.CredentialsFile))
	}

	spannerClient, err := spanner.NewClient(ctx, config.URL(), opts...)
	if err != nil {
		return nil, err
	}

	spannerAdminClient, err := databasev1.NewDatabaseAdminClient(ctx, opts...)
	if err != nil {
		spannerClient.Close()
		return nil, err
	}

	return &Client{
		client:      spannerClient,
		adminClient: spannerAdminClient,
	}, nil
}

func (c *Client) Close() {
	c.client.Close()
	c.adminClient.Close()
}

func (c *Client) ReadWriteTransaction(ctx context.Context, f func(context.Context, *spanner.ReadWriteTransaction) error) (commitTimestamp time.Time, err error) {
	return c.client.ReadWriteTransaction(ctx, f)
}

func (c *Client) Single() *spanner.ReadOnlyTransaction {
	return c.client.Single()
}

func (c *Client) UpdateDatabaseDdl(ctx context.Context, req *databasepb.UpdateDatabaseDdlRequest, opts ...gax.CallOption) (*databasev1.UpdateDatabaseDdlOperation, error) {
	return c.adminClient.UpdateDatabaseDdl(ctx, req, opts...)
}
