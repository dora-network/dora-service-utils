package secrets

import (
	"context"
	"fmt"
	"os"

	sm "cloud.google.com/go/secretmanager/apiv1"
	smpb "cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
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

func CreateSecret(ctx context.Context, projectID, secretID string, payload []byte) (string, error) {
	if os.Getenv(GoogleApplicationCredentialsEnv) == "" {
		return "", fmt.Errorf("%s not set", GoogleApplicationCredentialsEnv)
	}

	client, err := sm.NewClient(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to create secrets client: %w", err)
	}

	defer client.Close()

	createSecretReq := &smpb.CreateSecretRequest{
		Parent:   fmt.Sprintf("projects/%s", projectID),
		SecretId: secretID,
		Secret: &smpb.Secret{
			Replication: &smpb.Replication{
				Replication: &smpb.Replication_Automatic_{
					Automatic: &smpb.Replication_Automatic{},
				},
			},
		},
	}

	secret, err := client.CreateSecret(ctx, createSecretReq)
	if err != nil {
		return "", fmt.Errorf("failed to create secret: %s", err.Error())
	}

	addSecretVersionReq := &smpb.AddSecretVersionRequest{
		Parent: secret.Name,
		Payload: &smpb.SecretPayload{
			Data: payload,
		},
	}

	version, err := client.AddSecretVersion(ctx, addSecretVersionReq)
	if err != nil {
		return "", fmt.Errorf("failed to add secret version: %s", err.Error())
	}

	return version.Name, nil
}
