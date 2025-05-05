package integration

import (
	"context"
	"fmt"
	"os"
	"testing"

	gspanner "cloud.google.com/go/spanner"
	database "cloud.google.com/go/spanner/admin/database/apiv1"
	"cloud.google.com/go/spanner/admin/database/apiv1/databasepb"
	instance "cloud.google.com/go/spanner/admin/instance/apiv1"
	"cloud.google.com/go/spanner/admin/instance/apiv1/instancepb"

	"github.com/dora-network/dora-service-utils/spanner"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	redisv9 "github.com/redis/go-redis/v9"
	"github.com/twmb/franz-go/pkg/kadm"
	"github.com/twmb/franz-go/pkg/kgo"
)

type DoraNetwork struct {
	Pool               *dockertest.Pool
	Network            *docker.Network
	KafkaResource      *dockertest.Resource
	RedisResource      *dockertest.Resource
	SpannerResource    *dockertest.Resource
	APIGatewayResource *dockertest.Resource
}

func NewDoraNetwork(t *testing.T) (*DoraNetwork, error) {
	t.Helper()

	pool, err := dockertest.NewPool("")
	if err != nil {
		return nil, err
	}

	if err := pool.Client.Ping(); err != nil {
		return nil, err
	}

	network, err := pool.Client.CreateNetwork(docker.CreateNetworkOptions{
		Name: "dora-network-testing",
	})
	if err != nil {
		return nil, err
	}

	return &DoraNetwork{
		Pool:    pool,
		Network: network,
	}, nil
}

func (dora *DoraNetwork) CreateKafkaResource(t *testing.T, ctx context.Context) error {
	t.Helper()
	resource, err := dora.Pool.RunWithOptions(
		&dockertest.RunOptions{
			Name:       "kafka",
			Repository: "confluentinc/cp-kafka",
			Tag:        "latest",
			NetworkID:  dora.Network.ID,
			Hostname:   "kafka",
			Env: []string{
				"KAFKA_NODE_ID=1",
				"KAFKA_LISTENER_SECURITY_PROTOCOL_MAP=LISTENER_CONTROLLER:PLAINTEXT,LISTENER_DOCKER_INTERNAL:PLAINTEXT,LISTENER_DOCKER_EXTERNAL:PLAINTEXT",
				"KAFKA_ADVERTISED_LISTENERS=LISTENER_DOCKER_INTERNAL://kafka:29092,LISTENER_DOCKER_EXTERNAL://localhost:9092",
				"KAFKA_PROCESS_ROLES=broker,controller",
				"KAFKA_CONTROLLER_QUORUM_VOTERS=1@kafka:29093",
				"KAFKA_LISTENERS=LISTENER_CONTROLLER://kafka:29093,LISTENER_DOCKER_INTERNAL://kafka:29092,LISTENER_DOCKER_EXTERNAL://0.0.0.0:9092",
				"KAFKA_CONTROLLER_LISTENER_NAMES=LISTENER_CONTROLLER",
				"KAFKA_INTER_BROKER_LISTENER_NAME=LISTENER_DOCKER_INTERNAL",
				"KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR=1",
				"KAFKA_GROUP_INITIAL_REBALANCE_DELAY_MS=0",
				"KAFKA_TRANSACTION_STATE_LOG_MIN_ISR=1",
				"KAFKA_TRANSACTION_STATE_LOG_REPLICATION_FACTOR=1",
				"KAFKA_LOG_DIRS=/tmp/kraft-combined-logs",
				"CLUSTER_ID=dAtOC6X6SyiTN3BxRtMHbw",
			},
			PortBindings: map[docker.Port][]docker.PortBinding{
				"9092/tcp": {{HostIP: "localhost", HostPort: "9092/tcp"}},
			},
			ExposedPorts: []string{"9092/tcp"},
		}, func(config *docker.HostConfig) {
			config.AutoRemove = true
			config.RestartPolicy = docker.RestartPolicy{Name: "no"}
		},
	)
	if err != nil {
		return err
	}

	hostAndPort := fmt.Sprintf("127.0.0.1:%s", resource.GetPort("9092/tcp"))
	t.Log("Kafka host and port: ", hostAndPort)

	retryFunc := func() error {
		client, err := kgo.NewClient(
			kgo.SeedBrokers(hostAndPort),
			kgo.AllowAutoTopicCreation(),
		)
		if err != nil {
			return err
		}
		defer client.Close()

		adminClient := kadm.NewClient(client)
		_, err = adminClient.CreateTopic(ctx, 1, 1, nil, "test-topic")
		if err != nil {
			t.Logf("could not create topic: %s", err)
		}
		return err
	}

	if err = dora.Pool.Retry(retryFunc); err != nil {
		return fmt.Errorf("could not start kafka: %w", err)
	}

	dora.KafkaResource = resource
	return nil
}

func (dora *DoraNetwork) CreateRedisResource(t *testing.T, ctx context.Context) error {
	t.Helper()
	resource, err := dora.Pool.RunWithOptions(
		&dockertest.RunOptions{
			Name:       "redis",
			Repository: "redis",
			Tag:        "latest",
			NetworkID:  dora.Network.ID,
			Hostname:   "redis",
			PortBindings: map[docker.Port][]docker.PortBinding{
				"6379/tcp": {{HostIP: "localhost", HostPort: "6379/tcp"}},
			},
			ExposedPorts: []string{"6379/tcp"},
		}, func(config *docker.HostConfig) {
			config.AutoRemove = true
			config.RestartPolicy = docker.RestartPolicy{Name: "no"}
		},
	)
	if err != nil {
		return err
	}

	hostAndPort := resource.GetHostPort("6379/tcp")

	if err = dora.Pool.Retry(func() error {
		db := redisv9.NewClient(&redisv9.Options{
			Addr: hostAndPort,
		})

		return db.Ping(ctx).Err()
	}); err != nil {
		return fmt.Errorf("could not start redis: %w", err)
	}

	dora.RedisResource = resource
	return nil
}

func (dora *DoraNetwork) CreateSpannerResource(t *testing.T, ctx context.Context, config spanner.Config) error {
	t.Helper()
	resource, err := dora.Pool.RunWithOptions(
		&dockertest.RunOptions{
			Name:       "spanner",
			Repository: "gcr.io/cloud-spanner-emulator/emulator",
			Tag:        "latest",
			NetworkID:  dora.Network.ID,
			Hostname:   "spanner",
		}, func(config *docker.HostConfig) {
			config.AutoRemove = true
			config.RestartPolicy = docker.RestartPolicy{Name: "no"}
		},
	)
	if err != nil {
		return err
	}

	hostAndPort := resource.GetHostPort("9010/tcp")
	t.Logf("Setting SPANNER_EMULATOR_HOST: %s", hostAndPort)
	if err = os.Setenv("SPANNER_EMULATOR_HOST", hostAndPort); err != nil {
		return err
	}

	client, err := spanner.NewClient(ctx, config)
	if err != nil {
		return err
	}

	if err = dora.Pool.Retry(func() error {
		iter := client.Single().Query(ctx, gspanner.Statement{SQL: "select 1;"})
		if iter == nil {
			return fmt.Errorf("could not query spanner")
		}
		return nil
	}); err != nil {
		return fmt.Errorf("could not start spanner: %w", err)
	}

	dora.SpannerResource = resource
	return nil
}

func (dora *DoraNetwork) CreateAPIGatewayResource(t *testing.T, ctx context.Context) error {
	// TODO: This should start the API Gateway service using the Docker image we create, but we're not there yet so for now, just return nil
	return nil
}

func (dora *DoraNetwork) Cleanup() error {
	if dora.KafkaResource != nil {
		if err := dora.Pool.Purge(dora.KafkaResource); err != nil {
			return err
		}
	}

	if dora.RedisResource != nil {
		if err := dora.Pool.Purge(dora.RedisResource); err != nil {
			return err
		}
	}

	if dora.SpannerResource != nil {
		if err := dora.Pool.Purge(dora.SpannerResource); err != nil {
			return err
		}
	}

	if err := dora.Pool.Client.RemoveNetwork(dora.Network.ID); err != nil {
		return err
	}

	if dora.APIGatewayResource != nil {
		if err := dora.Pool.Purge(dora.APIGatewayResource); err != nil {
			return err
		}
	}

	return nil
}

func (dora *DoraNetwork) GetKafkaClient() (*kgo.Client, error) {
	return kgo.NewClient(
		kgo.SeedBrokers(dora.KafkaResource.GetHostPort("9092/tcp")),
	)
}

func (dora *DoraNetwork) GetRedisClient() (*redisv9.Client, error) {
	return redisv9.NewClient(&redisv9.Options{
		Addr: dora.RedisResource.GetHostPort("6379/tcp"),
	}), nil
}

func (dora *DoraNetwork) GetSpannerClient(ctx context.Context, config spanner.Config) (spanner.Client, error) {
	return spanner.NewClient(ctx, config)
}

func (dora *DoraNetwork) SetupInstance(ctx context.Context, config spanner.Config) error {
	admin, err := instance.NewInstanceAdminClient(ctx)
	if err != nil {
		return err
	}
	defer admin.Close()

	op, err := admin.CreateInstance(ctx, &instancepb.CreateInstanceRequest{
		Parent:     fmt.Sprintf("projects/%s", config.ProjectID),
		InstanceId: config.InstanceID,
		Instance: &instancepb.Instance{
			Config:      fmt.Sprintf("project/%s/instanceConfigs/emulator-config", config.ProjectID),
			DisplayName: "test",
			NodeCount:   1,
		},
	})
	if err != nil {
		return err
	}

	_, err = op.Wait(ctx)
	return err
}

func (dora *DoraNetwork) SetupDatabase(ctx context.Context, config spanner.Config) error {
	admin, err := database.NewDatabaseAdminClient(ctx)
	if err != nil {
		return err
	}
	defer admin.Close()

	op, err := admin.CreateDatabase(ctx, &databasepb.CreateDatabaseRequest{
		Parent:          fmt.Sprintf("projects/%s/instances/%s", config.ProjectID, config.InstanceID),
		CreateStatement: fmt.Sprintf("CREATE DATABASE %s", config.DatabaseID),
		DatabaseDialect: databasepb.DatabaseDialect_POSTGRESQL,
	})
	if err != nil {
		return err
	}

	_, err = op.Wait(ctx)
	return err
}
