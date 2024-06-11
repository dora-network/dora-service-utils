package migration_test

import (
	"cloud.google.com/go/spanner"
	database "cloud.google.com/go/spanner/admin/database/apiv1"
	"cloud.google.com/go/spanner/admin/database/apiv1/databasepb"
	instance "cloud.google.com/go/spanner/admin/instance/apiv1"
	"cloud.google.com/go/spanner/admin/instance/apiv1/instancepb"
	"context"
	"embed"
	"fmt"
	"github.com/dora-network/dora-service-utils/migration"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"log"
	"os"
	"testing"
	"time"
)

//go:embed test_migrations/*.sql
var migrationsFS embed.FS

const (
	projectName  = "doranetwork"
	instanceID   = "doratest"
	databaseName = "assets"
)

var pool *dockertest.Pool

func TestMain(m *testing.M) {
	var err error
	pool, err = dockertest.NewPool("")
	if err != nil {
		log.Fatalf("could not create docker pool: %+v", err)
		os.Exit(1)
	}

	if err = pool.Client.Ping(); err != nil {
		log.Fatalf("could not connect to docker: %+v", err)
		os.Exit(1)
	}

	code := m.Run()
	os.Exit(code)
}

func Test_Migrate(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Hour)
	defer cancel()
	resource, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "gcr.io/cloud-spanner-emulator/emulator",
		Tag:        "latest",
	}, func(config *docker.HostConfig) {
		// remove the container after the test is completed
		config.AutoRemove = true
		// do not auto restart the container we created for the test
		config.RestartPolicy = docker.RestartPolicy{Name: "no"}
	})
	require.NoError(t, err, "couldn't start cloud spanner emulator")

	defer t.Cleanup(func() {
		require.NoError(t, pool.Purge(resource), "could not purge spanner emulator")
	})

	// now that we've started the emulator container we need to get the host and port
	// and set the SPANNER_EMULATOR_HOST env var so that we can use it with the API
	// as specified in https://cloud.google.com/spanner/docs/emulator.
	// This should allow the client library to connect to the appropriate cloud spanner
	// instance we have created for the test
	hostAndPort := resource.GetHostPort("9010/tcp")
	t.Logf("Setting SPANNER_EMULATOR_HOST: %s", hostAndPort)
	os.Setenv("SPANNER_EMULATOR_HOST", hostAndPort)

	config := migration.Config{
		ProjectID:  projectName,
		InstanceID: instanceID,
		DatabaseID: databaseName,
	}
	client, err := migration.NewClient(ctx, config)
	require.NoError(t, err, "could not create migration spanner client")
	err = waitForSpanner(t, ctx, client)
	require.NoError(t, err, "could not connect to spanner emulator")

	setup(t, ctx)
	err = migration.Migrate(context.Background(), migrationsFS, config, client)
	require.NoError(t, err)
	got, err := migration.GetCurrentVersion(ctx, client, migration.SchemaVersionTable)
	require.NoError(t, err)
	var want int64 = 2
	assert.Equal(t, want, got)

	iter := client.Single().Query(ctx, spanner.Statement{
		SQL:    "select * from information_schema.tables where  table_schema = 'public' and table_type = 'BASE TABLE';",
		Params: map[string]interface{}{"p1": migration.SchemaVersionTable},
	})

	defer iter.Stop()

	gotTables := make([]string, 0)
	err = iter.Do(func(r *spanner.Row) error {
		var tableName string
		if err := r.ColumnByName("table_name", &tableName); err != nil {
			return err
		}
		gotTables = append(gotTables, tableName)
		return nil
	})
	wantTables := []string{migration.SchemaVersionTable, "users", "addresses"}
	assert.ElementsMatch(t, wantTables, gotTables)
}

func waitForSpanner(t *testing.T, ctx context.Context, client migration.SpannerClient) error {
	t.Helper()
	err := pool.Retry(func() error {
		iter := client.Single().Query(ctx, spanner.Statement{SQL: "select 1;"})
		if iter == nil {
			return fmt.Errorf("could not query spanner")
		}
		return nil
	})
	return err
}

func TestEnsureMigrationTable(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	resource, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "gcr.io/cloud-spanner-emulator/emulator",
		Tag:        "latest",
	}, func(config *docker.HostConfig) {
		// remove the container after the test is completed
		config.AutoRemove = true
		// do not auto restart the container we created for the test
		config.RestartPolicy = docker.RestartPolicy{Name: "no"}
	})
	require.NoError(t, err, "couldn't start cloud spanner emulator")

	defer t.Cleanup(func() {
		require.NoError(t, pool.Purge(resource), "could not purge spanner emulator")
	})

	// now that we've started the emulator container we need to get the host and port
	// and set the SPANNER_EMULATOR_HOST env var so that we can use it with the API
	// as specified in https://cloud.google.com/spanner/docs/emulator.
	// This should allow the client library to connect to the appropriate cloud spanner
	// instance we have created for the test
	hostAndPort := resource.GetHostPort("9010/tcp")
	t.Logf("Setting SPANNER_EMULATOR_HOST: %s", hostAndPort)
	os.Setenv("SPANNER_EMULATOR_HOST", hostAndPort)

	config := migration.Config{
		ProjectID:  projectName,
		InstanceID: instanceID,
		DatabaseID: databaseName,
	}
	client, err := migration.NewClient(ctx, config)
	require.NoError(t, err, "could not create migration spanner client")
	err = waitForSpanner(t, ctx, client)
	require.NoError(t, err, "could not connect to spanner emulator")

	setup(t, ctx)

	require.NoError(t, migration.EnsureMigrationTable(ctx, config, client, migration.SchemaVersionTable))

	iter := client.Single().Query(ctx, spanner.Statement{
		SQL:    "select * from information_schema.tables where table_name = $1;",
		Params: map[string]interface{}{"p1": migration.SchemaVersionTable},
	})
	defer iter.Stop()
	tableFound := false
	err = iter.Do(func(row *spanner.Row) error {
		var tableName string
		if err := row.ColumnByName("table_name", &tableName); err != nil {
			return err
		}
		t.Logf("table name: %s", tableName)
		if tableName == migration.SchemaVersionTable {
			tableFound = true
		}
		return nil
	})
	require.NoError(t, err)
	assert.True(t, tableFound, "schema migrations table not found, but should have been created")
}

func TestGetCurrentVersion(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	resource, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "gcr.io/cloud-spanner-emulator/emulator",
		Tag:        "latest",
	}, func(config *docker.HostConfig) {
		// remove the container after the test is completed
		config.AutoRemove = true
		// do not auto restart the container we created for the test
		config.RestartPolicy = docker.RestartPolicy{Name: "no"}
	})
	require.NoError(t, err, "couldn't start cloud spanner emulator")

	defer t.Cleanup(func() {
		require.NoError(t, pool.Purge(resource), "could not purge spanner emulator")
	})

	// now that we've started the emulator container we need to get the host and port
	// and set the SPANNER_EMULATOR_HOST env var so that we can use it with the API
	// as specified in https://cloud.google.com/spanner/docs/emulator.
	// This should allow the client library to connect to the appropriate cloud spanner
	// instance we have created for the test
	hostAndPort := resource.GetHostPort("9010/tcp")
	t.Logf("Setting SPANNER_EMULATOR_HOST: %s", hostAndPort)
	os.Setenv("SPANNER_EMULATOR_HOST", hostAndPort)

	config := migration.Config{
		ProjectID:  projectName,
		InstanceID: instanceID,
		DatabaseID: databaseName,
	}
	client, err := migration.NewClient(ctx, config)
	require.NoError(t, err, "could not create migration spanner client")
	err = waitForSpanner(t, ctx, client)
	require.NoError(t, err, "could not connect to spanner emulator")

	setup(t, ctx)

	err = migration.EnsureMigrationTable(ctx, config, client, migration.SchemaVersionTable)
	require.NoError(t, err, "could not create schema migrations table")
	var want int64 = 20
	_, err = client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		stmt := spanner.Statement{
			SQL: fmt.Sprintf(`INSERT INTO %s (version) VALUES
								(18),
								(19),
                                ($1)`, migration.SchemaVersionTable),
			Params: map[string]interface{}{"p1": want},
		}
		rowCount, err := txn.Update(ctx, stmt)
		if err != nil {
			return err
		}
		require.Equal(t, int64(3), rowCount, "expected 1 row to be updated")
		return err
	})
	require.NoError(t, err)
	got, err := migration.GetCurrentVersion(ctx, client, migration.SchemaVersionTable)
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestSetCurrentVersion(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	resource, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "gcr.io/cloud-spanner-emulator/emulator",
		Tag:        "latest",
	}, func(config *docker.HostConfig) {
		// remove the container after the test is completed
		config.AutoRemove = true
		// do not auto restart the container we created for the test
		config.RestartPolicy = docker.RestartPolicy{Name: "no"}
	})
	require.NoError(t, err, "couldn't start cloud spanner emulator")

	defer t.Cleanup(func() {
		require.NoError(t, pool.Purge(resource), "could not purge spanner emulator")
	})

	// now that we've started the emulator container we need to get the host and port
	// and set the SPANNER_EMULATOR_HOST env var so that we can use it with the API
	// as specified in https://cloud.google.com/spanner/docs/emulator.
	// This should allow the client library to connect to the appropriate cloud spanner
	// instance we have created for the test
	hostAndPort := resource.GetHostPort("9010/tcp")
	t.Logf("Setting SPANNER_EMULATOR_HOST: %s", hostAndPort)
	os.Setenv("SPANNER_EMULATOR_HOST", hostAndPort)

	config := migration.Config{
		ProjectID:  projectName,
		InstanceID: instanceID,
		DatabaseID: databaseName,
	}
	client, err := migration.NewClient(ctx, config)
	require.NoError(t, err, "could not create migration spanner client")
	err = waitForSpanner(t, ctx, client)
	require.NoError(t, err, "could not connect to spanner emulator")

	setup(t, ctx)

	err = migration.EnsureMigrationTable(ctx, config, client, migration.SchemaVersionTable)
	require.NoError(t, err, "could not create schema migrations table")
	var want int64 = 21
	_, err = client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		stmt := spanner.Statement{
			SQL: fmt.Sprintf(`INSERT INTO %s (version) VALUES
								(18),
								(19),
                                (20)`, migration.SchemaVersionTable),
		}
		rowCount, err := txn.Update(ctx, stmt)
		if err != nil {
			return err
		}
		require.Equal(t, int64(3), rowCount, "expected 1 row to be updated")
		return err
	})
	require.NoError(t, err)
	err = migration.SetCurrentVersion(ctx, client, migration.SchemaVersionTable, want)
	require.NoError(t, err)

	iter := client.Single().Query(ctx, spanner.Statement{
		SQL: "select version from public.schema_migrations order by version desc;",
	})

	versions := make([]int64, 0)
	err = iter.Do(func(r *spanner.Row) error {
		var version int64
		if err := r.ColumnByName("version", &version); err != nil {
			return err
		}
		versions = append(versions, version)
		return nil
	})
	require.NoError(t, err)
	assert.Equal(t, want, versions[0])
	assert.ElementsMatch(t, []int64{want, 20, 19, 18}, versions)
}

func setup(t *testing.T, ctx context.Context) {
	// first we need to create an instance on the emulator
	setupInstance(t, ctx)
	// now that we have an instance created, we should create the database and apply migrations
	setupDatabase(t, ctx)
}

func setupInstance(t *testing.T, ctx context.Context) {
	t.Helper()
	admin, err := instance.NewInstanceAdminClient(ctx)
	require.NoError(t, err, "creating instance admin client")
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
	require.NoError(t, err, "creating test spanner instance")

	i, err := op.Wait(ctx)
	require.NoError(t, err, "waiting for instance creation to complete")
	require.Equal(t, instancepb.Instance_READY, i.State, "instance not ready after wait")
}

func setupDatabase(t *testing.T, ctx context.Context) {
	t.Helper()
	admin, err := database.NewDatabaseAdminClient(ctx)
	require.NoError(t, err, "creating database admin client")
	defer admin.Close()
	op, err := admin.CreateDatabase(ctx, &databasepb.CreateDatabaseRequest{
		Parent:          fmt.Sprintf("projects/%s/instances/%s", projectName, instanceID),
		CreateStatement: fmt.Sprintf("CREATE DATABASE %s", databaseName),
		DatabaseDialect: databasepb.DatabaseDialect_POSTGRESQL,
	},
	)
	require.NoError(t, err, "could not create database")
	_, err = op.Wait(ctx)
	require.NoError(t, err, "create database failed")
}
