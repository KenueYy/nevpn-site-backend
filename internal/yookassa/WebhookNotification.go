package yookassa

type WebhookNotification struct {
	Type  string
	Event string

	Object YooKassaPayment
}

type YooKassaPayment struct {
	ID       string
	Status   string
	Amount   Amount
	Metadata map[string]string
	Paid     bool
}
