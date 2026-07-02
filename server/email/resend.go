package email

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	APIKey string
	HTTP   *http.Client
}

func New(apiKey string) *Client {
	return &Client{APIKey: apiKey, HTTP: &http.Client{Timeout: 15 * time.Second}}
}

type SendRequest struct {
	From    string            `json:"from"`
	To      []string          `json:"to"`
	Subject string            `json:"subject"`
	HTML    string            `json:"html,omitempty"`
	Text    string            `json:"text,omitempty"`
	ReplyTo string            `json:"reply_to,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
}

type SendResponse struct {
	ID string `json:"id"`
}

func (c *Client) Send(req SendRequest) (*SendResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequest("POST", "https://api.resend.com/emails", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+c.APIKey)
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := c.HTTP.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("resend send: %s: %s", resp.Status, string(respBody))
	}
	var out SendResponse
	if err := json.Unmarshal(respBody, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// VerifySvixSignature verifies a Svix-style webhook signature header used by Resend.
// Header format: "v1,<base64sig> v1,<base64sig> ..."
// Signed payload: msgID + "." + msgTimestamp + "." + body
func VerifySvixSignature(secret, msgID, msgTimestamp, signatureHeader string, body []byte) error {
	if secret == "" {
		return errors.New("missing webhook secret")
	}
	// Resend gives the secret prefixed with "whsec_"; strip it before HMAC.
	rawSecret := secret
	if strings.HasPrefix(rawSecret, "whsec_") {
		rawSecret = strings.TrimPrefix(rawSecret, "whsec_")
	}
	signed := fmt.Sprintf("%s.%s.%s", msgID, msgTimestamp, string(body))
	mac := hmac.New(sha256.New, []byte(rawSecret))
	mac.Write([]byte(signed))
	expected := mac.Sum(nil)

	for _, part := range strings.Split(signatureHeader, " ") {
		kv := strings.SplitN(part, ",", 2)
		if len(kv) != 2 || kv[0] != "v1" {
			continue
		}
		got, err := decodeSig(kv[1])
		if err != nil {
			continue
		}
		if hmac.Equal(expected, got) {
			return nil
		}
	}
	return errors.New("invalid signature")
}

func decodeSig(s string) ([]byte, error) {
	// Svix uses base64 standard with padding
	if b, err := hexDecodeIfHex(s); err == nil {
		return b, nil
	}
	return base64Decode(s)
}

func hexDecodeIfHex(s string) ([]byte, error) {
	if len(s)%2 != 0 {
		return nil, errors.New("not hex")
	}
	return hex.DecodeString(s)
}

func base64Decode(s string) ([]byte, error) {
	// Try standard then URL encoding, with and without padding.
	for _, dec := range []func(string) ([]byte, error){
		stdB64, urlB64, stdRawB64, urlRawB64,
	} {
		if b, err := dec(s); err == nil {
			return b, nil
		}
	}
	return nil, errors.New("not base64")
}
