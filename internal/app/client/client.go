package client

import (
	"crypto/tls"

	"resty.dev/v3"
)

type Client struct {
	http     *resty.Client
	jwtToken []byte
}

func NewClient(serverURL string, authFileName string) (*Client, error) {
	httpClient := resty.New()
	httpClient.
		SetHeader("User-Agent", "GophKeeper http client").
		SetBaseURL(serverURL)

	httpClient.SetTLSClientConfig(&tls.Config{
		InsecureSkipVerify: true,
	})

	token, err := readToken(authFileName)
	if err != nil {
		return nil, err
	}

	return &Client{
		http:     httpClient,
		jwtToken: token,
	}, nil
}
