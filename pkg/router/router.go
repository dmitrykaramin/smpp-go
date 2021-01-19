package router

import (
	"SMSRouter/internal"
	"fmt"
	"github.com/fiorix/go-smpp/smpp"
	"github.com/fiorix/go-smpp/smpp/pdu/pdufield"
	"github.com/fiorix/go-smpp/smpp/pdu/pdutext"
)

const (
	SourceAddrTON = 5
	SourceAddrNPI = 0
	DestAddrTON   = 1
	DestAddrNPI   = 1
)

func NewTransceiver(f smpp.HandlerFunc) (*smpp.Transceiver, error) {
	configuration, err := internal.GetConfig()
	if err != nil {
		return nil, err
	}

	SMPPRouterAddress := fmt.Sprintf(
		"%s:%s", configuration.SMS_ROUTER_IP, configuration.SMS_ROUTER_PORT,
	)
	tx := &smpp.Transceiver{
		Addr:    SMPPRouterAddress,
		User:    configuration.SMS_ROUTER_SYSTEM_ID,
		Passwd:  configuration.SMS_ROUTER_PASSWORD,
		Handler: f,
	}
	smppConn := tx.Bind()
	go func() {
		for c := range smppConn {
			fmt.Printf("SMPP connection status: %s \n", c.Status())
		}
	}()
	return tx, nil
}

func NewShortMessage(phone, message string) (*smpp.ShortMessage, error) {
	configuration, err := internal.GetConfig()
	if err != nil {
		return nil, err
	}

	return &smpp.ShortMessage{
		Src:           configuration.SMS_ROUTER_SOURCE_ADDR,
		Dst:           phone,
		Text:          pdutext.UCS2(message),
		Register:      pdufield.FinalDeliveryReceipt,
		SourceAddrTON: uint8(SourceAddrTON),
		SourceAddrNPI: uint8(SourceAddrNPI),
		DestAddrTON:   uint8(DestAddrTON),
		DestAddrNPI:   uint8(DestAddrNPI),
	}, err
}
