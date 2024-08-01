package app

import (
	"context"
	"log"
	"time"

	"github.com/thecaffeinedev/eventflow-pg-rmq/pkg/health"
	"github.com/thecaffeinedev/eventflow-pg-rmq/pkg/pglistener"
	"github.com/thecaffeinedev/eventflow-pg-rmq/pkg/rabbitmq"
)

type App struct {
	PgListener   *pglistener.PgListener
	RmqPublisher *rabbitmq.RabbitMQPublisher
}

func New(pgConnString, rabbitMQURL string) (*App, error) {
	for {
		err := health.Check(pgConnString, rabbitMQURL)
		if err == nil {
			log.Println("Health check passed. Continuing with application startup.")
			break
		}
		log.Printf("Health check failed: %v. Retrying in 5 seconds...\n", err)
		time.Sleep(5 * time.Second)
	}

	pgListener, err := pglistener.New(pgConnString)
	if err != nil {
		return nil, err
	}

	var rmqPublisher *rabbitmq.RabbitMQPublisher
	for i := 0; i < 5; i++ {
		rmqPublisher, err = rabbitmq.New(rabbitMQURL)
		if err == nil {
			break
		}
		log.Printf("Failed to create RabbitMQ publisher (attempt %d/5): %v. Retrying in 5 seconds...\n", i+1, err)
		time.Sleep(5 * time.Second)
	}
	if rmqPublisher == nil {
		return nil, err
	}

	return &App{
		PgListener:   pgListener,
		RmqPublisher: rmqPublisher,
	}, nil
}

func (a *App) Run(ctx context.Context) error {
	changes, err := a.PgListener.Listen(ctx)
	if err != nil {
		return err
	}

	for change := range changes {
		if err := a.RmqPublisher.Publish(change); err != nil {
			log.Printf("Failed to publish change: %v", err)
		} else {
			log.Printf("Successfully published change: %v", change)
		}
	}

	return nil
}

func (a *App) Close() {
	if a.PgListener != nil {
		a.PgListener.Close()
	}
	if a.RmqPublisher != nil {
		a.RmqPublisher.Close()
	}
}