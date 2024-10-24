package secrets_test

import (
	"context"
	"fmt"
	"github.com/dora-network/dora-service-utils/secrets"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"strings"
	"testing"
	"time"
)

func TestSecrets(t *testing.T) {
	cwd, err := os.Getwd()
	require.NoError(t, err)

	entries, err := os.ReadDir(cwd)
	require.NoError(t, err)

	var credentialsFile string

	for _, entry := range entries {
		if entry.IsDir() || strings.HasSuffix(entry.Name(), ".go") {
			continue
		}

		if entry.Name() == "service-account.json" {
			credentialsFile = fmt.Sprintf("%s/%s", cwd, entry.Name())
			break
		}
	}

	if credentialsFile == "" {
		t.Skipf("credentials file not found in %s", cwd)
	}

	// Set the GOOGLE_APPLICATION_CREDENTIALS environment variable
	require.NoError(t, os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", credentialsFile))
	projectID := "bond-market-413717"

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	secret, err := secrets.GetSecret(ctx, projectID, secrets.DefaultKafkaUserSecretID)
	require.NoError(t, err)
	require.NotNil(t, secret)
	assert.Equal(t, "user1", string(secret))

	secret, err = secrets.GetSecret(ctx, projectID, secrets.DefaultRedisUserSecretID)
	require.NoError(t, err)
	require.NotNil(t, secret)
	assert.Equal(t, "doratesting", string(secret))

	secretID := "test-id"
	payload := "test-payload"
	_, err = secrets.CreateSecret(context.Background(), projectID, secretID, []byte(payload))
	require.NoError(t, err)

	secret, err = secrets.GetSecret(ctx, projectID, secretID)
	require.NoError(t, err)
	require.NotNil(t, secret)
	assert.Equal(t, payload, string(secret))
}
