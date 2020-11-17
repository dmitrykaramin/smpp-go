package bus

import (
	"SMSRouter/internal"
	"fmt"
	"github.com/streadway/amqp"
)

func InitAMQP() (*amqp.Connection, error) {
	RabbitDSN := fmt.Sprintf(
		"amqp://%s:%s@%s:%s/%s",
		internal.Configuration.RABBIT_LOGIN,
		internal.Configuration.RABBIT_PASSWORD,
		internal.Configuration.RABBIT_HOST,
		internal.Configuration.RABBIT_PORT,
		internal.Configuration.RABBIT_VH,
	)

	return amqp.Dial(RabbitDSN)
}

func InitAMQPChannel(conn *amqp.Connection) (*amqp.Channel, error) {
	return conn.Channel()
}

func InitMessages(ch *amqp.Channel) (<-chan amqp.Delivery, error) {
	err := ch.ExchangeDeclare(
		internal.Configuration.RABBIT_EXCHANGE, // name
		"direct",                               // type
		true,                                   // durable
		false,                                  // auto-deleted
		false,                                  // internal
		false,                                  // no-wait
		nil,                                    // arguments
	)
	if err != nil {
		return nil, err
	}

	q, err := ch.QueueDeclare(
		internal.Configuration.RABBIT_QUEUE, // name
		true,                                // durable
		false,                               // delete when unused
		false,                               // exclusive
		false,                               // no-wait
		nil,                                 // arguments
	)
	if err != nil {
		return nil, err
	}

	err = ch.QueueBind(
		q.Name, // queue name
		internal.Configuration.RABBIT_ROUTING_KEY, // routing key
		internal.Configuration.RABBIT_EXCHANGE,    // exchange
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
