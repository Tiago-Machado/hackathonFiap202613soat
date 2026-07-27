package smtp

import (
	"context"
	"fmt"
	"net/smtp"
)

const (
	fromAddress    = "no-reply@fiapx.local"
	failureSubject = "Falha no processamento do seu vídeo"
)

type Notifier struct {
	address string
}

func NewNotifier(host, port string) *Notifier {
	return &Notifier{address: host + ":" + port}
}

func (n *Notifier) NotifyFailure(ctx context.Context, to, videoName, reason string) error {
	body := fmt.Sprintf("O processamento do vídeo %q falhou.\r\n\r\nMotivo: %s\r\n", videoName, reason)
	message := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\n\r\n%s",
		fromAddress, to, failureSubject, body)
	return smtp.SendMail(n.address, nil, fromAddress, []string{to}, []byte(message))
}
