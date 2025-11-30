package client

import (
	"crypto/tls"
	"errors"
	"os"

	"resty.dev/v3"
)

// Client wraps HTTP interactions with the GophKeeper server.
type Client struct {
	http     *resty.Client
	jwtToken []byte
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

	token, err := readTokenFromFile(authFileName)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}

	return &Client{
		http:     httpClient,
		jwtToken: token,
	}, nil
}

func (c *Client) SetDisableWarn(disable bool) {
	c.http.SetDisableWarn(disable)
}
