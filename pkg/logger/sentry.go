package logger

import (
	"SMSRouter/internal"
	"github.com/getsentry/sentry-go"
)

func StartSentry() error {
	configuration, err := internal.GetConfig()
	if err != nil {
		return err
	}

	return sentry.Init(sentry.ClientOptions{Dsn: configuration.SENTRY_DSN, Debug: true})
}
