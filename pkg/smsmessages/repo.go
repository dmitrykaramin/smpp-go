package smsmessages

import (
	"SMSRouter/internal"
	"SMSRouter/pkg/bus"
	"SMSRouter/pkg/db"
	"SMSRouter/pkg/logger"
	"SMSRouter/pkg/router"
	"context"
	"encoding/json"
	"github.com/fiorix/go-smpp/smpp"
	"github.com/fiorix/go-smpp/smpp/pdu"
	"github.com/fiorix/go-smpp/smpp/pdu/pdutlv"
	"github.com/fluent/fluent-logger-golang/fluent"
	"github.com/getsentry/sentry-go"
	"github.com/isayme/go-amqp-reconnect/rabbitmq"
	"github.com/jmoiron/sqlx"
	"github.com/streadway/amqp"
	"log"
	"net/http"
	"time"
)

type SMSMessage struct {
	MessageID       string      `db:"message_id" `
	MessageSequence int         `db:"message_sequence"`
	Phone           json.Number `json:"phone"`
	PhoneString     string      `db:"phone"`
	Message         string      `db:"message"`
	IsSent          bool        `db:"is_sent"`
	IsDelivered     bool        `db:"date_created"`
	DateCreated     time.Time   `db:"date_created"`
}

type SmsRepo struct {
	DBConn        *sqlx.DB
	SMPPTx        *smpp.Transceiver
	FluentConn    *fluent.Fluent
	RabbitMes     <-chan amqp.Delivery
	RabbitConn    *rabbitmq.Connection
	RabbitChannel *rabbitmq.Channel
	Context       context.Context
	CancelFunc    context.CancelFunc
}

type SMSMessenger interface {
	SaveInDB(message SMSMessage)
	SendBySMPP(message SMSMessage) error
	SetDelivered(messageID string)
	LogMessage(message SMSMessage) error
	GetRabbitChannel() *rabbitmq.Channel
	GetFluentConn() *fluent.Fluent
}

func NewSMSMessage() SMSMessage {
	return SMSMessage{
		IsSent:      true,
		IsDelivered: false,
		DateCreated: time.Now(), // UTC
	}
}

func NewSmsRepo() SmsRepo {
	return SmsRepo{}
}

func (m *SmsRepo) LogMessage(message SMSMessage) error {
	configuration, err := internal.GetConfig()

	if err != nil {
		return err
	}

	logger.MessageMap["phone"] = message.PhoneString
	logger.MessageMap["message"] = message.Message

	err = m.FluentConn.Post(configuration.FLUENT_TAG, logger.MessageMap)
	if err != nil {
		sentry.CaptureException(err)
	}

	return nil
}

func (m *SmsRepo) SaveInDB(message SMSMessage) {
	if m.DBConn == nil {
		return
	}

	_, err := m.DBConn.Exec(
		"INSERT INTO sms_sms (phone, message, message_id, date_created, is_sent, is_delivered) "+
			"VALUES ($1, $2, $3, $4, $5, $6)",
		message.PhoneString, message.Message, message.MessageID, message.DateCreated, message.IsSent,
		message.IsDelivered,
	)

	if err != nil {
		log.Fatal(err)
	}
}

func (m *SmsRepo) SetDelivered(messageID string) {
	if m.DBConn == nil {
		return
	}

	_, err := m.DBConn.Exec("UPDATE sms_sms SET is_delivered=True WHERE message_id=$1", messageID)

	if err != nil {
		sentry.CaptureException(err)
	}
}

func (m *SmsRepo) SendBySMPP(message SMSMessage) error {
	m.LogMessage(message)
	shortMessage, err := router.NewShortMessage(message.PhoneString, message.Message)
	if err != nil {
		return err
	}
	sm, err := m.SMPPTx.Submit(shortMessage)

	if err != nil {
		return err
	}

	message.MessageID = sm.RespID()
	m.SaveInDB(message)

	log.Printf("Received a message: %s", sm.RespID())

	return nil
}

func (m *SmsRepo) HealthCheck(w http.ResponseWriter, req *http.Request) {
	conn, err := smpp.Dial(m.SMPPTx.Addr, m.SMPPTx.TLS)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	conn.Close()
}

func (m *SmsRepo) StartInfrastructure() error {
	configuration, err := internal.GetConfig()

	if err != nil {
		return err
	}

	PDUHandlerFunc := func(repo *SmsRepo) func(pdu.Body) {
		return func(p pdu.Body) {
			if p.Header().ID == pdu.DeliverSMID && configuration.SAVE_MESSAGES_IN_DB {
				repo.SetDelivered(p.TLVFields()[pdutlv.TagReceiptedMessageID].String())
			}
		}
	}(m)

	// Init database if needed
	if configuration.SAVE_MESSAGES_IN_DB {
		dbConn, err := db.NewDBConn()
		if err != nil {
			return err
		}
		m.DBConn = dbConn
	}

	// Init RabbitMQ
	amqpConn, err := bus.InitAMQP()
	if err != nil {
		return err
	}
	m.RabbitConn = amqpConn

	ch, err := bus.NewAMQPChannel(amqpConn)
	if err != nil {
		return err
	}
	m.RabbitChannel = ch

	// Init SMPP
	transceiver, err := router.NewTransceiver(PDUHandlerFunc)
	if err != nil {
		return err
	}
	m.SMPPTx = transceiver

	return nil
}

func (m *SmsRepo) GetRabbitChannel() *rabbitmq.Channel {
	return m.RabbitChannel
}

func (m *SmsRepo) GetFluentConn() *fluent.Fluent {
	return m.FluentConn
}
