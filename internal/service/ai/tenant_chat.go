package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"sovera-core-api/internal/model"
)

type OrganizationAIRepositoryInterface interface {
	CreateConversation(ctx context.Context, orgID, userID uuid.UUID, title string) (*model.OrganizationAIConversation, error)
	GetConversationByID(ctx context.Context, orgID, userID, conversationID uuid.UUID) (*model.OrganizationAIConversation, error)
	ListConversationsByUser(ctx context.Context, orgID, userID uuid.UUID) ([]model.OrganizationAIConversation, error)
	ListConversationsByOrgAdmin(ctx context.Context, orgID uuid.UUID) ([]model.OrganizationAIConversation, error)
	ArchiveConversation(ctx context.Context, orgID, userID, conversationID uuid.UUID) error
	CreateChatLog(ctx context.Context, log model.OrganizationAIChatLog) error
	GetConversationHistory(ctx context.Context, orgID, userID, conversationID uuid.UUID) ([]model.OrganizationAIChatLog, error)
	GetAdminConversationHistory(ctx context.Context, orgID, conversationID uuid.UUID) ([]model.OrganizationAIChatLog, error)
	SearchSignalsByOrg(ctx context.Context, orgID uuid.UUID, queryStr string) ([]model.CorporateSignal, error)
	ListWatchlistByOrg(ctx context.Context, orgID uuid.UUID) ([]model.AICompanyWatchlist, error)
	SearchCompaniesPublic(ctx context.Context, queryStr string) ([]model.Company, error)
	ListOrganizationPrograms(ctx context.Context, orgID uuid.UUID) ([]model.OrganizationProgramInfo, error)
}

type OrganizationAIChatService struct {
	apiKey     string
	repo       OrganizationAIRepositoryInterface
	httpClient *http.Client
}

func NewOrganizationAIChatService(apiKey string, repo OrganizationAIRepositoryInterface) *OrganizationAIChatService {
	return &OrganizationAIChatService{
		apiKey:     apiKey,
		repo:       repo,
		httpClient: &http.Client{Timeout: 45 * time.Second},
	}
}

// ProcessChatMessage handles end-to-end multi-turn chat with guardrails, conversation history, and closure tool-calling
func (s *OrganizationAIChatService) ProcessChatMessage(ctx context.Context, orgID, userID uuid.UUID, role, message string, conversationID *uuid.UUID) (*model.OrganizationAIChatResponse, error) {
	startTime := time.Now()

	// 1. Resolve or create conversation thread
	var conv *model.OrganizationAIConversation
	var err error

	if conversationID != nil && *conversationID != uuid.Nil {
		conv, err = s.repo.GetConversationByID(ctx, orgID, userID, *conversationID)
		if err != nil {
			// Fallback: create new conversation if requested one does not exist or isn't owned by user
			title := message
			if len(title) > 40 {
				title = title[:40] + "..."
			}
			conv, err = s.repo.CreateConversation(ctx, orgID, userID, title)
			if err != nil {
				return nil, fmt.Errorf("failed to create new conversation: %w", err)
			}
		}
	} else {
		title := message
		if len(title) > 40 {
			title = title[:40] + "..."
		}
		conv, err = s.repo.CreateConversation(ctx, orgID, userID, title)
		if err != nil {
			return nil, fmt.Errorf("failed to create conversation thread: %w", err)
		}
	}

	// 2. Fetch conversation history for LLM context window
	history, _ := s.repo.GetConversationHistory(ctx, orgID, userID, conv.ID)

	// 3. Build system prompt & Gemini contents payload
	systemPrompt := fmt.Sprintf(`Kamu adalah Asisten CSR Intelligence untuk organisasi ID: %s (Role User: %s).

DOMAIN & TANGGUNG JAWAB:
1. Kamu HANYA menjawab pertanyaan dalam domain CSR (Corporate Social Responsibility), TJSL (Tanggung Jawab Sosial dan Lingkungan), ESG (Environmental, Social, Governance), keberlanjutan (sustainability), dan data CSR Intelligence platform.
2. Jika pertanyaan di luar domain CSR/ESG/TJSL (misal: olahraga, politik, resep makanan, hiburan), TOLAK DENGAN SOPAN.

TOOL TERSEDIA & PANDUAN PENGGUNAAN:
1. list_organization_programs: PANGGIL TOOL INI KETIKA USER MENANYAKAN PROGRAM ORGANISASI/LEMBAGA (misal: "Berapa program yang saya miliki?", "Apa saja program kami?", "Daftar program organisasi saya"). Tool ini sudah tersedia dan WAJIB dipanggil.
2. list_watchlist: Panggil tool ini ketika user meminta daftar watchlist / perusahaan pemantauan aktif organisasi.
3. search_signals: Panggil tool ini ketika user mencari sinyal CSR, alokasi budget CSR, atau berita CSR perusahaan.
4. search_companies: Panggil tool ini ketika user mencari profil perusahaan publik BEI / BUMN di master database.

ATURAN WAJIB SUMBER DATA & FORMAT HUMANIS:
- Jika user menanyakan tentang program organisasi milik mereka (misal "Berapa program yang saya miliki?"), KAMU WAJIB MEMANGGIL TOOL list_organization_programs. JANGAN PERNAH MENJAWAB SEBELUM MEMANGGIL TOOL TERSEBUT. JANGAN PERNAH MENGATAKAN BAHWA FITUR/TOOL BELUM TERHUBUNG.
- JANGAN PERNAH menampilkan string/JSON mentah (seperti [{"id":"..."}]) kepada user.
- Sampaikan hasil dari tool dengan bahasa Indonesia yang ramah, santun, terstruktur rapi, dan sebutkan jumlah total program beserta judul dan deskripsinya secara humanis.
- Untuk setiap perusahaan yang ditemukan di database, WAJIB sertakan tombol/link aksi ke halaman detail perusahaan menggunakan format Markdown link persis seperti ini: ` + "`[🔍 Lihat Detail](/corporates?search=NamaPerusahaan)`" + `.`, orgID.String(), role)


	// Build contents list for Gemini API
	contents := []map[string]interface{}{}

	// Inject history (prior turns)
	for _, logItem := range history {
		contents = append(contents, map[string]interface{}{
			"role":  "user",
			"parts": []map[string]interface{}{{"text": logItem.Message}},
		})
		contents = append(contents, map[string]interface{}{
			"role":  "model",
			"parts": []map[string]interface{}{{"text": logItem.Reply}},
		})
	}

	// Append current user message
	contents = append(contents, map[string]interface{}{
		"role":  "user",
		"parts": []map[string]interface{}{{"text": message}},
	})

	// 4. Define tool declarations (Function calling) WITHOUT org_id parameter
	toolsDeclaration := []map[string]interface{}{
		{
			"functionDeclarations": []map[string]interface{}{
				{
					"name":        "search_signals",
					"description": "Cari sinyal pendanaan CSR, alokasi budget, dan berita terbaru dari perusahaan",
					"parameters": map[string]interface{}{
						"type": "OBJECT",
						"properties": map[string]interface{}{
							"query": map[string]interface{}{
								"type":        "STRING",
								"description": "Kata kunci pencarian sinyal (misal: nama perusahaan, sektor, atau topik program)",
							},
						},
						"required": []string{"query"},
					},
				},
				{
					"name":        "search_companies",
					"description": "Cari daftar perusahaan publik BEI / BUMN di master database",
					"parameters": map[string]interface{}{
						"type": "OBJECT",
						"properties": map[string]interface{}{
							"query": map[string]interface{}{
								"type":        "STRING",
								"description": "Kata kunci nama perusahaan atau ticker (misal: Indosat, Pertamina, BBCA)",
							},
						},
						"required": []string{"query"},
					},
				},
				{
					"name":        "list_watchlist",
					"description": "Tampilkan daftar perusahaan yang ada dalam watchlist pemantauan aktif organisasi",
					"parameters": map[string]interface{}{
						"type":       "OBJECT",
						"properties": map[string]interface{}{},
					},
				},
				{
					"name":        "list_organization_programs",
					"description": "Tampilkan daftar program CSR dan program kerja sosial yang dimiliki oleh organisasi/lembaga pengguna saat ini (tenant)",
					"parameters": map[string]interface{}{
						"type":       "OBJECT",
						"properties": map[string]interface{}{},
					},
				},
			},
		},
	}

	// 5. Execute Gemini LLM call with function calling loop
	replyText, executedTools, err := s.callGeminiWithTools(ctx, systemPrompt, contents, toolsDeclaration, orgID)
	if err != nil {
		// Fallback graceful degradation response if LLM API fails
		replyText = "Mohon maaf, terjadi hambatan pada koneksi layanan AI. Silakan coba kembali beberapa saat lagi."
	}

	latency := int(time.Since(startTime).Milliseconds())

	// 6. Log interaction to audit table organization_ai_chat_logs
	toolCallNames := []string{}
	for _, t := range executedTools {
		toolCallNames = append(toolCallNames, t.ToolName)
	}

	_ = s.repo.CreateChatLog(ctx, model.OrganizationAIChatLog{
		OrgID:          orgID,
		UserID:         userID,
		ConversationID: conv.ID,
		Message:        message,
		Reply:          replyText,
		ToolsCalled:    executedTools,
		LatencyMS:      latency,
		ModelName:      "gemini-flash-lite-latest",
	})

	return &model.OrganizationAIChatResponse{
		Reply:          replyText,
		ConversationID: conv.ID,
		ToolsCalled:    toolCallNames,
		Timestamp:      time.Now(),
	}, nil
}

// callGeminiWithTools executes Gemini API call and handles function calling turns iteratively
func (s *OrganizationAIChatService) callGeminiWithTools(ctx context.Context, systemPrompt string, contents []map[string]interface{}, tools []map[string]interface{}, orgID uuid.UUID) (string, []model.ToolCallInfo, error) {
	if s.apiKey == "" {
		return "", nil, fmt.Errorf("AI_API_KEY is not configured")
	}

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/gemini-flash-lite-latest:generateContent?key=%s", s.apiKey)

	var executedTools []model.ToolCallInfo

	// Turn 1: Initial call to Gemini
	reqPayload := map[string]interface{}{
		"contents": contents,
		"systemInstruction": map[string]interface{}{
			"parts": []map[string]string{{"text": systemPrompt}},
		},
		"tools": tools,
		"toolConfig": map[string]interface{}{
			"functionCallingConfig": map[string]interface{}{
				"mode": "AUTO",
			},
		},
		"generationConfig": map[string]interface{}{
			"temperature": 0.2,
		},
	}

	respBody, err := s.doGeminiRequest(ctx, url, reqPayload)
	if err != nil {
		return "", nil, err
	}

	funcCall, funcArgs, candidateText, err := s.parseGeminiResponse(respBody)
	if err != nil {
		return "", nil, err
	}

	// If no function call, return direct reply
	if funcCall == "" {
		return candidateText, executedTools, nil
	}

	// Handle Function Call with closure binding of orgID
	toolResultText := ""
	queryArg, _ := funcArgs["query"].(string)

	switch funcCall {
	case "search_signals":
		signals, err := s.repo.SearchSignalsByOrg(ctx, orgID, queryArg)
		executedTools = append(executedTools, model.ToolCallInfo{
			ToolName:  "search_signals",
			Arguments: funcArgs,
			Output:    signals,
		})
		if err != nil || len(signals) == 0 {
			toolResultText = fmt.Sprintf("Hasil pencarian sinyal untuk '%s': Tidak ada sinyal CSR yang cocok di database.", queryArg)
		} else {
			var sb strings.Builder
			sb.WriteString(fmt.Sprintf("Ditemukan %d sinyal CSR untuk pencarian '%s':\n", len(signals), queryArg))
			for i, sig := range signals {
				sb.WriteString(fmt.Sprintf("%d. Perusahaan: %s | Sektor: %s | Source: %s\n   Ringkasan: %s\n   Estimasi Budget: Rp %.0f | Intent Score: %d\n   [🔍 Lihat Detail](/corporates?search=%s)\n\n",
					i+1, sig.CompanyName, sig.IndustrySector, sig.SourceType, sig.Summary, sig.EstimatedBudgetSignal, sig.IntentScore, sig.CompanyName))
			}
			toolResultText = sb.String()
		}

	case "search_companies":
		companies, err := s.repo.SearchCompaniesPublic(ctx, queryArg)
		executedTools = append(executedTools, model.ToolCallInfo{
			ToolName:  "search_companies",
			Arguments: funcArgs,
			Output:    companies,
		})
		if err != nil || len(companies) == 0 {
			toolResultText = fmt.Sprintf("Hasil pencarian perusahaan untuk '%s': Tidak ada perusahaan yang cocok di database master.", queryArg)
		} else {
			var sb strings.Builder
			sb.WriteString(fmt.Sprintf("Ditemukan %d perusahaan di database master untuk pencarian '%s':\n", len(companies), queryArg))
			for i, comp := range companies {
				tickerStr := ""
				if comp.Ticker != nil && *comp.Ticker != "" {
					tickerStr = fmt.Sprintf(" (%s)", *comp.Ticker)
				}
				webStr := ""
				if comp.Website != nil && *comp.Website != "" {
					webStr = fmt.Sprintf(" | Website: %s", *comp.Website)
				}
				sb.WriteString(fmt.Sprintf("%d. Nama: %s%s | Jenis: %s | Sektor: %s%s\n   Link Detail: [🔍 Lihat Detail](/corporates?search=%s)\n\n",
					i+1, comp.Name, tickerStr, comp.CompanyType, comp.IndustrySector, webStr, comp.Name))
			}
			toolResultText = sb.String()
		}

	case "list_watchlist":
		watchlist, err := s.repo.ListWatchlistByOrg(ctx, orgID)
		executedTools = append(executedTools, model.ToolCallInfo{
			ToolName:  "list_watchlist",
			Arguments: funcArgs,
			Output:    watchlist,
		})
		if err != nil || len(watchlist) == 0 {
			toolResultText = "Daftar Watchlist Organisasi: Belum ada perusahaan dalam pemantauan aktif."
		} else {
			var sb strings.Builder
			sb.WriteString(fmt.Sprintf("Daftar Watchlist Perusahaan Pemantauan Organisasi (%d entitas):\n", len(watchlist)))
			for i, w := range watchlist {
				sb.WriteString(fmt.Sprintf("%d. Perusahaan: %s | Keyword: %s | Interval: %d jam\n   Link Detail: [🔍 Lihat Detail](/corporates?search=%s)\n\n",
					i+1, w.CompanyName, strings.Join(w.MonitoringKeywords, ", "), w.CheckIntervalHours, w.CompanyName))
			}
			toolResultText = sb.String()
		}

	case "list_organization_programs":
		programs, err := s.repo.ListOrganizationPrograms(ctx, orgID)
		executedTools = append(executedTools, model.ToolCallInfo{
			ToolName:  "list_organization_programs",
			Arguments: funcArgs,
			Output:    programs,
		})
		if err != nil {
			toolResultText = fmt.Sprintf("Terjadi kesalahan saat mengambil daftar program: %v", err)
		} else if len(programs) == 0 {
			toolResultText = "Daftar Program Organisasi: Organisasi Anda saat ini belum memiliki program yang terdaftar di database."
		} else {
			var sb strings.Builder
			sb.WriteString(fmt.Sprintf("Organisasi Anda memiliki total %d program terdaftar:\n", len(programs)))
			for i, p := range programs {
				asnafStr := ""
				if p.AsnafCategory != "" {
					asnafStr = fmt.Sprintf(" | Asnaf: %s", p.AsnafCategory)
				}
				beneficiaryStr := ""
				if p.TargetBeneficiaries != "" {
					beneficiaryStr = fmt.Sprintf(" | Target Penerima: %s", p.TargetBeneficiaries)
				}
				sb.WriteString(fmt.Sprintf("%d. Judul Program: %s\n   - Sektor/Klaster: %s | Pilar ESG: %s%s%s\n   - Deskripsi: %s\n\n",
					i+1, p.Title, p.PrimaryCluster, p.ESGPillar, asnafStr, beneficiaryStr, p.Description))
			}
			toolResultText = sb.String()
		}

	default:
		toolResultText = "Tool tidak dikenali."
	}

	// Turn 2: Send function execution result back to Gemini
	contentsTurn2 := append(contents, map[string]interface{}{
		"role": "model",
		"parts": []map[string]interface{}{
			{
				"functionCall": map[string]interface{}{
					"name": funcCall,
					"args": funcArgs,
				},
			},
		},
	})

	contentsTurn2 = append(contentsTurn2, map[string]interface{}{
		"role": "user",
		"parts": []map[string]interface{}{
			{
				"functionResponse": map[string]interface{}{
					"name": funcCall,
					"response": map[string]interface{}{
						"name":   funcCall,
						"content": toolResultText,
					},
				},
			},
		},
	})

	reqPayloadTurn2 := map[string]interface{}{
		"contents": contentsTurn2,
		"systemInstruction": map[string]interface{}{
			"parts": []map[string]string{{"text": systemPrompt}},
		},
		"generationConfig": map[string]interface{}{
			"temperature": 0.2,
		},
	}

	respBodyTurn2, err := s.doGeminiRequest(ctx, url, reqPayloadTurn2)
	if err != nil {
		// Fallback to presenting raw tool result if turn 2 fails
		return fmt.Sprintf("Berdasarkan hasil pencarian database:\n%s", toolResultText), executedTools, nil
	}

	_, _, finalReply, err := s.parseGeminiResponse(respBodyTurn2)
	if err != nil || strings.TrimSpace(finalReply) == "" {
		return fmt.Sprintf("Berikut hasil pencarian:\n%s", toolResultText), executedTools, nil
	}

	return finalReply, executedTools, nil
}

func (s *OrganizationAIChatService) doGeminiRequest(ctx context.Context, url string, reqPayload interface{}) ([]byte, error) {
	jsonBytes, err := json.Marshal(reqPayload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Gemini API error %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return bodyBytes, nil
}

func (s *OrganizationAIChatService) parseGeminiResponse(bodyBytes []byte) (string, map[string]interface{}, string, error) {
	var geminiResp struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text         string `json:"text,omitempty"`
					FunctionCall *struct {
						Name string                 `json:"name"`
						Args map[string]interface{} `json:"args"`
					} `json:"functionCall,omitempty"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}

	if err := json.Unmarshal(bodyBytes, &geminiResp); err != nil || len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return "", nil, "", fmt.Errorf("invalid gemini response payload")
	}

	parts := geminiResp.Candidates[0].Content.Parts
	for _, part := range parts {
		if part.FunctionCall != nil {
			return part.FunctionCall.Name, part.FunctionCall.Args, "", nil
		}
	}

	for _, part := range parts {
		if part.Text != "" {
			return "", nil, part.Text, nil
		}
	}

	return "", nil, "", nil
}
