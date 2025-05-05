package emulators

import (
	"context"
	"fmt"

	"cloud.google.com/go/spanner"
	database "cloud.google.com/go/spanner/admin/database/apiv1"
	"cloud.google.com/go/spanner/admin/database/apiv1/databasepb"
	instance "cloud.google.com/go/spanner/admin/instance/apiv1"
	"cloud.google.com/go/spanner/admin/instance/apiv1/instancepb"

	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
)

type SpannerEmulator struct {
	Pool *dockertest.Pool
}

type SpannerClient interface {
	Single() *spanner.ReadOnlyTransaction
}

func NewSpannerEmulator() (*SpannerEmulator, error) {
	var (
		err  error
		pool *dockertest.Pool
	)
	pool, err = dockertest.NewPool("")
	if err != nil {
		return nil, err
	}

	if err = pool.Client.Ping(); err != nil {
		return nil, err
	}

	return &SpannerEmulator{
		Pool: pool,
	}, nil
}

func (s *SpannerEmulator) Start() (*dockertest.Resource, error) {
	resource, err := s.Pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "gcr.io/cloud-spanner-emulator/emulator",
		Tag:        "latest",
	}, func(config *docker.HostConfig) {
		// remove the container after the test is completed
		config.AutoRemove = true
		// do not auto restart the container we created for the test
		config.RestartPolicy = docker.RestartPolicy{Name: "no"}
	})
	if err != nil {
		return nil, err
	}

	return resource, err
}

func (s *SpannerEmulator) Cleanup(resource *dockertest.Resource) error {
	return s.Pool.Purge(resource)
}

func (s *SpannerEmulator) Wait(ctx context.Context, client SpannerClient) error {
	err := s.Pool.Retry(func() error {
		iter := client.Single().Query(ctx, spanner.Statement{SQL: "select 1;"})
		if iter == nil {
			return fmt.Errorf("could not query spanner")
		}
		return nil
	})
	return err
}

func (s *SpannerEmulator) SetupInstance(ctx context.Context, projectName, instanceID string) (*instancepb.Instance, error) {
	admin, err := instance.NewInstanceAdminClient(ctx)
	if err != nil {
		return nil, err
	}
	defer admin.Close()
	op, err := admin.CreateInstance(ctx, &instancepb.CreateInstanceRequest{
		Parent:     fmt.Sprintf("projects/%s", projectName),
		InstanceId: instanceID,
		Instance: &instancepb.Instance{
			Config:      fmt.Sprintf("project/%s/instanceConfigs/emulator-config", projectName),
			DisplayName: "Test Instance",
			NodeCount:   1,
		},
	})
	if err != nil {
		return nil, err
	}

	i, err := op.Wait(ctx)
	if err != nil {
		return nil, err
	}

	return i, err
}

func (s *SpannerEmulator) SetupDatabase(ctx context.Context, projectName, instanceID, databaseName string) (*databasepb.Database, error) {
	admin, err := database.NewDatabaseAdminClient(ctx)
	if err != nil {
		return nil, err
	}
	defer admin.Close()
	op, err := admin.CreateDatabase(ctx, &databasepb.CreateDatabaseRequest{
		Parent:          fmt.Sprintf("projects/%s/instances/%s", projectName, instanceID),
		CreateStatement: fmt.Sprintf("CREATE DATABASE %s", databaseName),
		DatabaseDialect: databasepb.DatabaseDialect_POSTGRESQL,
	},
	)
	if err != nil {
		return nil, err
	}
	return op.Wait(ctx)
}
