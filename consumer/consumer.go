package consumer

import (
	"encoding/json"
	"log"
	"my-chat-app/services"
	"my-chat-app/websockets"

	"github.com/streadway/amqp"
)

type Consumer struct {
	conn        *amqp.Connection
	channel     *amqp.Channel
	chatService services.ChatService
	queueName   string
}

func NewConsumer(amqpURL string, chatService services.ChatService) (*Consumer, error) {
	conn, err := amqp.Dial(amqpURL)
	if err != nil {
		return nil, err
	}

	channel, err := conn.Channel()
	if err != nil {
		return nil, err
	}

	// Declare the queue here (within the NewConsumer function)
	_, err = channel.QueueDeclare(
		"chat_queue", // name
		true,         // durable
		false,        // delete when unused
		false,        // exclusive
		false,        // no-wait
		nil,          // arguments
	)
	if err != nil {
		return nil, err // Return the error if queue declaration fails
	}

	return &Consumer{
		conn:        conn,
		channel:     channel,
		chatService: chatService,
		queueName:   "chat_queue",
	}, nil
}

func (c *Consumer) StartConsuming() error {
	msgs, err := c.channel.Consume(
		c.queueName, // queue
		"",          // consumer
		false,       // auto-ack  <-- CRITICAL:  Set to false!
		false,       // exclusive
		false,       // no-local
		false,       // no-wait
		nil,         // args
	)
	if err != nil {
		return err
	}

	forever := make(chan bool)

	go func() {
		for d := range msgs {
			log.Printf("Received a message: %s", d.Body)

			var wsMessage websockets.WebSocketMessage
			if err := json.Unmarshal(d.Body, &wsMessage); err != nil {
				log.Printf("Error unmarshaling message: %v", err)
				d.Nack(false, false) // Negatively acknowledge, don't requeue
				continue
			}

			// Now, use the *FULL* SendMessage method, with file info.
			_, err := c.chatService.SendMessage(
				wsMessage.SenderID,
				wsMessage.ReceiverID,
				wsMessage.GroupID,
				wsMessage.Content,
				wsMessage.ReplyToMessageID,
				wsMessage.FileName,
				wsMessage.FilePath,
				wsMessage.FileType,
				wsMessage.FileSize,
				wsMessage.FileChecksum,
			)
			if err != nil {
				log.Printf("Error processing message: %v", err)
				d.Nack(false, true) // Negatively acknowledge, requeue
			} else {
				d.Ack(false) // Acknowledge message processing
			}
		}
	}()

	log.Printf(" [*] Waiting for messages. To exit press CTRL+C")
	<-forever

	return nil
}

func (c *Consumer) Close() {
	if c.channel != nil {
		c.channel.Close()
	}
	if c.conn != nil {
		c.conn.Close()
	}
}
