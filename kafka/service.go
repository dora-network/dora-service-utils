package kafka

import (
	"context"

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
) {
	for _, msg := range messages {
		bs, err := proto.Marshal(msg)
		if err != nil {
			logger.Err(err).Msgf("failed to marshal transaction response: %v", msg)
			continue
		}
		client.Produce(
			ctx, &kgo.Record{
				Topic: topic,
				Value: bs,
			}, func(r *kgo.Record, err error) {
				if err != nil {
					logger.Err(err).
						Str("topic", topic).
						Any("message", msg).
						Msg("failed to produce transaction response record")
					return
				}

				if logger.GetLevel() == zerolog.DebugLevel {
					logger.Debug().
						Str("topic", topic).
						Any("message", msg).
						Msg("Produced transaction response record")
				}
			},
		)
	}
}
