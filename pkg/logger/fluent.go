package logger

import (
	"SMSRouter/internal"
	"github.com/fluent/fluent-logger-golang/fluent"
)

var MessageMap = make(map[string]string)

func StartFluent() (*fluent.Fluent, error) {
	configuration, err := internal.GetConfig()

	if err != nil {
		return nil, err
	}

	return fluent.New(
		fluent.Config{
			FluentPort: configuration.FLUENT_PORT,
			FluentHost: configuration.FLUENT_HOST,
		},
	)
}
