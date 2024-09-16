package rpcclient

import (
	"context"
	"errors"
	"math/rand"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

const REPLY_QUEUE = "amq.rabbitmq.reply-to"

func randomString(l int) string {
	bytes := make([]byte, l)
	for i := 0; i < l; i++ {
		bytes[i] = byte(randInt(65, 90))
	}
	return string(bytes)
}

func randInt(min int, max int) int {
	return min + rand.Intn(max-min)
}

func MakeRequest(request string, routingKey string) (string, error) {
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil {
		return "", err
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		return "", err
	}
	defer ch.Close()

	msgs, err := ch.Consume(
		REPLY_QUEUE, // queue
		"",          // consumer
		true,        // auto-ack
		false,       // exclusive
		false,       // no-local
		false,       // no-wait
		nil,         // args
	)
	if err != nil {
		return "", err
	}

	corrId := randomString(32)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = ch.PublishWithContext(ctx,
		"",         // exchange
		routingKey, // routing key
		false,      // mandatory
		false,      // immediate
		amqp.Publishing{
			ContentType:   "text/plain",
			CorrelationId: corrId,
			ReplyTo:       REPLY_QUEUE,
			Body:          []byte(request),
		})
	if err != nil {
		return "", err
	}

	for d := range msgs {
		if corrId == d.CorrelationId {
			return string(d.Body), nil
		}
	}
	return "", errors.New("No answer")
}
