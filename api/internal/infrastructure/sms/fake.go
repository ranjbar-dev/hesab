package sms

import (
	"context"
	"log"
)

// FakeSender logs the message instead of sending it. Every code it "delivers" is FixedCode.
// TODO: implement the real sms.ir provider (https://sms.ir) — HTTP client, API key + line number from config, template send. Swap this out in main.go.
const FixedCode = "123456"

type FakeSender struct{ Log *log.Logger }

func (f FakeSender) Send(_ context.Context, phone, message string) error {
	f.Log.Printf("fake SMS to %s: %s", phone, message)
	return nil
}
