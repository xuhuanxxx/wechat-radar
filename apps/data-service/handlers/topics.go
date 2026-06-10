package handlers

import (
	"database/sql"
	"net/http"
	"strings"

	"github.com/xuhuanxxx/wechat-radar/apps/data-service/models"
)

// Topics returns topics for a chatroom
func (h *Handlers) Topics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	chatroomID := r.URL.Query().Get("chatroom_id")
	date := r.URL.Query().Get("date")

	query := "SELECT id, chatroom_id, date, topic, category, confidence, message_ids FROM topics WHERE 1=1"
	args := []interface{}{}

	if chatroomID != "" {
		query += " AND chatroom_id = ?"
		args = append(args, chatroomID)
	}
	if date != "" {
		query += " AND date = ?"
		args = append(args, date)
	}
	query += " ORDER BY date DESC, confidence DESC"

	rows, err := h.db.Query(query, args...)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer rows.Close()

	topics := []models.Topic{}
	for rows.Next() {
		var t models.Topic
		var messageIDs sql.NullString
		if err := rows.Scan(&t.ID, &t.ChatroomID, &t.Date, &t.Topic, &t.Category, &t.Confidence, &messageIDs); err != nil {
			continue
		}
		if messageIDs.Valid {
			t.MessageIDs = messageIDs.String
		}
		topics = append(topics, t)
	}

	writeJSON(w, http.StatusOK, models.TopicsResponse{
		OK:         true,
		ChatroomID: chatroomID,
		Date:       date,
		Topics:     topics,
	})
}

// TopicDetail handles GET /api/topics/:id
func (h *Handlers) TopicDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	idStr := getPathParam(r, "/api/topics/")
	if idStr == "" {
		writeError(w, http.StatusBadRequest, "topic id required")
		return
	}

	var t models.Topic
	var messageIDs sql.NullString
	err := h.db.QueryRow(
		"SELECT id, chatroom_id, date, topic, category, confidence, message_ids FROM topics WHERE id = ?",
		idStr,
	).Scan(&t.ID, &t.ChatroomID, &t.Date, &t.Topic, &t.Category, &t.Confidence, &messageIDs)
	if err != nil {
		if err == sql.ErrNoRows {
			writeError(w, http.StatusNotFound, "Topic not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if messageIDs.Valid {
		t.MessageIDs = messageIDs.String
	}

	writeJSON(w, http.StatusOK, models.TopicDetailResponse{
		OK:    true,
		Topic: t,
	})
}

// TopicLinks handles GET /api/topics/links
func (h *Handlers) TopicLinks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	date := r.URL.Query().Get("date")
	if date == "" {
		writeError(w, http.StatusBadRequest, "date required")
		return
	}

	// Extract links from topic-related messages for the given date
	// MVP: find messages on the date that contain URLs and are related to topics
	rows, err := h.db.Query(`
		SELECT m.chatroom_id, m.local_id, m.content, m.date
		FROM messages m
		WHERE m.date = ? AND m.content LIKE '%http%'
		ORDER BY m.timestamp DESC
		LIMIT 200`,
		date,
	)
	if err != nil {
		writeJSON(w, http.StatusOK, models.TopicLinksResponse{
			OK:    true,
			Date:  date,
			Links: []models.Link{},
			Count: 0,
		})
		return
	}
	defer rows.Close()

	links := []models.Link{}
	seen := make(map[string]bool)
	for rows.Next() {
		var chatroomID, msgID, content, msgDate string
		if err := rows.Scan(&chatroomID, &msgID, &content, &msgDate); err != nil {
			continue
		}
		urls := urlRegex.FindAllString(content, -1)
		for _, url := range urls {
			key := chatroomID + "|" + url
			if seen[key] {
				continue
			}
			seen[key] = true
			links = append(links, models.Link{
				ChatroomID: chatroomID,
				MessageID:  msgID,
				URL:        url,
				Date:       msgDate,
			})
		}
	}

	writeJSON(w, http.StatusOK, models.TopicLinksResponse{
		OK:    true,
		Date:  date,
		Links: links,
		Count: len(links),
	})
}

// AnalyzeTopics triggers topic analysis
func (h *Handlers) AnalyzeTopics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req models.TopicAnalyzeRequest
	if err := parseJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	if req.ChatroomID == "" || req.Date == "" {
		writeError(w, http.StatusBadRequest, "chatroom_id and date required")
		return
	}

	// Simple keyword-based topic extraction
	topics, err := h.extractTopics(req.ChatroomID, req.Date)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, models.TopicAnalyzeResponse{
		OK:     true,
		Topics: topics,
	})
}

func (h *Handlers) extractTopics(chatroomID, date string) ([]models.Topic, error) {
	// Get messages for the day
	rows, err := h.db.Query(
		"SELECT content FROM messages WHERE chatroom_id = ? AND date = ?",
		chatroomID, date,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Simple keyword frequency analysis
	wordFreq := make(map[string]int)
	for rows.Next() {
		var content string
		if err := rows.Scan(&content); err != nil {
			continue
		}
		words := extractWords(content)
		for _, w := range words {
			if len(w) > 3 {
				wordFreq[w]++
			}
		}
	}

	// Get top words as topics
	topics := []models.Topic{}
	for word, freq := range wordFreq {
		if freq >= 2 {
			topics = append(topics, models.Topic{
				ChatroomID: chatroomID,
				Date:       date,
				Topic:      word,
				Category:   "general",
				Confidence: float64(freq) / 10.0,
			})
		}
	}

	return topics, nil
}

func extractWords(content string) []string {
	// Simple word extraction - split by common delimiters
	content = strings.ToLower(content)
	replacer := strings.NewReplacer(
		".", " ", ",", " ", "!", " ", "?", " ",
		"；", " ", "：", " ", "\"", " ", "'", " ",
		"（", " ", "）", " ", "【", " ", "】", " ",
	)
	content = replacer.Replace(content)
	return strings.Fields(content)
}
