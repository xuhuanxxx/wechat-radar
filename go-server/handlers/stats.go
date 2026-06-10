package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"go-server/models"
)

// Stats returns dashboard statistics
func (h *Handlers) Stats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	rangeParam := getQueryParam(r, "range", "7d")
	anchorDate := getQueryParam(r, "anchorDate", "")

	window, err := parseRange(rangeParam, anchorDate)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Query stats
	stats, err := h.queryStats(window)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	stats.Range = rangeParam
	stats.Window = window

	writeJSON(w, http.StatusOK, stats)
}

// Intelligence returns the intelligence brief
func (h *Handlers) Intelligence(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	rangeParam := getQueryParam(r, "range", "7d")
	window, err := parseRange(rangeParam, "")
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	brief, err := h.queryIntelligence(window)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, brief)
}

func (h *Handlers) queryStats(window models.TimeWindow) (*models.StatsResponse, error) {
	resp := &models.StatsResponse{
		OK:           true,
		ActiveGroups: []models.ActiveGroup{},
		Categories:   []models.CategoryStat{},
	}

	// Total messages
	var totalMessages int
	err := h.db.QueryRow(
		"SELECT COUNT(*) FROM messages WHERE date >= ? AND date <= ?",
		window.StartDate, window.EndDate,
	).Scan(&totalMessages)
	if err != nil {
		return nil, err
	}
	resp.Cards.TotalMessages = totalMessages

	// Unique senders
	var uniqueSenders int
	err = h.db.QueryRow(
		"SELECT COUNT(DISTINCT sender) FROM messages WHERE date >= ? AND date <= ?",
		window.StartDate, window.EndDate,
	).Scan(&uniqueSenders)
	if err != nil {
		return nil, err
	}
	resp.Cards.UniqueSenders = uniqueSenders

	// Active groups
	rows, err := h.db.Query(`
		SELECT chatroom_id, COUNT(*) as msg_count, COUNT(DISTINCT sender) as sender_count
		FROM messages
		WHERE date >= ? AND date <= ?
		GROUP BY chatroom_id
		ORDER BY msg_count DESC
		LIMIT 10`,
		window.StartDate, window.EndDate,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var g models.ActiveGroup
		if err := rows.Scan(&g.ChatroomID, &g.MessageCount, &g.SenderCount); err != nil {
			continue
		}
		// Get group name
		var name string
		h.db.QueryRow("SELECT name FROM groups WHERE chatroom_id = ?", g.ChatroomID).Scan(&name)
		if name != "" {
			g.Name = name
		} else {
			g.Name = g.ChatroomID
		}
		g.TopSenders = []models.SenderCount{}
		resp.ActiveGroups = append(resp.ActiveGroups, g)
	}

	// Trend data (daily message counts)
	trendRows, err := h.db.Query(`
		SELECT date, COUNT(*) as count
		FROM messages
		WHERE date >= ? AND date <= ?
		GROUP BY date
		ORDER BY date`,
		window.StartDate, window.EndDate,
	)
	if err != nil {
		return nil, err
	}
	defer trendRows.Close()

	resp.Trend.Labels = []string{}
	resp.Trend.Data = []models.TrendPoint{}
	for trendRows.Next() {
		var date string
		var count int
		if err := trendRows.Scan(&date, &count); err != nil {
			continue
		}
		resp.Trend.Labels = append(resp.Trend.Labels, date)
		resp.Trend.Data = append(resp.Trend.Data, models.TrendPoint{Date: date, Count: count})
	}

	// Categories (from topics)
	catRows, err := h.db.Query(`
		SELECT category, COUNT(*) as count
		FROM topics
		WHERE date >= ? AND date <= ?
		GROUP BY category
		ORDER BY count DESC`,
		window.StartDate, window.EndDate,
	)
	if err != nil {
		return nil, err
	}
	defer catRows.Close()

	for catRows.Next() {
		var c models.CategoryStat
		if err := catRows.Scan(&c.Category, &c.Count); err != nil {
			continue
		}
		resp.Categories = append(resp.Categories, c)
	}

	return resp, nil
}

func (h *Handlers) queryIntelligence(window models.TimeWindow) (*models.IntelligenceBrief, error) {
	brief := &models.IntelligenceBrief{
		OK: true,
	}

	// Get total messages in period
	var totalMessages int
	h.db.QueryRow(
		"SELECT COUNT(*) FROM messages WHERE date >= ? AND date <= ?",
		window.StartDate, window.EndDate,
	).Scan(&totalMessages)

	// Get previous period for comparison
	days := int(window.EndDateTime.Sub(window.StartDateTime).Hours()/24) + 1
	prevEnd := window.StartDateTime.AddDate(0, 0, -1)
	prevStart := prevEnd.AddDate(0, 0, -days+1)
	prevStartStr := prevStart.Format("2006-01-02")
	prevEndStr := prevEnd.Format("2006-01-02")

	var prevMessages int
	h.db.QueryRow(
		"SELECT COUNT(*) FROM messages WHERE date >= ? AND date <= ?",
		prevStartStr, prevEndStr,
	).Scan(&prevMessages)

	// Build summary
	var changeStr string
	if prevMessages > 0 {
		changePct := float64(totalMessages-prevMessages) / float64(prevMessages) * 100
		if changePct > 0 {
			changeStr = fmt.Sprintf("(+%.0f%%)", changePct)
		} else if changePct < 0 {
			changeStr = fmt.Sprintf("(%.0f%%)", changePct)
		} else {
			changeStr = "(持平)"
		}
	} else {
		changeStr = "(新数据)"
	}

	brief.Summary = fmt.Sprintf("过去 %d 天共产生 %d 条消息 %s", days, totalMessages, changeStr)

	// Top active groups as highlights
	rows, err := h.db.Query(`
		SELECT chatroom_id, COUNT(*) as cnt
		FROM messages
		WHERE date >= ? AND date <= ?
		GROUP BY chatroom_id
		ORDER BY cnt DESC
		LIMIT 3`,
		window.StartDate, window.EndDate,
	)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var chatroomID string
			var cnt int
			if err := rows.Scan(&chatroomID, &cnt); err != nil {
				continue
			}
			var name string
			h.db.QueryRow("SELECT name FROM groups WHERE chatroom_id = ?", chatroomID).Scan(&name)
			if name == "" {
				name = chatroomID
			}
			brief.Highlights = append(brief.Highlights, fmt.Sprintf("%s: %d 条消息", name, cnt))
		}
	}

	return brief, nil
}

func parseRange(rangeParam, anchorDate string) (models.TimeWindow, error) {
	var end time.Time
	if anchorDate != "" {
		var err error
		end, err = time.Parse("2006-01-02", anchorDate)
		if err != nil {
			return models.TimeWindow{}, fmt.Errorf("invalid anchorDate: %s", anchorDate)
		}
	} else {
		end = time.Now()
	}
	end = end.Truncate(24 * time.Hour)

	days := 7
	if strings.HasSuffix(rangeParam, "d") {
		if d, err := strconv.Atoi(strings.TrimSuffix(rangeParam, "d")); err == nil && d > 0 {
			days = d
		}
	}

	start := end.AddDate(0, 0, -(days - 1))

	return models.TimeWindow{
		StartDate:      start.Format("2006-01-02"),
		EndDate:        end.Format("2006-01-02"),
		StartDateTime:  start,
		EndDateTime:    end,
	}, nil
}
