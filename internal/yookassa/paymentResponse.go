package yookassa

import (
	"time"
)

type PaymentResponse struct {
	ID           string
	Status       string
	Amount       Amount
	Confirmation Confirmation
	Test         bool
	Metadata     map[string]string
	Description  string
}

type CheckPaymentResponse struct {
	ID             string
	Status         string
	Amount         Amount
	IncomeAmount   Amount
	Description    string
	Recipient      Recipient
	PaymentMethod  PaymentMethod
	CapturedAt     time.Time
	CreatedAt      time.Time
	Test           bool
	RefundedAmount Amount
	Paid           bool
	Refundable     bool
	Metadata       map[string]string
}

type Recipient struct {
	AccountId string
	GatewayId string
}

type PaymentMethod struct {
	Type          string
	ID            string
	Saved         bool
	Status        string
	Title         string
	AccountNumber string
}
