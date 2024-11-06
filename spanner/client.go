package spanner

import (
	"cloud.google.com/go/spanner"
	databasev1 "cloud.google.com/go/spanner/admin/database/apiv1"
	"cloud.google.com/go/spanner/admin/database/apiv1/databasepb"
	"context"
	"github.com/googleapis/gax-go/v2"
	"google.golang.org/api/option"
	"os"
	"time"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -generate

//counterfeiter:generate . Client
type Client interface {
	ReadWriteTransaction(ctx context.Context, f func(context.Context, *spanner.ReadWriteTransaction) error) (commitTimestamp time.Time, err error)
	Single() *spanner.ReadOnlyTransaction
	UpdateDatabaseDdl(ctx context.Context, req *databasepb.UpdateDatabaseDdlRequest, opts ...gax.CallOption) (*databasev1.UpdateDatabaseDdlOperation, error)
	Close() error
}

type client struct {
	client      *spanner.Client
	adminClient *databasev1.DatabaseAdminClient
}

func NewClient(ctx context.Context, config Config) (Client, error) {
	var opts []option.ClientOption
	if config.CredentialsFile != "" && os.Getenv("SPANNER_EMULATOR_HOST") == "" {
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

	return &client{
		client:      spannerClient,
		adminClient: spannerAdminClient,
	}, nil
}

func (c *client) Close() error {
	c.client.Close()
	return c.adminClient.Close()
}

func (c *client) ReadWriteTransaction(ctx context.Context, f func(context.Context, *spanner.ReadWriteTransaction) error) (commitTimestamp time.Time, err error) {
	return c.client.ReadWriteTransaction(ctx, f)
}

func (c *client) Single() *spanner.ReadOnlyTransaction {
	return c.client.Single()
}

func (c *client) UpdateDatabaseDdl(ctx context.Context, req *databasepb.UpdateDatabaseDdlRequest, opts ...gax.CallOption) (*databasev1.UpdateDatabaseDdlOperation, error) {
	return c.adminClient.UpdateDatabaseDdl(ctx, req, opts...)
}
