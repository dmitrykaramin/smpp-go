package main

import (
	"SMSRouter/pkg/bus"
	"SMSRouter/pkg/logger"
	"SMSRouter/pkg/smsmessages"
	"context"
	"encoding/json"
	"fmt"
	"github.com/getsentry/sentry-go"
	"golang.org/x/sync/errgroup"
	"log"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"syscall"
	"time"
)

const (
	FlushSentrySec = 4
)

func StartHTTPServer(ctx context.Context, g *errgroup.Group, done context.CancelFunc, smsRepo *smsmessages.SmsRepo) {
	HTTPServer := &http.Server{Addr: ":8080"}
	g.Go(func() error {
		http.HandleFunc("/health", smsRepo.HealthCheck)
		err := HTTPServer.ListenAndServe()

		fmt.Println(err)

		if err != nil {
			fmt.Println("HTTP SERVER CLOSING")
			sentry.CaptureException(err)
			done()
			return err
		}
		return nil
	})

	<-ctx.Done()
	_ = HTTPServer.Shutdown(ctx)
	fmt.Println("HTTP SERVER STOPPED")
}

func CloseConnections(smsrepo *smsmessages.SmsRepo) {
	var err error

	// Close all connections
	err = smsrepo.FluentConn.Close()
	if err != nil {
		sentry.CaptureException(err)
	}

	err = smsrepo.RabbitConn.Close()
	if err != nil {
		sentry.CaptureException(err)
	}

	err = smsrepo.RabbitChannel.Close()
	if err != nil {
		sentry.CaptureException(err)
	}

	err = smsrepo.SMPPTx.Close()
	if err != nil {
		sentry.CaptureException(err)
	}

	if smsrepo.DBConn != nil {
		err = smsrepo.DBConn.Close()
		if err != nil {
			sentry.CaptureException(err)
		}
	}
}

func StartService(ctx context.Context, smsRepo smsmessages.SMSMessenger) func() error {
	return func() error {
		var phoneRegex = regexp.MustCompile(`^7\d{10}$`)

		Message := smsmessages.NewSMSMessage()
		messages, err := bus.InitMessages(smsRepo.NewRabbitChannel())
		if err != nil {
			sentry.CaptureException(err)
			sentry.Flush(time.Second * 5)
			return err
		}

		fmt.Println("Started Server")

		go func() {
			for message := range messages {
				fmt.Printf("MESSAGE RECEIVED %v \n", message)
				sentry.CaptureMessage(fmt.Sprintf("MESSAGE RECEIVED %v \n", message))

				err := json.Unmarshal(message.Body, &Message)
				if err != nil {
					sentry.CaptureException(err)
					continue
				}

				Message.PhoneString = string(Message.Phone)

				sentry.CaptureMessage(fmt.Sprintf("MESSAGE RECEIVED %v, %v", Message.PhoneString, Message.Message))

				if !phoneRegex.MatchString(Message.PhoneString) {
					sentry.CaptureException(fmt.Errorf(
						fmt.Sprintf("wrong phone format: %s", Message.Phone),
					))
					continue
				}
			}
		}()

		//err = smsRepo.SendBySMPP(Message)
		//if err != nil {
		//	sentry.CaptureException(err)
		//	continue
		//}

		<-ctx.Done()
		fmt.Println("Finish Server")
		return nil
	}
}

func ManualCancel(ctx context.Context, cancelFunc context.CancelFunc) func() error {
	return func() error {
		signalChannel := make(chan os.Signal, 1)
		signal.Notify(signalChannel, os.Interrupt, syscall.SIGTERM)

		select {
		case sig := <-signalChannel:
			fmt.Printf("Received signal: %s\n", sig)
			cancelFunc()
		case <-ctx.Done():
			fmt.Printf("closing ManualCancel goroutine\n")
			return ctx.Err()
		}
		return nil
	}
}

func main() {
	var err error

	ctx, done := context.WithCancel(context.Background())
	g, gctx := errgroup.WithContext(ctx)

	SmsRepo := smsmessages.NewSmsRepo()

	if err != nil {
		log.Fatalf("[Config] error: %s", err)
	}

	err = logger.StartSentry()
	if err != nil {
		log.Fatalf("[Sentry] error: %s", err)
	}

	FluentConn, err := logger.StartFluent()
	if err != nil {
		sentry.CaptureException(err)
		log.Fatalf("[Fluent] error: %s", err)
	}
	SmsRepo.FluentConn = FluentConn

	err = SmsRepo.StartInfrastructure()
	if err != nil {
		sentry.CaptureException(err)
		log.Fatalf("[Infrastructure] error: %s", err)
	}

	defer CloseConnections(&SmsRepo)
	defer sentry.Flush(FlushSentrySec * time.Second)

	go StartHTTPServer(ctx, g, done, &SmsRepo)
	g.Go(ManualCancel(gctx, done))
	g.Go(StartService(gctx, &SmsRepo))

	fmt.Println("All started")

	err = g.Wait()
	if err != nil {
		fmt.Printf("Service stopped, reason: %v", err)
	}

	fmt.Println("Service stopped")
}
