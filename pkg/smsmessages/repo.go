package smsmessages

import (
	"SMSRouter/internal"
	"SMSRouter/pkg/bus"
	"SMSRouter/pkg/logger"
	"SMSRouter/pkg/router"
	"encoding/json"
	"fmt"
	"github.com/fiorix/go-smpp/smpp"
	"github.com/fluent/fluent-logger-golang/fluent"
	"github.com/getsentry/sentry-go"
	"github.com/jmoiron/sqlx"
	"github.com/streadway/amqp"
	"log"
	"net/http"
	"regexp"
	"time"
)

type SMSMessage struct {
	MessageID       string      `db:"message_id" `
	MessageSequence int         `db:"message_sequence"`
	Phone           json.Number `db:"phone" json:"phone"`
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
	RabbitConn    *amqp.Connection
	RabbitChannel *amqp.Channel
}

type SMSMessanger interface {
	SaveInDB(message SMSMessage)
	SendBySMPP(phone, message string)
}

func NewSMSMessage() SMSMessage {
	return SMSMessage{
		IsSent:      true,
		IsDelivered: false,
		DateCreated: time.Now(), // UTC
	}
}

func (m *SmsRepo) SaveInDB(message SMSMessage) {
	if m.DBConn == nil {
		return
	}

	_, err := m.DBConn.Exec(
		"INSERT INTO sms_sms (phone, message, message_id, date_created, is_sent, is_delivered) "+
			"VALUES ($1, $2, $3, $4, $5, $6)",
		string(message.Phone), message.Message, message.MessageID, message.DateCreated, message.IsSent,
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

func (m *SmsRepo) SendBySMPP(phone, message string) (*smpp.ShortMessage, error) {
	sm, err := m.SMPPTx.Submit(router.NewShortMessage(phone, message))

	if err != nil {
		sentry.CaptureException(err)
		return nil, err
	}

	logger.MessageMap["phone"] = phone
	logger.MessageMap["message"] = message

	return sm, nil
}

func (m *SmsRepo) StartService() {
	var phoneRegex = regexp.MustCompile(`^7\d{10}$`)
	MessageTemplate := NewSMSMessage()
	messages, err := bus.InitMessages(m.RabbitChannel)

	if err != nil {
		sentry.CaptureException(err)
		return
	}

	for message := range messages {
		err := json.Unmarshal(message.Body, &MessageTemplate)
		if err != nil {
			sentry.CaptureException(err)
			continue
		}

		phoneString := string(MessageTemplate.Phone)

		if !phoneRegex.MatchString(phoneString) {
			sentry.CaptureException(fmt.Errorf(
				fmt.Sprintf("wrong phone format: %s", MessageTemplate.Phone),
			))
			continue
		}

		logger.MessageMap["phone"] = phoneString
		logger.MessageMap["message"] = MessageTemplate.Message
		m.FluentConn.Post(internal.Configuration.FLUENT_TAG, logger.MessageMap)

		sm, err := m.SendBySMPP(phoneString, MessageTemplate.Message)

		if err != nil {
			sentry.CaptureException(err)
			continue
		}

		MessageTemplate.MessageID = sm.RespID()
		m.SaveInDB(MessageTemplate)

		log.Printf("Received a message: %s", sm.RespID())
	}
}

func (m *SmsRepo) HealthCheck(w http.ResponseWriter, req *http.Request) {
	conn, err := smpp.Dial(m.SMPPTx.Addr, m.SMPPTx.TLS)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	conn.Close()
}
