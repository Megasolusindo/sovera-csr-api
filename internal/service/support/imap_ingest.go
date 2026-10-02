package support

import (
	"context"
	"fmt"
	"io"
	"log"
	"strings"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/emersion/go-message/mail"
	"sovera-core-api/internal/repository"
)

type IMAPConfig struct {
	Server   string
	Port     int
	Username string
	Password string
	UseTLS   bool
}

type IMAPIngester struct {
	config IMAPConfig
	repo   *repository.SupportRepository
}

func NewIMAPIngester(config IMAPConfig, repo *repository.SupportRepository) *IMAPIngester {
	return &IMAPIngester{
		config: config,
		repo:   repo,
	}
}

// Start begins a polling loop to fetch unread emails.
func (i *IMAPIngester) Start(ctx context.Context) {
	log.Printf("[IMAP Ingester] Starting email listener for %s", i.config.Username)

	// Poll every 1 minute
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	// Initial poll
	i.pollUnreadEmails()

	for {
		select {
		case <-ctx.Done():
			log.Println("[IMAP Ingester] Shutting down...")
			return
		case <-ticker.C:
			i.pollUnreadEmails()
		}
	}
}

func (i *IMAPIngester) pollUnreadEmails() {
	addr := fmt.Sprintf("%s:%d", i.config.Server, i.config.Port)
	
	var c *client.Client
	var err error

	if i.config.UseTLS {
		c, err = client.DialTLS(addr, nil)
	} else {
		// Use DialStartTLS if possible, or just Dial
		// For simplicity we will assume TLS is preferred for IMAP (port 993)
		c, err = client.DialTLS(addr, nil)
	}

	if err != nil {
		log.Printf("[IMAP Ingester] Failed to connect to server: %v", err)
		return
	}
	defer c.Logout()

	if err := c.Login(i.config.Username, i.config.Password); err != nil {
		log.Printf("[IMAP Ingester] Failed to login: %v", err)
		return
	}

	// Select INBOX
	_, err = c.Select("INBOX", false)
	if err != nil {
		log.Printf("[IMAP Ingester] Failed to select INBOX: %v", err)
		return
	}

	// Search for UNSEEN messages
	searchCriteria := imap.NewSearchCriteria()
	// Note: Depending on the server, we might just search without FlagRecent and just NOT imap.FlagSeen
	// For simplicity, let's search for NOT SEEN
	searchCriteria.WithoutFlags = []string{imap.SeenFlag}

	searchData, err := c.Search(searchCriteria)
	if err != nil {
		log.Printf("[IMAP Ingester] Failed to search INBOX: %v", err)
		return
	}
	
	seqNums := searchData
	if len(seqNums) == 0 {
		return // No new messages
	}

	log.Printf("[IMAP Ingester] Found %d unread messages.", len(seqNums))

	// Fetch messages
	var seqSet imap.SeqSet
	seqSet.AddNum(seqNums...)

	fetchOptions := []imap.FetchItem{
		imap.FetchEnvelope,
		imap.FetchItem("BODY[]"),
	}

	messages := make(chan *imap.Message, 10)
	done := make(chan error, 1)
	go func() {
		done <- c.Fetch(&seqSet, fetchOptions, messages)
	}()

	for msg := range messages {
		env := msg.Envelope
		var subject, messageID, inReplyTo, references, senderEmail, senderName string
		var textBody, htmlBody string

		if env != nil {
			subject = env.Subject
			messageID = env.MessageId
			inReplyTo = env.InReplyTo
			// Envelope unfortunately does not expose References directly, but we can parse it from the body section header if needed.
			
			if len(env.From) > 0 {
				senderName = env.From[0].PersonalName
				senderEmail = fmt.Sprintf("%s@%s", env.From[0].MailboxName, env.From[0].HostName)
			}
		}

		// Guard: skip messages the server returned without an envelope.
		// These have no usable subject/sender/message-id and would otherwise be
		// ingested as empty tickets, piling up as stale PENDING drafts and firing
		// false-positive SLA alerts. Mark SEEN so they are not re-fetched forever.
		if env == nil {
			log.Printf("[IMAP Ingester] Skipping message seq=%d: no envelope (malformed/empty email)", msg.SeqNum)
			var seqSetSkip imap.SeqSet
			seqSetSkip.AddNum(msg.SeqNum)
			item := imap.FormatFlagsOp(imap.AddFlags, true)
			flags := []interface{}{imap.SeenFlag}
			if err := c.Store(&seqSetSkip, item, flags, nil); err != nil {
				log.Printf("[IMAP Ingester] Failed to mark envelope-less email seq=%d as seen: %v", msg.SeqNum, err)
			}
			continue
		}

		// Fallback for message ID
		if messageID == "" {
			messageID = fmt.Sprintf("generated-%d", time.Now().UnixNano())
		}

		log.Printf("[IMAP Ingester] Processing email: %s (MsgID: %s)", subject, messageID)

		// Read the body section
		var body io.Reader
		for _, r := range msg.Body {
			body = r
			break
		}
		if body != nil {
			mr, err := mail.CreateReader(body)
			if err == nil {
				// Get references from header if available
				if refList := mr.Header.Get("References"); refList != "" {
					references = refList
				}

				// Iterate over parts
				for {
					p, err := mr.NextPart()
					if err == io.EOF {
						break
					} else if err != nil {
						log.Printf("[IMAP Ingester] Warning parsing part: %v", err)
						break
					}

					switch h := p.Header.(type) {
					case *mail.InlineHeader:
						contentType, _, _ := h.ContentType()
						b, _ := io.ReadAll(p.Body)
						
						if strings.HasPrefix(contentType, "text/plain") {
							textBody += string(b)
						} else if strings.HasPrefix(contentType, "text/html") {
							htmlBody += string(b)
						}
					}
				}
			} else {
				log.Printf("[IMAP Ingester] Warning: could not create mail reader: %v", err)
			}
		}

		// Guard: skip messages with no meaningful content. Even with an envelope,
		// an email with empty subject, empty sender, and empty body carries nothing
		// a CS agent can act on; ingesting it only produces an un-reviewable draft.
		if strings.TrimSpace(subject) == "" &&
			strings.TrimSpace(senderEmail) == "" &&
			strings.TrimSpace(textBody) == "" &&
			strings.TrimSpace(htmlBody) == "" {
			log.Printf("[IMAP Ingester] Skipping message seq=%d (MsgID: %s): empty subject, sender, and body", msg.SeqNum, messageID)
			var seqSetSkip imap.SeqSet
			seqSetSkip.AddNum(msg.SeqNum)
			item := imap.FormatFlagsOp(imap.AddFlags, true)
			flags := []interface{}{imap.SeenFlag}
			if err := c.Store(&seqSetSkip, item, flags, nil); err != nil {
				log.Printf("[IMAP Ingester] Failed to mark empty email seq=%d as seen: %v", msg.SeqNum, err)
			}
			continue
		}

		// Save to Database via Repository
		if i.repo != nil {
			err = i.repo.IngestEmail(
				context.Background(),
				messageID,
				inReplyTo,
				references,
				senderEmail,
				senderName,
				subject,
				textBody,
				htmlBody,
			)
			if err != nil {
				log.Printf("[IMAP Ingester] Failed to ingest email to DB: %v", err)
			} else {
				// Mark as SEEN only if successfully ingested
				var seqSetMark imap.SeqSet
				seqSetMark.AddNum(msg.SeqNum)
				item := imap.FormatFlagsOp(imap.AddFlags, true)
				flags := []interface{}{imap.SeenFlag}
				if err := c.Store(&seqSetMark, item, flags, nil); err != nil {
					log.Printf("[IMAP Ingester] Failed to mark email %s as seen: %v", messageID, err)
				}
			}
		}
	}

	if err := <-done; err != nil {
		log.Printf("[IMAP Ingester] Fetch command error: %v", err)
	}
}
