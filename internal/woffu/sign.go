package woffu

import (
	"fmt"
	"time"
)

// DoSign clocks in/out on Woffu with the given coordinates.
func DoSign(companyClient *Client, token string, lat, lon float64) (string, error) {
	body := woffuSignBody{
		AgreementEventID: nil,
		DeviceID:         "WebApp",
		Latitude:         lat,
		Longitude:        lon,
		RequestID:        nil,
		TimezoneOffset:   timezoneOffset(),
	}

	var resp woffuSignResponse
	err := companyClient.doJSON("POST", "/api/svc/signs/signs", body, map[string]string{
		"Authorization": "Bearer " + token,
	}, &resp)
	if err != nil {
		return "", fmt.Errorf("sign: %w", err)
	}

	return resp.SignEventID, nil
}

func timezoneOffset() int {
	_, offset := time.Now().Zone()
	return -(offset / 60)
}
