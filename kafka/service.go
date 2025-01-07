package kafka

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"
	"github.com/twmb/franz-go/pkg/kgo"
	"google.golang.org/protobuf/proto"
)

// SendMessageToTopic will sends to the message specific topic and it will logs the errors if any error occures
func SendMessageToTopic(
	ctx context.Context,
	client *kgo.Client,
	logger zerolog.Logger,
	topic string,
	messages ...proto.Message,
) error {
	records := make([]*kgo.Record, 0)
	for _, msg := range messages {
		bs, err := proto.Marshal(msg)
		if err != nil {
			return fmt.Errorf("failed to marshal transaction response: %v", msg)
		}
		records = append(records, &kgo.Record{
			Topic: topic,
			Value: bs,
		})
	}
	result := client.ProduceSync(ctx, records...)
	return result.FirstErr()
}
