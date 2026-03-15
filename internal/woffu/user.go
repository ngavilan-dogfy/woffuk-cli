package woffu

import (
	"fmt"
	"strconv"
)

// GetAvailableEvents returns the user's available events (vacations, hours, etc).
func GetAvailableEvents(companyClient *Client, token string) ([]AvailableUserEvent, error) {
	var data []woffuAgreementEventAvailability
	err := companyClient.doJSON("GET", "/api/user-agreement-events/availability", nil, map[string]string{
		"Authorization": "Bearer " + token,
	}, &data)
	if err != nil {
		return nil, fmt.Errorf("get available events: %w", err)
	}

	events := make([]AvailableUserEvent, 0, len(data))
	for _, item := range data {
		available := 0.0
		if len(item.AvailableFormatted.Values) > 0 {
			available, _ = strconv.ParseFloat(item.AvailableFormatted.Values[0], 64)
		}

		unit := "days"
		if item.AvailableFormatted.Resource == "_HoursFormatted" {
			unit = "hours"
		}

		events = append(events, AvailableUserEvent{
			Name:      item.Name,
			Available: available,
			Unit:      unit,
		})
	}

	return events, nil
}
