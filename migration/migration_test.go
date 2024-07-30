package migration_test

import (
	gs "cloud.google.com/go/spanner"
	"cloud.google.com/go/spanner/admin/instance/apiv1/instancepb"
	"context"
	"embed"
	"fmt"
	"github.com/dora-network/dora-service-utils/migration"
	"github.com/dora-network/dora-service-utils/spanner"
	"github.com/dora-network/dora-service-utils/testing/emulators"
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

var emulator *emulators.SpannerEmulator

func TestMain(m *testing.M) {
	var err error
	emulator, err = emulators.NewSpannerEmulator()
	if err != nil {
		log.Fatalf("could not create spanner emulator: %+v", err)
	}

	code := m.Run()
	os.Exit(code)
}

func Test_Migrate(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Hour)
	defer cancel()
	resource, err := emulator.Start()
	require.NoError(t, err, "couldn't start cloud spanner emulator")

	defer t.Cleanup(func() {
		require.NoError(t, emulator.Cleanup(resource), "could not purge spanner emulator")
	})

	// Now that we've started the emulator container we need to get the host and port
	// and set the SPANNER_EMULATOR_HOST env var so that we can use it with the API
	// as specified in https://cloud.google.com/spanner/docs/emulator.
	// This should allow the client library to connect to the appropriate cloud spanner
	// instance we have created for the test
	hostAndPort := resource.GetHostPort("9010/tcp")
	require.NoError(t, os.Setenv("SPANNER_EMULATOR_HOST", hostAndPort), "setting spanner emulator host")

	config := spanner.Config{
		ProjectID:  projectName,
		InstanceID: instanceID,
		DatabaseID: databaseName,
	}
	client, err := spanner.NewClient(ctx, config)
	require.NoError(t, err, "could not create migration spanner client")
	err = emulator.Wait(ctx, client)
	require.NoError(t, err, "could not connect to spanner emulator")

	setup(t, ctx)
	err = migration.Migrate(context.Background(), migrationsFS, config, client)
	require.NoError(t, err)
	got, err := migration.GetCurrentVersion(ctx, client, migration.SchemaVersionTable)
	require.NoError(t, err)
	var want int64 = 2
	assert.Equal(t, want, got)

	iter := client.Single().Query(ctx, gs.Statement{
		SQL:    "select * from information_schema.tables where  table_schema = 'public' and table_type = 'BASE TABLE';",
		Params: map[string]interface{}{"p1": migration.SchemaVersionTable},
	})

	defer iter.Stop()

	gotTables := make([]string, 0)
	err = iter.Do(func(r *gs.Row) error {
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

func TestEnsureMigrationTable(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	resource, err := emulator.Start()
	require.NoError(t, err, "couldn't start cloud spanner emulator")

	defer t.Cleanup(func() {
		require.NoError(t, emulator.Cleanup(resource), "could not purge spanner emulator")
	})

	// Now that we've started the emulator container we need to get the host and port
	// and set the SPANNER_EMULATOR_HOST env var so that we can use it with the API
	// as specified in https://cloud.google.com/spanner/docs/emulator.
	// This should allow the client library to connect to the appropriate cloud spanner
	// instance we have created for the test
	hostAndPort := resource.GetHostPort("9010/tcp")
	require.NoError(t, os.Setenv("SPANNER_EMULATOR_HOST", hostAndPort), "setting spanner emulator host")

	config := spanner.Config{
		ProjectID:  projectName,
		InstanceID: instanceID,
		DatabaseID: databaseName,
	}
	client, err := spanner.NewClient(ctx, config)
	require.NoError(t, err, "could not create migration spanner client")
	err = emulator.Wait(ctx, client)
	require.NoError(t, err, "could not connect to spanner emulator")

	setup(t, ctx)

	require.NoError(t, migration.EnsureMigrationTable(ctx, config, client, migration.SchemaVersionTable))

	iter := client.Single().Query(ctx, gs.Statement{
		SQL:    "select * from information_schema.tables where table_name = $1;",
		Params: map[string]interface{}{"p1": migration.SchemaVersionTable},
	})
	defer iter.Stop()
	tableFound := false
	err = iter.Do(func(row *gs.Row) error {
		var tableName string
		if err := row.ColumnByName("table_name", &tableName); err != nil {
			return err
		}
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
	resource, err := emulator.Start()
	require.NoError(t, err, "couldn't start cloud spanner emulator")

	defer t.Cleanup(func() {
		require.NoError(t, emulator.Cleanup(resource), "could not purge spanner emulator")
	})

	// Now that we've started the emulator container we need to get the host and port
	// and set the SPANNER_EMULATOR_HOST env var so that we can use it with the API
	// as specified in https://cloud.google.com/spanner/docs/emulator.
	// This should allow the client library to connect to the appropriate cloud spanner
	// instance we have created for the test
	hostAndPort := resource.GetHostPort("9010/tcp")
	require.NoError(t, os.Setenv("SPANNER_EMULATOR_HOST", hostAndPort), "setting spanner emulator host")

	config := spanner.Config{
		ProjectID:  projectName,
		InstanceID: instanceID,
		DatabaseID: databaseName,
	}
	client, err := spanner.NewClient(ctx, config)
	require.NoError(t, err, "could not create migration spanner client")
	err = emulator.Wait(ctx, client)
	require.NoError(t, err, "could not connect to spanner emulator")

	setup(t, ctx)

	err = migration.EnsureMigrationTable(ctx, config, client, migration.SchemaVersionTable)
	require.NoError(t, err, "could not create schema migrations table")
	var want int64 = 20
	_, err = client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *gs.ReadWriteTransaction) error {
		stmt := gs.Statement{
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
	resource, err := emulator.Start()
	require.NoError(t, err, "couldn't start cloud spanner emulator")

	defer t.Cleanup(func() {
		require.NoError(t, emulator.Cleanup(resource), "could not purge spanner emulator")
	})

	// Now that we've started the emulator container we need to get the host and port
	// and set the SPANNER_EMULATOR_HOST env var so that we can use it with the API
	// as specified in https://cloud.google.com/spanner/docs/emulator.
	// This should allow the client library to connect to the appropriate cloud spanner
	// instance we have created for the test
	hostAndPort := resource.GetHostPort("9010/tcp")
	require.NoError(t, os.Setenv("SPANNER_EMULATOR_HOST", hostAndPort), "setting spanner emulator host")

	config := spanner.Config{
		ProjectID:  projectName,
		InstanceID: instanceID,
		DatabaseID: databaseName,
	}
	client, err := spanner.NewClient(ctx, config)
	require.NoError(t, err, "could not create migration spanner client")
	err = emulator.Wait(ctx, client)
	require.NoError(t, err, "could not connect to spanner emulator")

	setup(t, ctx)

	err = migration.EnsureMigrationTable(ctx, config, client, migration.SchemaVersionTable)
	require.NoError(t, err, "could not create schema migrations table")
	var want int64 = 21
	_, err = client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *gs.ReadWriteTransaction) error {
		stmt := gs.Statement{
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

	iter := client.Single().Query(ctx, gs.Statement{
		SQL: "select version from public.schema_migrations order by version desc;",
	})

	versions := make([]int64, 0)
	err = iter.Do(func(r *gs.Row) error {
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
	i, err := emulator.SetupInstance(ctx, projectName, instanceID)
	require.NoError(t, err, "waiting for instance creation to complete")
	require.Equal(t, instancepb.Instance_READY, i.State, "instance not ready after wait")
}

func setupDatabase(t *testing.T, ctx context.Context) {
	t.Helper()
	_, err := emulator.SetupDatabase(ctx, projectName, instanceID, databaseName)
	require.NoError(t, err, "create database failed")
}
