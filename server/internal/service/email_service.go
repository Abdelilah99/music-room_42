package service

import (
	"fmt"
	"net/smtp"
)

type EmailService struct {
	host     string
	port     string
	from     string
	user     string
	password string
}

func NewEmailService(host, port, from, user, password string) *EmailService {
	return &EmailService{host: host, port: port, from: from, user: user, password: password}
}

func (s *EmailService) Send(to, subject, body string) error {
	addr := fmt.Sprintf("%s:%s", s.host, s.port)
	msg := []byte(
		"From: " + s.from + "\r\n" +
			"To: " + to + "\r\n" +
			"Subject: " + subject + "\r\n" +
			"Content-Type: text/plain; charset=UTF-8\r\n" +
			"\r\n" + body,
	)

	var auth smtp.Auth
	if s.user != "" && s.password != "" {
		auth = smtp.PlainAuth("", s.user, s.password, s.host)
	}

	return smtp.SendMail(addr, auth, s.from, []string{to}, msg)
}
