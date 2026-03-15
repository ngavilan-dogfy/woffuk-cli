package woffu

import (
	"fmt"
	"strings"
	"time"
)

// CalendarDay represents a single day in the calendar view.
type CalendarDay struct {
	Date       string   `json:"date"`
	DayName    string   `json:"day"`
	Status     string   `json:"status"` // "working", "weekend", "holiday", "absence"
	Mode       string   `json:"mode"`   // "office", "remote", ""
	IsHoliday  bool     `json:"is_holiday"`
	IsWeekend  bool     `json:"is_weekend"`
	HasAbsence bool     `json:"has_absence"`
	EventNames []string `json:"events,omitempty"`
}

// GetCalendarMonth fetches calendar data for a specific month (0 = current).
func GetCalendarMonth(companyClient *Client, token string, month int) ([]CalendarDay, error) {
	now := time.Now()
	targetMonth := now.Month()
	targetYear := now.Year()
	if month > 0 && month <= 12 {
		targetMonth = time.Month(month)
		if targetMonth < now.Month() {
			targetYear++ // Assume next year if month is in the past
		}
	}

	prevYearEnd := fmt.Sprintf("%d-12-31T23:00:00.000Z", targetYear-1)

	var data []WoffuCalendarEvent
	err := companyClient.doJSON("GET", "/api/users/calendar-events?fromDate="+prevYearEnd, nil, map[string]string{
		"Authorization": "Bearer " + token,
	}, &data)
	if err != nil {
		return nil, fmt.Errorf("get calendar: %w", err)
	}

	var days []CalendarDay
	for _, e := range data {
		if !strings.HasPrefix(e.Date, fmt.Sprintf("%d-%02d", targetYear, targetMonth)) {
			continue
		}

		date := e.Date
		if idx := strings.Index(date, "T"); idx != -1 {
			date = date[:idx]
		}

		t, _ := time.Parse("2006-01-02", date)
		dayName := t.Weekday().String()[:3]

		status := "working"
		if e.IsWeekend {
			status = "weekend"
		} else if e.Calendar.HasHoliday {
			status = "holiday"
		} else if len(e.Event.AbsenceEvents) > 0 {
			status = "absence"
		}

		mode := ""
		if !e.IsWeekend && !e.Calendar.HasHoliday {
			allPresence := make([]presenceEvent, 0, len(e.Event.PresenceEvents)+len(e.Event.PresencePendingEvents))
			allPresence = append(allPresence, e.Event.PresenceEvents...)
			allPresence = append(allPresence, e.Event.PresencePendingEvents...)
			for _, p := range allPresence {
				if strings.Contains(strings.ToLower(p.AgreementEvent), "teletrabajo") {
					mode = "remote"
					break
				}
			}
			if mode == "" {
				mode = "office"
			}
		}

		var eventNames []string
		eventNames = append(eventNames, e.Calendar.HolidayNames...)
		eventNames = append(eventNames, e.Calendar.EventNames...)

		days = append(days, CalendarDay{
			Date:       date,
			DayName:    dayName,
			Status:     status,
			Mode:       mode,
			IsHoliday:  e.Calendar.HasHoliday,
			IsWeekend:  e.IsWeekend,
			HasAbsence: len(e.Event.AbsenceEvents) > 0,
			EventNames: eventNames,
		})
	}

	return days, nil
}
