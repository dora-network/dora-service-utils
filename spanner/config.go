package spanner

import (
	"fmt"
	"time"
)

// Config represents the configuration required to access the Google Cloud Spanner
// instance and database for migration.
type Config struct {
	// ProjectID is the Google cloud project id
	ProjectID string `mapstructure:"project_id"`
	// InstanceID is the name of the Google cloud spanner instance
	InstanceID string `mapstructure:"instance_id"`
	// DatabaseID is the name of the Google cloud spanner database
	DatabaseID string `mapstructure:"database_id"`
	// CredentialsFile is the path Google cloud spanner credentials file
	CredentialsFile string `mapstructure:"credentials_file"`
	// MigrationTimeout is the timeout for the migration process
	MigrationTimeout time.Duration `mapstructure:"migration_timeout"`
}

func (c Config) URL() string {
	return fmt.Sprintf("projects/%s/instances/%s/databases/%s", c.ProjectID, c.InstanceID, c.DatabaseID)
}
