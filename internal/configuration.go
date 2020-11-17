package internal

import (
	"fmt"
	"github.com/getsentry/sentry-go"
	"github.com/tkanos/gonfig"
)

type configuration struct {
	RABBIT_LOGIN       string
	RABBIT_PASSWORD    string
	RABBIT_VH          string
	RABBIT_HOST        string
	RABBIT_PORT        string
	RABBIT_ROUTING_KEY string
	RABBIT_EXCHANGE    string
	RABBIT_QUEUE       string

	DB_USERNAME string
	DB_PASSWORD string
	DB_PORT     string
	DB_HOST     string
	DB_NAME     string

	SAVE_MESSAGES_IN_DB bool

	SMS_ROUTER_IP          string
	SMS_ROUTER_PORT        string
	SMS_ROUTER_SYSTEM_ID   string
	SMS_ROUTER_PASSWORD    string
	SMS_ROUTER_SOURCE_ADDR string

	FLUENT_TAG  string
	FLUENT_HOST string
	FLUENT_PORT int

	SENTRY_DSN string
}

var Configuration configuration

func SetConfig(filepath string) error {
	configuration := configuration{}
	err := gonfig.GetConf(filepath, &configuration)
	if err != nil {
		sentry.CaptureException(fmt.Errorf("project is not configured: %s", err))
		return err
	}
	Configuration = configuration

	return nil
}
