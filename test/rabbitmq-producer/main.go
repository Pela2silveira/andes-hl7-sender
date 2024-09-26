package main

import (
	"os"
    "os/signal"
	"syscall"
	"time"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"github.com/streadway/amqp"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	mongoURI        = "mongodb://appuser:appsecret@mongodb:27017/hospital"
	mongoDatabase   = "hospital"
	mongoCollection = "patients"
	rabbitmqURI     = "amqp://guest:guest@rabbitmq:5672/"
	queueName       = "adt04mosaiq"
)

func main() {
	// Create a root context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle termination signals to cancel the context
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("Shutting down gracefully...")
		cancel()
	}()
	
	// Connect to MongoDB
	clientOptions := options.Client().ApplyURI(mongoURI)
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		log.Fatalf("Error conectando a MongoDB: %v", err)
	}
	defer client.Disconnect(ctx)

	// Access the collection
	collection := client.Database(mongoDatabase).Collection(mongoCollection)

	// Connect to RabbitMQ
	conn, err := amqp.Dial(rabbitmqURI)
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	defer conn.Close()

	// Create a channel
	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("Failed to open a channel: %v", err)
	}
	defer ch.Close()

	// Declare a queue
	_, err = ch.QueueDeclare(
		queueName, // queue name
		true,      // durable
		false,     // delete when unused
		false,     // exclusive
		false,     // no-wait
		nil,       // arguments
	)
	if err != nil {
		log.Fatalf("Failed to declare a queue: %v", err)
	}

	for {
		select {
		case <-ctx.Done():
			fmt.Println("Context cancelled, exiting loop")
			return
		default:
			// Retrieve data from MongoDB
			cursor, err := collection.Find(ctx, bson.D{})
			if err != nil {
				log.Fatalf("Failed to find documents: %v", err)
			}

			for cursor.Next(ctx) {
				var patient bson.M
				if err := cursor.Decode(&patient); err != nil {
					log.Fatalf("Failed to decode document: %v", err)
				}

				jsonData, err := json.Marshal(patient)
				if err != nil {
					log.Fatalf("Failed to marshal document: %v", err)
				}

				err = ch.Publish(
					"",
					queueName,
					false,
					false,
					amqp.Publishing{
						ContentType: "application/json",
						Body:        jsonData,
					})
				if err != nil {
					log.Fatalf("Failed to publish message: %v", err)
				}

				fmt.Printf("Sent message: %s\n", jsonData)
			}

			cursor.Close(ctx)
			time.Sleep(5 * time.Second) // Sleep to avoid constant polling
		}
	}
}
