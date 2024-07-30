package spanner

import "fmt"

// Config represents the configuration required to access the Google Cloud Spanner
// instance and database for migration.
type Config struct {
	// ProjectID is the Google cloud project id
	ProjectID string
	// InstanceID is the name of the Google cloud spanner instance
	InstanceID string
	// DatabaseID is the name of the Google cloud spanner database
	DatabaseID string
	// CredentialsFile is the path Google cloud spanner credentials file
	CredentialsFile string
}

func (c Config) URL() string {
	return fmt.Sprintf("projects/%s/instances/%s/databases/%s", c.ProjectID, c.InstanceID, c.DatabaseID)
}
