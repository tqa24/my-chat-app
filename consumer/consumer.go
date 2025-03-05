package consumer

import (
	"encoding/json"
	"log"
	"my-chat-app/services"
	"my-chat-app/websockets"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/streadway/amqp"
)

const (
	MaxRetryCount    = 5
	RetryCountHeader = "x-retry-count"
)

type Consumer struct {
	conn             *amqp.Connection
	channel          *amqp.Channel
	ChatService      services.ChatService
	queueName        string
	dlxName          string // Dead Letter Exchange name
	dlqName          string // Dead Letter Queue name
	retryMetric      *prometheus.CounterVec
	deadLetterMetric prometheus.Counter
}

func NewConsumer(amqpURL string, chatService services.ChatService,
	retryMetric *prometheus.CounterVec, deadLetterMetric prometheus.Counter) (*Consumer, error) {
	conn, err := amqp.Dial(amqpURL)
	if err != nil {
		return nil, err
	}

	channel, err := conn.Channel()
	if err != nil {
		return nil, err
	}

	// First, try to delete the existing queue if it exists
	// This is necessary because we can't modify the arguments of an existing queue
	_, err = channel.QueueDelete(
		"chat_queue", // name
		false,        // ifUnused (false = delete even if in use)
		false,        // ifEmpty (false = delete even if not empty)
		false,        // noWait
	)
	// Ignore the error - it might not exist yet

	// Declare the Dead Letter Exchange
	dlxName := "chat_dlx"
	err = channel.ExchangeDeclare(
		dlxName,  // name
		"direct", // type
		true,     // durable
		false,    // auto-deleted
		false,    // internal
		false,    // no-wait
		nil,      // arguments
	)
	if err != nil {
		return nil, err
	}

	// Declare the Dead Letter Queue
	dlqName := "chat_dlq"
	_, err = channel.QueueDeclare(
		dlqName, // name
		true,    // durable
		false,   // delete when unused
		false,   // exclusive
		false,   // no-wait
		nil,     // arguments
	)
	if err != nil {
		return nil, err
	}

	// Bind the Dead Letter Queue to the Dead Letter Exchange
	err = channel.QueueBind(
		dlqName,  // queue name
		"failed", // routing key
		dlxName,  // exchange
		false,    // no-wait
		nil,      // arguments
	)
	if err != nil {
		return nil, err
	}

	// Declare the main queue with dead letter configuration
	args := amqp.Table{
		"x-dead-letter-exchange":    dlxName,
		"x-dead-letter-routing-key": "failed",
	}

	_, err = channel.QueueDeclare(
		"chat_queue", // name
		true,         // durable
		false,        // delete when unused
		false,        // exclusive
		false,        // no-wait
		args,         // arguments with DLX configuration
	)
	if err != nil {
		return nil, err
	}

	return &Consumer{
		conn:             conn,
		channel:          channel,
		ChatService:      chatService,
		queueName:        "chat_queue",
		dlxName:          dlxName,
		dlqName:          dlqName,
		retryMetric:      retryMetric,
		deadLetterMetric: deadLetterMetric,
	}, nil
}

func (c *Consumer) StartConsuming() error {
	msgs, err := c.channel.Consume(
		c.queueName, // queue
		"",          // consumer
		false,       // auto-ack  <-- CRITICAL: Set to false!
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

			// Check retry count from headers
			retryCount := 0
			if d.Headers != nil {
				if count, exists := d.Headers[RetryCountHeader]; exists {
					if countInt, ok := count.(int32); ok {
						retryCount = int(countInt)
					}
				}
			}

			var wsMessage websockets.WebSocketMessage
			if err := json.Unmarshal(d.Body, &wsMessage); err != nil {
				log.Printf("Error unmarshaling message: %v", err)
				// Don't retry parse errors - send directly to DLQ
				c.deadLetterMetric.Inc() // Increment dead letter counter
				d.Nack(false, false)
				continue
			}

			// Process the message
			_, err := c.ChatService.SendMessage(
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

				// Check if we've reached max retries
				if retryCount >= MaxRetryCount {
					log.Printf("Message failed after %d retries, sending to DLQ", MaxRetryCount)
					c.deadLetterMetric.Inc() // Increment dead letter counter
					d.Nack(false, false)     // Don't requeue, will go to DLQ
				} else {
					// Increment retry count and republish
					retryCount++
					log.Printf("Retrying message, attempt %d of %d", retryCount, MaxRetryCount)

					// Increment retry metric with the attempt number
					c.retryMetric.WithLabelValues(strconv.Itoa(retryCount)).Inc()

					// Create headers for the next attempt
					headers := amqp.Table{
						RetryCountHeader: retryCount,
					}

					// Republish with updated retry count
					err = c.channel.Publish(
						"",          // exchange
						c.queueName, // routing key
						false,       // mandatory
						false,       // immediate
						amqp.Publishing{
							ContentType: "application/json",
							Body:        d.Body,
							Headers:     headers,
						},
					)

					if err != nil {
						log.Printf("Error republishing message: %v", err)
					}

					// Acknowledge the original message
					d.Ack(false)
				}
			} else {
				// Successfully processed
				d.Ack(false)
			}
		}
	}()

	log.Printf(" [*] Waiting for messages. To exit press CTRL+C")
	<-forever

	return nil
}

// Add a method to process failed messages from DLQ if needed
func (c *Consumer) ProcessDeadLetterQueue() error {
	msgs, err := c.channel.Consume(
		c.dlqName, // queue
		"",        // consumer
		false,     // auto-ack
		false,     // exclusive
		false,     // no-local
		false,     // no-wait
		nil,       // args
	)
	if err != nil {
		return err
	}

	go func() {
		for d := range msgs {
			log.Printf("Processing dead letter message: %s", d.Body)
			// Here you could implement logic to handle failed messages
			// For example, storing them in a database for later analysis

			// Acknowledge the message from DLQ
			d.Ack(false)
		}
	}()

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
