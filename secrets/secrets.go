package secrets

import (
	sm "cloud.google.com/go/secretmanager/apiv1"
	smpb "cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"context"
	"fmt"
	"os"
)

const (
	GoogleApplicationCredentialsEnv = "GOOGLE_APPLICATION_CREDENTIALS"
	DefaultKafkaUserSecretID        = "kafka-user"
	DefaultKafkaPasswordSecretID    = "kafka-password"
	DefaultRedisUserSecretID        = "redis-user"
	DefaultRedisPasswordSecretID    = "redis-password"
)

func GetSecret(ctx context.Context, projectID, secretID string) ([]byte, error) {
	if os.Getenv(GoogleApplicationCredentialsEnv) == "" {
		return nil, fmt.Errorf("%s not set", GoogleApplicationCredentialsEnv)
	}

	client, err := sm.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create secrets client: %w", err)
	}

	defer client.Close()

	secretName := fmt.Sprintf("projects/%s/secrets/%s/versions/latest", projectID, secretID)
	req := &smpb.AccessSecretVersionRequest{
		Name: secretName,
	}

	result, err := client.AccessSecretVersion(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve secret from secret manager: %w", err)
	}

	return result.GetPayload().GetData(), nil
}
