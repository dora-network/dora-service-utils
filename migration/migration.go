package migration

import (
	"cloud.google.com/go/spanner"
	"cloud.google.com/go/spanner/admin/database/apiv1/databasepb"
	"context"
	"fmt"
	"io/fs"
	"log"
)

const (
	SchemaVersionTable = "schema_migrations"
)

// Migrate applies each migration in the migrations directory to the Cloud Spanner database.
func Migrate(ctx context.Context, migrationsFS fs.FS, cfg Config, client Client) error {
	if err := EnsureMigrationTable(ctx, cfg, client, SchemaVersionTable); err != nil {
		return err
	}

	currentVersion, err := GetCurrentVersion(ctx, client, SchemaVersionTable)
	if err != nil {
		return err
	}
	return fs.WalkDir(migrationsFS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		parser := NewParser(migrationsFS, path)

		version, err := parser.ParseVersion()
		if err != nil {
			return err
		}

		if version < currentVersion {
			log.Printf("Skipping migration %d, current version: %d\n", version, currentVersion)
			return nil
		}

		stmts, err := parser.ParseMigrationFile()
		if err != nil {
			return err
		}

		req := databasepb.UpdateDatabaseDdlRequest{
			Database:   cfg.URL(),
			Statements: stmts,
		}
		op, err := client.UpdateDatabaseDdl(ctx, &req)
		if err != nil {
			return err
		}

		err = op.Wait(ctx)
		if err != nil && !op.Done() {
			return fmt.Errorf("failed to apply migration %d: %w", version, err)
		}

		if err != nil && op.Done() {
			return fmt.Errorf("migrations applied but with errors %d: %w", version, err)
		}

		if err = SetCurrentVersion(ctx, client, SchemaVersionTable, version); err != nil {
			return fmt.Errorf("failed to set current migration version %d: %w", version, err)
		}

		return nil
	})
}

func EnsureMigrationTable(ctx context.Context, config Config, client Client, tableName string) error {
	rows := client.Single().Read(ctx, tableName, spanner.AllKeys(), []string{"version"})
	if err := rows.Do(func(row *spanner.Row) error {
		return nil
	}); err == nil {
		return nil
	}

	req := databasepb.UpdateDatabaseDdlRequest{
		Database: config.URL(),
		Statements: []string{
			fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s  (
				version BIGINT NOT NULL PRIMARY KEY,
				created_at TIMESTAMPTZ NOT NULL DEFAULT now()
			);`, tableName),
		},
	}
	op, err := client.UpdateDatabaseDdl(ctx, &req)
	if err != nil {
		return err
	}

	err = op.Wait(ctx)
	if err != nil && !op.Done() {
		return err
	}
	if err != nil && op.Done() {
		return err
	}
	return nil
}

func GetCurrentVersion(ctx context.Context, client Client, tableName string) (int64, error) {
	stmt := spanner.NewStatement(fmt.Sprintf("SELECT version FROM %s ORDER BY version DESC LIMIT 1", tableName))
	iter := client.Single().Query(ctx, stmt)
	defer iter.Stop()

	var version int64
	if err := iter.Do(func(r *spanner.Row) error {
		return r.Column(0, &version)
	}); err != nil {
		return 0, err
	}
	return version, nil
}

func SetCurrentVersion(ctx context.Context, client Client, tableName string, version int64) error {
	_, err := client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		stmt := spanner.Statement{
			SQL: fmt.Sprintf(`INSERT INTO %s (version) VALUES
                                ($1)`, tableName),
			Params: map[string]interface{}{"p1": version},
		}
		_, err := txn.Update(ctx, stmt)
		if err != nil {
			return err
		}
		return err
	})

	if err != nil {
		return err
	}

	return nil
}
