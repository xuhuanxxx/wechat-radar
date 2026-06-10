package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"go-server/models"
)

// GenerateReport generates a report for a chatroom
func (h *Handlers) GenerateReport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req models.ReportRequest
	if err := parseJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	if req.ChatroomID == "" {
		writeError(w, http.StatusBadRequest, "chatroom_id required")
		return
	}

	report, err := h.generateReport(req.ChatroomID, req.Date, req.Range)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, models.ReportResponse{
		OK:     true,
		Report: report,
	})
}

func (h *Handlers) generateReport(chatroomID, date, reportRange string) (string, error) {
	// Get group name
	var name string
	h.db.QueryRow("SELECT name FROM groups WHERE chatroom_id = ?", chatroomID).Scan(&name)
	if name == "" {
		name = chatroomID
	}

	// Determine date range
	if date == "" && reportRange == "" {
		reportRange = "7d"
	}

	var startDate, endDate string
	if date != "" {
		startDate = date
		endDate = date
	} else {
		window, err := parseRange(reportRange, "")
		if err != nil {
			return "", err
		}
		startDate = window.StartDate
		endDate = window.EndDate
	}

	// Get stats
	var messageCount, uniqueSenders int
	err := h.db.QueryRow(
		"SELECT COUNT(*), COUNT(DISTINCT sender) FROM messages WHERE chatroom_id = ? AND date >= ? AND date <= ?",
		chatroomID, startDate, endDate,
	).Scan(&messageCount, &uniqueSenders)
	if err != nil {
		return "", err
	}

	// Get top senders
	rows, err := h.db.Query(`
		SELECT sender, COUNT(*) as cnt
		FROM messages
		WHERE chatroom_id = ? AND date >= ? AND date <= ?
		GROUP BY sender
		ORDER BY cnt DESC
		LIMIT 5`,
		chatroomID, startDate, endDate,
	)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	var topSenders []string
	for rows.Next() {
		var sender string
		var cnt int
		if err := rows.Scan(&sender, &cnt); err != nil {
			continue
		}
		topSenders = append(topSenders, fmt.Sprintf("%s (%d 条)", sender, cnt))
	}

	// Build report
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# %s 群聊报告\n\n", name))
	sb.WriteString(fmt.Sprintf("**时间范围**: %s 至 %s\n\n", startDate, endDate))
	sb.WriteString(fmt.Sprintf("**消息总数**: %d\n\n", messageCount))
	sb.WriteString(fmt.Sprintf("**活跃人数**: %d\n\n", uniqueSenders))

	if len(topSenders) > 0 {
		sb.WriteString("**活跃成员**: \n")
		for _, s := range topSenders {
			sb.WriteString(fmt.Sprintf("- %s\n", s))
		}
		sb.WriteString("\n")
	}

	return sb.String(), nil
}
