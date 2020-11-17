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

func NewTransceiver(f smpp.HandlerFunc) *smpp.Transceiver {
	SMPPRouterAddress := fmt.Sprintf(
		"%s:%s", internal.Configuration.SMS_ROUTER_IP, internal.Configuration.SMS_ROUTER_PORT,
	)
	tx := &smpp.Transceiver{
		Addr:    SMPPRouterAddress,
		User:    internal.Configuration.SMS_ROUTER_SYSTEM_ID,
		Passwd:  internal.Configuration.SMS_ROUTER_PASSWORD,
		Handler: f,
	}
	smppConn := tx.Bind()
	go func() {
		for c := range smppConn {
			fmt.Printf("SMPP connection status: %s", c.Status())
		}
	}()
	return tx
}

func NewShortMessage(phone, message string) *smpp.ShortMessage {
	return &smpp.ShortMessage{
		Src:           internal.Configuration.SMS_ROUTER_SOURCE_ADDR,
		Dst:           phone,
		Text:          pdutext.UCS2(message),
		Register:      pdufield.FinalDeliveryReceipt,
		SourceAddrTON: uint8(SourceAddrTON),
		SourceAddrNPI: uint8(SourceAddrNPI),
		DestAddrTON:   uint8(DestAddrTON),
		DestAddrNPI:   uint8(DestAddrNPI),
	}
}
