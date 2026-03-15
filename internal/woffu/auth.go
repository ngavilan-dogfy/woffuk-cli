package woffu

import (
	"fmt"
	"io"
	"strings"
)

// Authenticate performs the full Woffu login flow and returns a bearer token.
func Authenticate(client *Client, companyClient *Client, email, password string) (string, error) {
	// Step 1: Check new login
	var newLogin woffuNewLogin
	err := client.doJSON("GET", "/svc/accounts/authorization/use-new-login?email="+email, nil, nil, &newLogin)
	if err != nil {
		return "", fmt.Errorf("use-new-login: %w", err)
	}

	// Step 2: Get login configuration
	var loginConfig woffuLoginConfiguration
	err = client.doJSON("GET", "/svc/accounts/companies/login-configuration-by-email?email="+email, nil, nil, &loginConfig)
	if err != nil {
		return "", fmt.Errorf("login-configuration: %w", err)
	}

	// Step 3: Get token (form-urlencoded) and capture cookie
	formBody := fmt.Sprintf("grant_type=password&username=%s&password=%s", email, password)
	resp, err := client.doRaw(requestOptions{
		Method:      "POST",
		Path:        "/svc/accounts/authorization/token",
		Body:        strings.NewReader(formBody),
		ContentType: "application/x-www-form-urlencoded",
	})
	if err != nil {
		return "", fmt.Errorf("get token: %w", err)
	}
	defer resp.Body.Close()
	io.ReadAll(resp.Body)

	// Extract cookies from response
	var cookies []string
	for _, c := range resp.Cookies() {
		cookies = append(cookies, c.Name+"="+c.Value)
	}
	cookieHeader := fmt.Sprintf(`user-language="es"; woffu.lang=es; %s`, strings.Join(cookies, "; "))

	// Step 4: Get company-scoped token using the cookie
	var tokenResp woffuGetToken
	err = companyClient.doJSON("GET", "/api/svc/accounts/authorization/users/token", nil, map[string]string{
		"Cookie": cookieHeader,
	}, &tokenResp)
	if err != nil {
		return "", fmt.Errorf("company token: %w", err)
	}

	return tokenResp.Token, nil
}
