package email

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"strings"
	"time"
	"crypto/sha256"
	"encoding/hex"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/emersion/go-message/mail"

	"kwork-assistant/internal/domain"
)

type Config struct {
	Host          string
	Port          int
	Username      string
	Password      string
	UseTLS        bool
	Folder        string
	SenderFilter  string
	SubjectFilter string
	LookbackHours int
}

type IMAPClient struct {
	cfg Config
}

func NewIMAPClient(cfg Config) *IMAPClient {
	if cfg.Folder == "" {
		cfg.Folder = "INBOX"
	}
	if cfg.LookbackHours == 0 {
		cfg.LookbackHours = 48
	}
	return &IMAPClient{cfg: cfg}
}

func (c *IMAPClient) Health() error {
	addr := fmt.Sprintf("%s:%d", c.cfg.Host, c.cfg.Port)
	var cl *client.Client
	var err error

	if c.cfg.UseTLS {
		cl, err = client.DialTLS(addr, &tls.Config{ServerName: c.cfg.Host})
	} else {
		cl, err = client.Dial(addr)
	}
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer cl.Logout()

	if err := cl.Login(c.cfg.Username, c.cfg.Password); err != nil {
		return fmt.Errorf("failed to login: %w", err)
	}
	
	_, err = cl.Select(c.cfg.Folder, true)
	if err != nil {
		return fmt.Errorf("failed to select folder %s: %w", c.cfg.Folder, err)
	}
	return nil
}

func (c *IMAPClient) FetchRecentEmails(ctx context.Context) ([]domain.InboundEmail, error) {
	addr := fmt.Sprintf("%s:%d", c.cfg.Host, c.cfg.Port)
	var cl *client.Client
	var err error

	if c.cfg.UseTLS {
		cl, err = client.DialTLS(addr, &tls.Config{ServerName: c.cfg.Host})
	} else {
		cl, err = client.Dial(addr)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}
	defer cl.Logout()

	if err := cl.Login(c.cfg.Username, c.cfg.Password); err != nil {
		return nil, fmt.Errorf("failed to login: %w", err)
	}
	
	mbox, err := cl.Select(c.cfg.Folder, true)
	if err != nil {
		return nil, fmt.Errorf("failed to select folder: %w", err)
	}

	if mbox.Messages == 0 {
		return nil, nil
	}

	since := time.Now().Add(-time.Duration(c.cfg.LookbackHours) * time.Hour)

	criteria := imap.NewSearchCriteria()
	criteria.Since = since
	
	uids, err := cl.UidSearch(criteria)
	if err != nil {
		return nil, fmt.Errorf("failed to search: %w", err)
	}

	if len(uids) == 0 {
		return nil, nil
	}

	seqSet := new(imap.SeqSet)
	seqSet.AddNum(uids...)

	section := &imap.BodySectionName{}
	items := []imap.FetchItem{imap.FetchEnvelope, imap.FetchUid, section.FetchItem()}

	messages := make(chan *imap.Message, len(uids))
	done := make(chan error, 1)

	go func() {
		done <- cl.UidFetch(seqSet, items, messages)
	}()

	var emails []domain.InboundEmail

	for msg := range messages {
		// Apply filters
		sender := ""
		if msg.Envelope != nil && len(msg.Envelope.From) > 0 {
			sender = msg.Envelope.From[0].Address()
		}
		subject := ""
		if msg.Envelope != nil {
			subject = msg.Envelope.Subject
		}

		if c.cfg.SenderFilter != "" && !strings.Contains(strings.ToLower(sender), strings.ToLower(c.cfg.SenderFilter)) {
			continue
		}
		if c.cfg.SubjectFilter != "" && !strings.Contains(strings.ToLower(subject), strings.ToLower(c.cfg.SubjectFilter)) {
			continue
		}

		body := msg.GetBody(section)
		if body == nil {
			continue
		}

		mr, err := mail.CreateReader(body)
		if err != nil {
			continue
		}

		var textBody, htmlBody string
		
		for {
			p, err := mr.NextPart()
			if err == io.EOF {
				break
			} else if err != nil {
				break
			}

			switch h := p.Header.(type) {
			case *mail.InlineHeader:
				contentType, _, _ := h.ContentType()
				b, _ := io.ReadAll(p.Body)
				if contentType == "text/plain" {
					textBody += string(b)
				} else if contentType == "text/html" {
					htmlBody += string(b)
				}
			}
		}
		
		msgID := ""
		if msg.Envelope != nil {
			msgID = msg.Envelope.MessageId
		}
		
		receivedAt := time.Now()
		if msg.Envelope != nil && !msg.Envelope.Date.IsZero() {
			receivedAt = msg.Envelope.Date
		}

		rawHash := hashString(textBody + htmlBody + subject + sender)
		
		// Fallback for MessageID
		if msgID == "" {
			msgID = fmt.Sprintf("UID-%d-%s", msg.Uid, rawHash)
		}

		emails = append(emails, domain.InboundEmail{
			ProviderUID:      fmt.Sprintf("%d", msg.Uid),
			MessageID:        msgID,
			Sender:           sender,
			Subject:          subject,
			TextBody:         textBody,
			HTMLBody:         htmlBody,
			ReceivedAt:       receivedAt,
			RawHash:          rawHash,
			ProcessingStatus: domain.ProcessingStatusNew,
		})
	}

	if err := <-done; err != nil {
		return nil, fmt.Errorf("failed to fetch messages: %w", err)
	}

	return emails, nil
}

func hashString(s string) string {
	h := sha256.New()
	h.Write([]byte(s))
	return hex.EncodeToString(h.Sum(nil))
}
