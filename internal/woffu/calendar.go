package woffu

import (
	"fmt"
	"strings"
	"time"
)

// GetSignInfo returns today's signing information.
func GetSignInfo(companyClient *Client, token string) (*SignInfo, error) {
	now := time.Now()
	todayStr := now.Format("2006-01-02")
	prevYearEnd := fmt.Sprintf("%d-12-31T23:00:00.000Z", now.Year()-1)

	var data []WoffuCalendarEvent
	err := companyClient.doJSON("GET", "/api/users/calendar-events?fromDate="+prevYearEnd, nil, map[string]string{
		"Authorization": "Bearer " + token,
	}, &data)
	if err != nil {
		return nil, fmt.Errorf("get calendar events: %w", err)
	}

	var todayEntry *WoffuCalendarEvent
	for i := range data {
		if strings.HasPrefix(data[i].Date, todayStr) {
			todayEntry = &data[i]
			break
		}
	}

	isTelework := false
	shouldSign := false

	if todayEntry != nil {
		// Check both approved and pending presence events for telework
		allPresence := append(todayEntry.Event.PresenceEvents, todayEntry.Event.PresencePendingEvents...)
		for _, e := range allPresence {
			if strings.Contains(strings.ToLower(e.AgreementEvent), "teletrabajo") {
				isTelework = true
				break
			}
		}

		shouldSign = !todayEntry.IsWeekend &&
			!todayEntry.Calendar.HasHoliday &&
			!todayEntry.Calendar.HasEvent &&
			len(todayEntry.Event.AbsenceEvents) == 0
	}

	var nextEvents []SignEvent
	for _, e := range data {
		if e.Date < todayStr {
			continue
		}
		if e.Calendar.HasHoliday || e.Calendar.HasEvent || len(e.Event.AbsenceEvents) > 0 {
			date := e.Date
			if idx := strings.Index(date, "T"); idx != -1 {
				date = date[:idx]
			}
			var names []string
			names = append(names, e.Calendar.HolidayNames...)
			names = append(names, e.Calendar.EventNames...)
			nextEvents = append(nextEvents, SignEvent{Date: date, Names: names})
		}
	}

	return &SignInfo{
		Date:       todayStr,
		IsTelework: isTelework,
		ShouldSign: shouldSign,
		NextEvents: nextEvents,
	}, nil
}
