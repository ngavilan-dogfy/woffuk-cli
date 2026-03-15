package woffu

import (
	"fmt"
	"strings"
	"time"
)

// SignRecord represents a single clock in/out event.
type SignRecord struct {
	Date string `json:"date"`
	Time string `json:"time"`
	Type string `json:"type"` // "in" or "out"
}

type woffuSignRecord struct {
	SignEventId    int    `json:"SignEventId"`
	UserId         int    `json:"UserId"`
	TrueDate       string `json:"TrueDate"`
	TrueTime       string `json:"TrueTime"`
	Date           string `json:"Date"`
	SignIn         bool   `json:"SignIn"`
	Latitude       float64 `json:"Latitude"`
	Longitude      float64 `json:"Longitude"`
}

// GetSignHistory fetches sign records for a date range.
func GetSignHistory(companyClient *Client, token string, from, to time.Time) ([]SignRecord, error) {
	fromStr := from.Format("2006-01-02") + "T00:00:00.000Z"
	toStr := to.Format("2006-01-02") + "T23:59:59.999Z"

	var data []woffuSignRecord
	err := companyClient.doJSON("GET",
		fmt.Sprintf("/api/signs?fromDate=%s&toDate=%s&pageSize=200", fromStr, toStr),
		nil, map[string]string{
			"Authorization": "Bearer " + token,
		}, &data)
	if err != nil {
		return nil, fmt.Errorf("get sign history: %w", err)
	}

	records := make([]SignRecord, 0, len(data))
	for _, s := range data {
		date := s.Date
		if idx := strings.Index(date, "T"); idx != -1 {
			date = date[:idx]
		}

		timeStr := s.TrueTime
		if timeStr == "" {
			// Parse from Date field
			t, err := time.Parse("2006-01-02T15:04:05", s.Date)
			if err == nil {
				timeStr = t.Format("15:04")
			}
		}

		signType := "in"
		if !s.SignIn {
			signType = "out"
		}

		records = append(records, SignRecord{
			Date: date,
			Time: timeStr,
			Type: signType,
		})
	}

	return records, nil
}
