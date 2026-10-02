package handler

import (
	"log"
	"time"
	
	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	
	"sovera-core-api/internal/service/support"
)

type SupportHandler struct {
	pool *pgxpool.Pool
	smtp *support.SmtpOutbound
}

func NewSupportHandler(pool *pgxpool.Pool, smtpSvc *support.SmtpOutbound) *SupportHandler {
	return &SupportHandler{pool: pool, smtp: smtpSvc}
}

func (h *SupportHandler) KBSearch(c *fiber.Ctx) error {
	// Dummy implementation for Phase 1
	query := c.Query("query")
	
	// TODO: implement pgvector semantic search
	return c.JSON(fiber.Map{
		"results": []fiber.Map{
			{
				"title": "Panduan Paket Berlangganan",
				"content": "Kami memiliki paket Basic, Pro, dan Enterprise. Harga mulai dari Rp 1.500.000/bulan.",
				"confidence": 0.95,
			},
		},
		"query": query,
	})
}

func (h *SupportHandler) GetAccountStatus(c *fiber.Ctx) error {
	tenantID := c.Query("tenant_id")
	
	return c.JSON(fiber.Map{
		"tenant_id": tenantID,
		"plan": "PRO",
		"status": "ACTIVE",
		"payment_status": "PAID",
	})
}

func (h *SupportHandler) GetTicketHistory(c *fiber.Ctx) error {
	email := c.Query("email")
	
	return c.JSON(fiber.Map{
		"email": email,
		"history": []fiber.Map{},
	})
}

func (h *SupportHandler) GetPendingTickets(c *fiber.Ctx) error {
	// Returns tickets that need AI processing
	rows, err := h.pool.Query(c.Context(), `
		SELECT t.id, t.status, e.id, e.subject, e.body_text, e.sender_email
		FROM support.support_tickets t
		JOIN support.support_emails e ON e.ticket_id = t.id
		WHERE t.status = 'OPEN' AND NOT EXISTS (
			SELECT 1 FROM support.support_drafts d WHERE d.ticket_id = t.id AND d.status != 'REJECTED'
		)
	`)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	defer rows.Close()

	var tickets []fiber.Map
	for rows.Next() {
		var tID, status, emailID, subject, body, sender string
		if err := rows.Scan(&tID, &status, &emailID, &subject, &body, &sender); err == nil {
			tickets = append(tickets, fiber.Map{
				"ticket_id":    tID,
				"status":       status,
				"email_id":     emailID,
				"subject":      subject,
				"body":         body,
				"sender_email": sender,
			})
		}
	}

	return c.JSON(fiber.Map{"tickets": tickets})
}

func (h *SupportHandler) GetStaleDrafts(c *fiber.Ctx) error {
	hours := c.QueryInt("hours", 4)
	
	rows, err := h.pool.Query(c.Context(), `
		SELECT d.ticket_id, e.subject, d.created_at
		FROM support.support_drafts d
		JOIN support.support_emails e ON e.ticket_id = d.ticket_id
		WHERE d.status = 'PENDING' AND d.created_at < NOW() - INTERVAL '1 hour' * $1
	`, hours)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	defer rows.Close()

	var drafts []fiber.Map
	for rows.Next() {
		var tID, subject string
		var createdAt time.Time
		if err := rows.Scan(&tID, &subject, &createdAt); err == nil {
			drafts = append(drafts, fiber.Map{
				"ticket_id":  tID,
				"subject":    subject,
				"created_at": createdAt,
			})
		}
	}

	return c.JSON(fiber.Map{"drafts": drafts})
}

type SaveDraftRequest struct {
	TicketID   string  `json:"ticket_id"`
	Text       string  `json:"text"`
	Sources    []string`json:"sources"`
	Confidence float64 `json:"confidence"`
}

func (h *SupportHandler) SaveDraft(c *fiber.Ctx) error {
	var req SaveDraftRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	// Insert into support_drafts
	_, err := h.pool.Exec(c.Context(), `
		INSERT INTO support.support_drafts (ticket_id, draft_text, sources_used, confidence, gate_decision, status)
		VALUES ($1, $2, $3, $4, 'NEEDS_REVIEW', 'PENDING')
		ON CONFLICT (ticket_id) DO UPDATE 
		SET draft_text = EXCLUDED.draft_text, confidence = EXCLUDED.confidence, gate_decision = 'NEEDS_REVIEW', status = 'PENDING'
	`, req.TicketID, req.Text, "[]", req.Confidence)
	
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	// Update support_emails status
	h.pool.Exec(c.Context(), `UPDATE support.support_emails SET status = 'IN_REVIEW' WHERE ticket_id = $1`, req.TicketID)

	// TODO: Send to Telegram Review Group

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Draft saved and sent for review.",
	})
}

type UpdateDraftRequest struct {
	TicketID string `json:"ticket_id"`
	Decision string `json:"decision"` // APPROVED, REJECTED
}

func (h *SupportHandler) UpdateDraft(c *fiber.Ctx) error {
	var req UpdateDraftRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	_, err := h.pool.Exec(c.Context(), `
		UPDATE support.support_drafts
		SET gate_decision = $1::text, status = CASE WHEN $1::text = 'APPROVED' THEN 'SENT' ELSE 'PENDING' END
		WHERE ticket_id = $2
	`, req.Decision, req.TicketID)
	
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	
	// TODO: actually send the email if APPROVED
	if req.Decision == "APPROVED" && h.smtp != nil {
		// Fetch draft & email info
		var toEmail, subject, bodyText string
		var inReplyTo, references *string
		err := h.pool.QueryRow(c.Context(), `
			SELECT e.sender_email, e.subject, d.draft_text, e.message_id, e.references_header
			FROM support.support_drafts d
			JOIN support.support_emails e ON e.ticket_id = d.ticket_id
			WHERE d.ticket_id = $1
		`, req.TicketID).Scan(&toEmail, &subject, &bodyText, &inReplyTo, &references)
		
		if err == nil {
			go func() {
				var ir, ref string
				if inReplyTo != nil { ir = *inReplyTo }
				if references != nil { ref = *references }
				err := h.smtp.SendReply(toEmail, subject, bodyText, ir, ref)
				if err != nil {
					log.Printf("[Support] Failed to send email via SMTP: %v", err)
				} else {
					log.Printf("[Support] Successfully sent reply email to %s", toEmail)
				}
			}()
		} else {
			log.Printf("[Support] Failed to fetch draft for email reply: %v", err)
		}
	}

	return c.JSON(fiber.Map{
		"success": true,
	})
}

type EditDraftRequest struct {
	TicketID string `json:"ticket_id"`
	Text     string `json:"text"`
}

// EditDraft replaces the AI draft reply text with a human-revised version.
// The draft stays PENDING so it can be reviewed/approved again afterwards.
func (h *SupportHandler) EditDraft(c *fiber.Ctx) error {
	var req EditDraftRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}
	if req.TicketID == "" || req.Text == "" {
		return c.Status(400).JSON(fiber.Map{"error": "ticket_id and text are required"})
	}

	ct, err := h.pool.Exec(c.Context(), `
		UPDATE support.support_drafts
		SET draft_text = $1::text,
		    final_text = $1::text,
		    gate_decision = 'EDITED',
		    status = 'PENDING',
		    awaiting_edit_by = NULL,
		    updated_at = NOW()
		WHERE ticket_id = $2
	`, req.Text, req.TicketID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	if ct.RowsAffected() == 0 {
		return c.Status(404).JSON(fiber.Map{"error": "draft not found for ticket_id"})
	}

	return c.JSON(fiber.Map{
		"success": true,
	})
}
