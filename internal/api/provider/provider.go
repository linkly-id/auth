package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/linkly-id/auth/internal/utilities"
	"golang.org/x/oauth2"
)

var defaultTimeout time.Duration = time.Second * 10

func init() {
	timeoutStr := os.Getenv("GOTRUE_INTERNAL_HTTP_TIMEOUT")
	if timeoutStr != "" {
		if timeout, err := time.ParseDuration(timeoutStr); err != nil {
			log.Fatalf("error loading GOTRUE_INTERNAL_HTTP_TIMEOUT: %v", err.Error())
		} else if timeout != 0 {
			defaultTimeout = timeout
		}
	}
}

type Claims struct {
	// Reserved claims
	Issuer  string  `json:"iss,omitempty" structs:"iss,omitempty"`
	Subject string  `json:"sub,omitempty" structs:"sub,omitempty"`
	Aud     string  `json:"aud,omitempty" structs:"aud,omitempty"`
	Iat     float64 `json:"iat,omitempty" structs:"iat,omitempty"`
	Exp     float64 `json:"exp,omitempty" structs:"exp,omitempty"`

	// Default profile claims
	Name              string `json:"name,omitempty" structs:"name,omitempty"`
	FamilyName        string `json:"family_name,omitempty" structs:"family_name,omitempty"`
	GivenName         string `json:"given_name,omitempty" structs:"given_name,omitempty"`
	MiddleName        string `json:"middle_name,omitempty" structs:"middle_name,omitempty"`
	NickName          string `json:"nickname,omitempty" structs:"nickname,omitempty"`
	PreferredUsername string `json:"preferred_username,omitempty" structs:"preferred_username,omitempty"`
	Profile           string `json:"profile,omitempty" structs:"profile,omitempty"`
	Picture           string `json:"picture,omitempty" structs:"picture,omitempty"`
	Website           string `json:"website,omitempty" structs:"website,omitempty"`
	Gender            string `json:"gender,omitempty" structs:"gender,omitempty"`
	Birthdate         string `json:"birthdate,omitempty" structs:"birthdate,omitempty"`
	ZoneInfo          string `json:"zoneinfo,omitempty" structs:"zoneinfo,omitempty"`
	Locale            string `json:"locale,omitempty" structs:"locale,omitempty"`
	UpdatedAt         string `json:"updated_at,omitempty" structs:"updated_at,omitempty"`
	Email             string `json:"email,omitempty" structs:"email,omitempty"`
	EmailVerified     bool   `json:"email_verified,omitempty" structs:"email_verified"`
	Phone             string `json:"phone,omitempty" structs:"phone,omitempty"`
	PhoneVerified     bool   `json:"phone_verified,omitempty" structs:"phone_verified"`

	// Custom profile claims that are provider specific
	CustomClaims map[string]interface{} `json:"custom_claims,omitempty" structs:"custom_claims,omitempty"`

	// TODO: Deprecate in next major release
	FullName    string `json:"full_name,omitempty" structs:"full_name,omitempty"`
	AvatarURL   string `json:"avatar_url,omitempty" structs:"avatar_url,omitempty"`
	Slug        string `json:"slug,omitempty" structs:"slug,omitempty"`
	ProviderId  string `json:"provider_id,omitempty" structs:"provider_id,omitempty"`
	UserNameKey string `json:"user_name,omitempty" structs:"user_name,omitempty"`
}

// Email is a struct that provides information on whether an email is verified or is the primary email address
type Email struct {
	Email    string
	Verified bool
	Primary  bool
}

// UserProvidedData is a struct that contains the user's data returned from the oauth provider
type UserProvidedData struct {
	Emails   []Email
	Metadata *Claims
}

// Provider is an interface for interacting with external account providers
type Provider interface {
	AuthCodeURL(string, ...oauth2.AuthCodeOption) string
}

// OAuthProvider specifies additional methods needed for providers using OAuth
type OAuthProvider interface {
	AuthCodeURL(string, ...oauth2.AuthCodeOption) string
	GetUserData(context.Context, *oauth2.Token) (*UserProvidedData, error)
	GetOAuthToken(string) (*oauth2.Token, error)
}

func chooseHost(base, defaultHost string) string {
	if base == "" {
		return "https://" + defaultHost
	}

	baseLen := len(base)
	if base[baseLen-1] == '/' {
		return base[:baseLen-1]
	}

	return base
}

func makeRequest(ctx context.Context, tok *oauth2.Token, g *oauth2.Config, url string, dst interface{}) error {
	client := g.Client(ctx, tok)
	client.Timeout = defaultTimeout
	res, err := client.Get(url)
	if err != nil {
		return err
	}
	defer utilities.SafeClose(res.Body)

	bodyBytes, _ := io.ReadAll(res.Body)
	res.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
		return httpError(res.StatusCode, string(bodyBytes))
	}

	if err := json.NewDecoder(res.Body).Decode(dst); err != nil {
		return err
	}

	return nil
}
