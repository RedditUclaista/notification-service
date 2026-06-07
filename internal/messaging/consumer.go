package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/RedditUclaista/notification-service/internal/usecases"
	amqp "github.com/rabbitmq/amqp091-go"
)

type AMQPConsumer struct {
	connUrl string
	useCase usecases.NotificationUseCase
}

type BrokerEvent struct {
	UserID string `json:"user_id"` // Usuario que recibirá la notificación (dueño del post)
	PostID string `json:"post_id"` // ID del post involucrado
	Type   string `json:"type"`    // "vote" o "comment"
}

func NewAMQPConsumer(connUrl string, uc usecases.NotificationUseCase) *AMQPConsumer {
	return &AMQPConsumer{
		connUrl: connUrl,
		useCase: uc,
	}
}

func (c *AMQPConsumer) Start(ctx context.Context) {
	// Iniciamos la reconexión automática en una Goroutine
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				fmt.Println("AMQP: Attempting to connect to LavinMQ...")
				conn, err := amqp.Dial(c.connUrl)
				if err != nil {
					fmt.Printf("AMQP: Connection failed: %v. Retrying in 5 seconds...\n", err)
					time.Sleep(5 * time.Second)
					continue
				}

				ch, err := conn.Channel()
				if err != nil {
					fmt.Printf("AMQP: Failed to open channel: %v\n", err)
					conn.Close()
					time.Sleep(5 * time.Second)
					continue
				}

				err = c.setupTopology(ch)
				if err != nil {
					fmt.Printf("AMQP: Failed to setup topology: %v\n", err)
					ch.Close()
					conn.Close()
					time.Sleep(5 * time.Second)
					continue
				}

				// Escuchamos los mensajes de la cola de notificaciones
				deliveries, err := ch.Consume(
					"notification.queue",
					"",
					false, // auto-ack = false (manejamos confirmación manual)
					false,
					false,
					false,
					nil,
				)
				if err != nil {
					fmt.Printf("AMQP: Failed to consume: %v\n", err)
					ch.Close()
					conn.Close()
					time.Sleep(5 * time.Second)
					continue
				}

				fmt.Println("AMQP: Connected to LavinMQ. Listening for vote and comment events...")

				closeChan := conn.NotifyClose(make(chan *amqp.Error))

				c.handleDeliveries(ctx, deliveries)

				// Esperamos a que se caiga la conexión
				errClose := <-closeChan
				fmt.Printf("AMQP: Connection closed: %v. Reconnecting...\n", errClose)
				ch.Close()
				conn.Close()
			}
		}
	}()
}

func (c *AMQPConsumer) setupTopology(ch *amqp.Channel) error {
	// Declaramos el Exchange de Votos
	err := ch.ExchangeDeclare(
		"vote.exchange",
		"topic",
		true,  // durable
		false, // auto-deleted
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	// Declaramos el Exchange de Comentarios
	err = ch.ExchangeDeclare(
		"comment.exchange",
		"topic",
		true,  // durable
		false, // auto-deleted
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	// Declaramos nuestra cola de Notificaciones
	_, err = ch.QueueDeclare(
		"notification.queue",
		true,  // durable
		false, // auto-delete
		false, // exclusive
		false,
		nil,
	)
	if err != nil {
		return err
	}

	// Vinculamos la cola a 'vote.exchange' con comodín '#' para recibir cualquier llave de enrutamiento
	err = ch.QueueBind(
		"notification.queue",
		"#",
		"vote.exchange",
		false,
		nil,
	)
	if err != nil {
		return err
	}

	// Vinculamos la cola a 'comment.exchange'
	err = ch.QueueBind(
		"notification.queue",
		"#",
		"comment.exchange",
		false,
		nil,
	)
	if err != nil {
		return err
	}

	return nil
}

func (c *AMQPConsumer) handleDeliveries(ctx context.Context, deliveries <-chan amqp.Delivery) {
	for d := range deliveries {
		var event BrokerEvent
		err := json.Unmarshal(d.Body, &event)
		if err != nil {
			fmt.Printf("AMQP: Invalid message JSON: %v. Body: %s\n", err, string(d.Body))
			// Hacemos reject sin reenviar a la cola para no trabar el flujo
			_ = d.Reject(false)
			continue
		}

		if event.UserID == "" || event.PostID == "" || event.Type == "" {
			fmt.Printf("AMQP: Missing event fields. Event: %+v\n", event)
			_ = d.Reject(false)
			continue
		}

		// Enviamos al UseCase para encolarse en el buffer de 2 segundos
		err = c.useCase.BufferEvent(ctx, event.UserID, event.PostID, event.Type)
		if err != nil {
			fmt.Printf("AMQP: Failed to process event in buffer: %v\n", err)
			// Re-encolamos para reintento en caso de falla interna
			_ = d.Nack(false, true)
			continue
		}

		// Alerta procesada y encolada con éxito, confirmamos
		_ = d.Ack(false)
	}
}
