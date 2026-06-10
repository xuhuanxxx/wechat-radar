package handlers

import (
	"net/http"
	"strings"

	"github.com/xuhuanxxx/wechat-radar/apps/data-service/models"
)

// AIClassify handles POST /api/ai-classify with keyword-based classification
func (h *Handlers) AIClassify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req models.AIClassifyRequest
	if err := parseJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	if len(req.ChatroomIDs) == 0 {
		writeJSON(w, http.StatusOK, models.AIClassifyResponse{
			OK:         true,
			Results:    []models.AIClassifyResult{},
			Classified: 0,
		})
		return
	}

	results := []models.AIClassifyResult{}
	for _, chatroomID := range req.ChatroomIDs {
		result := h.classifyGroup(chatroomID, req.Date)
		results = append(results, result)
	}

	writeJSON(w, http.StatusOK, models.AIClassifyResponse{
		OK:         true,
		Results:    results,
		Classified: len(results),
	})
}

func (h *Handlers) classifyGroup(chatroomID, date string) models.AIClassifyResult {
	// Get group name
	var name string
	h.db.QueryRow("SELECT name FROM groups WHERE chatroom_id = ?", chatroomID).Scan(&name)
	if name == "" {
		name = chatroomID
	}

	// Build query for messages
	query := "SELECT content FROM messages WHERE chatroom_id = ?"
	args := []interface{}{chatroomID}
	if date != "" {
		query += " AND date = ?"
		args = append(args, date)
	}

	rows, err := h.db.Query(query, args...)
	if err != nil {
		return models.AIClassifyResult{
			ChatroomID: chatroomID,
			Name:       name,
			Category:   "unknown",
			Confidence: 0,
		}
	}
	defer rows.Close()

	// Simple keyword-based classification
	keywords := map[string][]string{
		"tech":     {"代码", "编程", "开发", "bug", "git", "github", "API", "服务器", "部署", "技术", "code", "programming", "dev"},
		"product":  {"产品", "需求", "用户", "功能", "迭代", "PRD", "设计", "体验", "product", "feature", "user"},
		"business": {"销售", "客户", "合同", "商务", "合作", " revenue", "业务", "商机", "sales", "business", "deal"},
		"ops":      {"运维", "监控", "报警", "日志", "备份", "扩容", "ops", "monitor", "alert"},
		"social":   {"聚会", "活动", "旅游", "吃饭", "周末", "节日", "social", "party", "event"},
		"finance":  {"财务", "报销", "预算", "发票", "工资", "薪资", "finance", "budget", "invoice"},
	}

	categoryScores := make(map[string]int)
	var totalWords int

	for rows.Next() {
		var content string
		if err := rows.Scan(&content); err != nil {
			continue
		}
		content = strings.ToLower(content)
		for cat, words := range keywords {
			for _, word := range words {
				if strings.Contains(content, strings.ToLower(word)) {
					categoryScores[cat]++
				}
			}
		}
		totalWords += len(content)
	}

	// Determine best category
	bestCategory := "general"
	bestScore := 0
	for cat, score := range categoryScores {
		if score > bestScore {
			bestScore = score
			bestCategory = cat
		}
	}

	confidence := 0.5
	if totalWords > 0 {
		confidence = float64(bestScore) / float64(totalWords) * 10.0
		if confidence > 1.0 {
			confidence = 1.0
		}
		if confidence < 0.1 {
			confidence = 0.1
		}
	}

	var reason string
	if bestScore > 0 {
		reason = "Matched keywords for " + bestCategory
	}

	return models.AIClassifyResult{
		ChatroomID: chatroomID,
		Name:       name,
		Category:   bestCategory,
		Confidence: confidence,
		Reason:     reason,
	}
}
