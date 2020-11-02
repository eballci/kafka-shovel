package services

import (
	"context"
	"fmt"

	"github.com/Shopify/sarama"
	"github.com/Trendyol/kafka-shovel/kafka"
	"github.com/google/uuid"
)

const runningKeyHeader = "running_key"

type Interceptor func(ctx context.Context, message *sarama.ConsumerMessage) context.Context

type eventHandler struct {
	runningKey string
	service    Service
}

func NewEventHandler(service Service) kafka.EventHandler {
	return &eventHandler{
		service:    service,
		runningKey: uuid.New().String(),
	}
}

func (e *eventHandler) Setup(session sarama.ConsumerGroupSession) error {
	return nil
}

func (e *eventHandler) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

func (e *eventHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for message := range claim.Messages() {
		fmt.Printf("Received key: %s, topic: %s \n", string(message.Key), message.Topic)
		if e.doesMessageProcessed(message) {
			fmt.Printf("Message already is processed. Shovel execution halted.  key: %s, topic: %s \n", string(message.Key), message.Topic)
			session.MarkMessage(message, "")
			continue
		}

		err := e.service.OperateEvent(context.Background(), message)
		if err != nil {
			fmt.Println("Error executing err: ", err)
		}

		session.MarkMessage(message, "")
	}

	return nil
}

func (e *eventHandler) doesMessageProcessed(message *sarama.ConsumerMessage) bool {
	for i := 0; i < len(message.Headers); i++ {
		if string(message.Headers[i].Key) == runningKeyHeader {
			if string(message.Headers[i].Value) == e.runningKey {
				return true
			} else {
				message.Headers[i].Value = []byte(e.runningKey)
				return false
			}
		}
	}

	message.Headers = append(message.Headers, &sarama.RecordHeader{
		Key:   []byte(runningKeyHeader),
		Value: []byte(e.runningKey),
	})
	return false
}
