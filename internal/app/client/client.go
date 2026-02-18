package client

import (
	"crypto/tls"
	"errors"
	"os"

	"resty.dev/v3"
)

// Client wraps HTTP interactions with the GophKeeper server.
type Client struct {
	http         *resty.Client
	accessToken  []byte
	refreshToken []byte
	configFile   string
}

// NewClient initializes a Client with the given server URL and auth token file.
func NewClient(
	serverURL string,
	authFileName string,
	insecureSkipVerify bool,
) (*Client, error) {
	httpClient := resty.New()
	httpClient.
		SetHeader("User-Agent", "GophKeeper http client").
		SetBaseURL(serverURL)

	httpClient.SetTLSClientConfig(&tls.Config{
		InsecureSkipVerify: insecureSkipVerify,
	})

	tokens, err := readTokensFromFile(authFileName)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}

	var accessToken string
	var refreshToken string
	if tokens != nil {
		accessToken = tokens.AccessToken
		refreshToken = tokens.RefreshToken
	}

	return &Client{
		http:         httpClient,
		accessToken:  []byte(accessToken),
		refreshToken: []byte(refreshToken),
		configFile:   authFileName,
	}, nil
}

func (c *Client) SetDisableWarn(disable bool) {
	c.http.SetDisableWarn(disable)
}
