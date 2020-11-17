package main

import (
	"SMSRouter/internal"
	"SMSRouter/pkg/bus"
	"SMSRouter/pkg/db"
	"SMSRouter/pkg/logger"
	"SMSRouter/pkg/router"
	"SMSRouter/pkg/smsmessages"
	"github.com/fiorix/go-smpp/smpp/pdu"
	"github.com/fiorix/go-smpp/smpp/pdu/pdutlv"
	"github.com/getsentry/sentry-go"
	"log"
	"net/http"
)

func StartLoggers(smsrepo *smsmessages.SmsRepo) error {
	// Init Sentry
	err := sentry.Init(sentry.ClientOptions{Dsn: internal.Configuration.SENTRY_DSN, Debug: true})

	if err != nil {
		return err
	}

	// Init Fluent
	fluentConn, err := logger.InitFluent()
	if err != nil {
		return err
	}

	smsrepo.FluentConn = fluentConn

	return nil
}

func StartInfrastructure(smsrepo *smsmessages.SmsRepo) error {
	PDUHandlerFunc := func(repo *smsmessages.SmsRepo) func(pdu.Body) {
		return func(p pdu.Body) {
			if p.Header().ID == pdu.DeliverSMID && internal.Configuration.SAVE_MESSAGES_IN_DB {
				repo.SetDelivered(p.TLVFields()[pdutlv.TagReceiptedMessageID].String())
			}
		}
	}(smsrepo)

	// Init database if needed
	if internal.Configuration.SAVE_MESSAGES_IN_DB {
		dbConn, err := db.InitDB()
		if err != nil {
			return err
		}
		smsrepo.DBConn = dbConn
	}

	// Init RabbitMQ
	amqpConn, err := bus.InitAMQP()
	if err != nil {
		return err
	}
	smsrepo.RabbitConn = amqpConn

	ch, err := bus.InitAMQPChannel(amqpConn)

	if err != nil {
		return err
	}
	smsrepo.RabbitChannel = ch

	// Init SMPP
	transceiver := router.NewTransceiver(PDUHandlerFunc)
	smsrepo.SMPPTx = transceiver

	return nil
}

func StartHTTPServer(smsrepo *smsmessages.SmsRepo) error {
	http.HandleFunc("/health", smsrepo.HealthCheck)
	return http.ListenAndServe(":8080", nil)
}

func CloseConnections(smsrepo *smsmessages.SmsRepo) {
	// Close all connections
	smsrepo.FluentConn.Close()
	smsrepo.RabbitConn.Close()
	smsrepo.RabbitChannel.Close()
	smsrepo.SMPPTx.Close()
	if smsrepo.DBConn != nil {
		smsrepo.DBConn.Close()
	}
}

func main() {
	var err error
	SmsRepo := smsmessages.SmsRepo{}

	err = internal.SetConfig("settings.json")
	if err != nil {
		log.Fatalf("error configuring project: %s", err)
	}

	err = StartLoggers(&SmsRepo)
	if err != nil {
		sentry.CaptureException(err)
		log.Fatalf("logging start up error: %s", err)
	}

	err = StartInfrastructure(&SmsRepo)
	if err != nil {
		sentry.CaptureException(err)
		log.Fatalf("infrastructure start up error: %s", err)
	}

	defer CloseConnections(&SmsRepo)

	go SmsRepo.StartService()

	err = StartHTTPServer(&SmsRepo)
	if err != nil {
		sentry.CaptureException(err)
	}
}
