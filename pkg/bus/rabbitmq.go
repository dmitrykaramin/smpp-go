package bus

import (
	"SMSRouter/internal"
	"fmt"
	"github.com/streadway/amqp"
)

func InitAMQP() (*amqp.Connection, error) {
	configuration, err := internal.GetConfig()

	if err != nil {
		return nil, err
	}

	RabbitDSN := fmt.Sprintf(
		"amqp://%s:%s@%s:%s/%s",
		configuration.RABBIT_LOGIN,
		configuration.RABBIT_PASSWORD,
		configuration.RABBIT_HOST,
		configuration.RABBIT_PORT,
		configuration.RABBIT_VH,
	)

	return amqp.Dial(RabbitDSN)
}

func NewAMQPChannel(conn *amqp.Connection) (*amqp.Channel, error) {
	return conn.Channel()
}

func InitMessages(ch *amqp.Channel) (<-chan amqp.Delivery, error) {
	configuration, err := internal.GetConfig()

	if err != nil {
		return nil, err
	}

	err = ch.ExchangeDeclare(
		configuration.RABBIT_EXCHANGE, // name
		"direct",                      // type
		true,                          // durable
		false,                         // auto-deleted
		false,                         // internal
		false,                         // no-wait
		nil,                           // arguments
	)
	if err != nil {
		return nil, err
	}

	q, err := ch.QueueDeclare(
		configuration.RABBIT_QUEUE, // name
		true,                       // durable
		false,                      // delete when unused
		false,                      // exclusive
		false,                      // no-wait
		nil,                        // arguments
	)
	if err != nil {
		return nil, err
	}

	err = ch.QueueBind(
		q.Name,                           // queue name
		configuration.RABBIT_ROUTING_KEY, // routing key
		configuration.RABBIT_EXCHANGE,    // exchange
		false,
		nil,
	)
	if err != nil {
		return nil, err
	}

	msgs, err := ch.Consume(
		q.Name, // queue
		"",     // consumer
		true,   // auto-ack
		false,  // exclusive
		false,  // no-local
		false,  // no-wait
		nil,    // args
	)
	if err != nil {
		return nil, err
	}

	return msgs, nil
}
