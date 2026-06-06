package mailer

import (
	"crypto/tls"
	"fmt"
	"net/smtp"
	"strings"
)

type Mailer struct {
	host     string
	port     string
	user     string
	pass     string
	from     string
}

func New(host, port, user, pass, from string) *Mailer {
	return &Mailer{host: host, port: port, user: user, pass: pass, from: from}
}

func (m *Mailer) SendMagicLink(to, link string) error {
	subject := "Your Rize sign-in link"
	body := fmt.Sprintf(
		"Click the link below to sign in to Rize. It expires in 15 minutes.\n\n%s\n\nIf you didn't request this, ignore this email.",
		link,
	)
	return m.send(to, subject, body)
}

func (m *Mailer) send(to, subject, body string) error {
	msg := strings.Join([]string{
		fmt.Sprintf("From: %s", m.from),
		fmt.Sprintf("To: %s", to),
		fmt.Sprintf("Subject: %s", subject),
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"",
		body,
	}, "\r\n")

	addr := fmt.Sprintf("%s:%s", m.host, m.port)
	auth := smtp.PlainAuth("", m.user, m.pass, m.host)

	tlsConfig := &tls.Config{ServerName: m.host, InsecureSkipVerify: false}
	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		// fallback to STARTTLS
		return smtp.SendMail(addr, auth, m.from, []string{to}, []byte(msg))
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, m.host)
	if err != nil {
		return err
	}
	defer client.Quit()

	if err := client.Auth(auth); err != nil {
		return err
	}
	if err := client.Mail(m.from); err != nil {
		return err
	}
	if err := client.Rcpt(to); err != nil {
		return err
	}
	w, err := client.Data()
	if err != nil {
		return err
	}
	_, err = w.Write([]byte(msg))
	if err != nil {
		return err
	}
	return w.Close()
}
