package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"golang.org/x/oauth2"
)

type ProviderConfig struct {
	Name         string
	ClientID     string
	ClientSecret string
	Scopes       []string
	AuthURL      string
	TokenURL     string
	UserInfoURL  string
}

type ProviderService struct {
	providers    map[string]*oauth2.Config
	userInfoURLs map[string]string
}

type UserInfo struct {
	ID      string `json:"id"`
	Email   string `json:"email"`
	Name    string `json:"name"`
	Picture string `json:"picture"`
}

func NewProviderService(configs []ProviderConfig) *ProviderService {
	ps := &ProviderService{
		providers:    make(map[string]*oauth2.Config),
		userInfoURLs: make(map[string]string),
	}

	for _, cfg := range configs {
		if cfg.Name == "" {
			continue
		}
		ps.providers[cfg.Name] = &oauth2.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			Scopes:       cfg.Scopes,
			Endpoint: oauth2.Endpoint{
				AuthURL:  cfg.AuthURL,
				TokenURL: cfg.TokenURL,
			},
			RedirectURL: "",
		}
		ps.userInfoURLs[cfg.Name] = cfg.UserInfoURL
	}

	return ps
}

func (ps *ProviderService) GetAuthCodeURL(provider, state, redirectURI string) (string, error) {
	p, ok := ps.providers[provider]
	if !ok {
		return "", fmt.Errorf("provider not found: %s", provider)
	}
	p.RedirectURL = redirectURI
	return p.AuthCodeURL(state), nil
}

func (ps *ProviderService) Exchange(provider, code, redirectURI string) (*oauth2.Token, error) {
	p, ok := ps.providers[provider]
	if !ok {
		return nil, fmt.Errorf("provider not found: %s", provider)
	}
	p.RedirectURL = redirectURI
	return p.Exchange(context.Background(), code)
}

func (ps *ProviderService) GetUserInfo(provider string, token *oauth2.Token) (*UserInfo, error) {
	userInfoURL, ok := ps.userInfoURLs[provider]
	if !ok {
		return nil, fmt.Errorf("provider not found: %s", provider)
	}

	client := &http.Client{}
	req, err := http.NewRequest("GET", userInfoURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var userInfo UserInfo
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, err
	}

	return &userInfo, nil
}

func (ps *ProviderService) HasProvider(name string) bool {
	_, ok := ps.providers[name]
	return ok
}
