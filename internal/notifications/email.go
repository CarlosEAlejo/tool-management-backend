package notifications

type VerificationEmailSender interface {
	SendVerificationCode(email string, code string) error
}

type NoopVerificationEmailSender struct{}

func (NoopVerificationEmailSender) SendVerificationCode(email string, code string) error {
	return nil
}
