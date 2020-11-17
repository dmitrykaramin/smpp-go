package logger

import (
	"SMSRouter/internal"
	"github.com/fluent/fluent-logger-golang/fluent"
)

var MessageMap = make(map[string]string)

func InitFluent() (*fluent.Fluent, error) {
	return fluent.New(
		fluent.Config{
			FluentPort: internal.Configuration.FLUENT_PORT,
			FluentHost: internal.Configuration.FLUENT_HOST,
		},
	)
}

//
//forever := make(chan bool)
//log.Printf(" [*] Waiting for messages. To exit press CTRL+C")
//<-forever
//data := make(map[string]string, 0)

//sms := smsMessages.NewSMSMessage()
//fluentConn := logger.InitFluent()

//https://github.com/tkanos/gonfig

//defer fluentConn.Close()
//tag := "myapp.access"
//sms.Phone = 1234567890
//sms.MessageTemplate = "sms message"
//
//for i := 0; i < 100; i++ {
//	e := fluentConn.Post(tag, sms)
//	if e != nil {
//		log.Println("Error while posting log: ", e)
//	} else {
//		log.Println("Success to post log")
//	}
//	time.Sleep(1000 * time.Millisecond)
//}
