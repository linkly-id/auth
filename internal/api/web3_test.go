package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/linkly-id/auth/internal/api/apierrors"
	"github.com/linkly-id/auth/internal/conf"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type Web3TestSuite struct {
	suite.Suite
	API    *API
	Config *conf.GlobalConfiguration
}

func TestWeb3(t *testing.T) {
	api, config, err := setupAPIForTest()
	require.NoError(t, err)

	ts := &Web3TestSuite{
		API:    api,
		Config: config,
	}
	defer api.db.Close()

	suite.Run(t, ts)
}

func (ts *Web3TestSuite) TestNonSolana() {
	var buffer bytes.Buffer
	require.NoError(ts.T(), json.NewEncoder(&buffer).Encode(map[string]interface{}{
		"chain": "blockchain",
	}))

	req := httptest.NewRequest(http.MethodPost, "http://localhost/token?grant_type=web3", &buffer)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	ts.API.handler.ServeHTTP(w, req)

	var firstResult struct {
		ErrorCode string `json:"error_code"`
		Message   string `json:"msg"`
	}

	assert.NoError(ts.T(), json.NewDecoder(w.Result().Body).Decode(&firstResult))
	assert.Equal(ts.T(), apierrors.ErrorCodeWeb3UnsupportedChain, firstResult.ErrorCode)
	assert.Equal(ts.T(), "Unsupported chain", firstResult.Message)
}

func (ts *Web3TestSuite) TestDisabled() {
	defer func() {
		ts.Config.External.Web3Solana.Enabled = true
	}()

	ts.Config.External.Web3Solana.Enabled = false

	var buffer bytes.Buffer
	require.NoError(ts.T(), json.NewEncoder(&buffer).Encode(map[string]interface{}{
		"chain": "solana",
	}))

	req := httptest.NewRequest(http.MethodPost, "http://localhost/token?grant_type=web3", &buffer)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	ts.API.handler.ServeHTTP(w, req)

	var firstResult struct {
		ErrorCode string `json:"error_code"`
		Message   string `json:"msg"`
	}

	assert.NoError(ts.T(), json.NewDecoder(w.Result().Body).Decode(&firstResult))
	assert.Equal(ts.T(), apierrors.ErrorCodeWeb3ProviderDisabled, firstResult.ErrorCode)
	assert.Equal(ts.T(), "Web3 provider is disabled", firstResult.Message)
}

func (ts *Web3TestSuite) TestHappyPath_FullMessage() {
	defer func() {
		ts.API.overrideTime = nil
	}()

	examples := []struct {
		now       string
		message   string
		signature string
	}{
		{
			now:       "2025-03-29T00:09:59Z",
			message:   "linkly.id wants you to sign in with your Solana account:\n9pStGkfG4TfFkk5VBwaP6XPLVXr8mq6uWfFJcchWHdwP\n\nStatement\n\nURI: https://linkly.id/\nVersion: 1\nIssued At: 2025-03-29T00:00:00Z\nExpiration Time: 2025-03-29T00:10:00Z\nNot Before: 2025-03-29T00:00:00Z",
			signature: "ZrHNyLlKHwDs2nE8uM8pBrzgRbLGy5sKuiFXChdygoLN4NZK4mcozcF1HV5sK6aZyfI5ao+kz98kr88tbGgkBg==",
		},
		{
			now:       "2025-05-16T15:01:59Z",
			message:   "localhost:5173 wants you to sign in with your Solana account:\n2S3iKQhUGa8qy9GAdHWNJhjwcCKTPhBaCpt84SaFCXXa\n\nSign in on localhost\n\nURI: http://localhost:5173/\nVersion: 1\nIssued At: 2025-05-16T14:52:03.613Z",
			signature: "Ybkgj9JAc4NGFpCn7KCOXJEKzcOqWVqEuKSkb6t30iTK+Y74xZUWn9Uq/VbNhMIkuiZSh2Xi7Y6EwzWUn3UMAw==",
		},
	}

	for _, example := range examples {
		ts.API.overrideTime = func() time.Time {
			t, _ := time.Parse(time.RFC3339, example.now)
			return t
		}

		var buffer bytes.Buffer
		require.NoError(ts.T(), json.NewEncoder(&buffer).Encode(map[string]interface{}{
			"chain":     "solana",
			"message":   example.message,
			"signature": example.signature,
		}))

		req := httptest.NewRequest(http.MethodPost, "http://localhost/token?grant_type=web3", &buffer)
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		ts.API.handler.ServeHTTP(w, req)

		assert.Equal(ts.T(), http.StatusOK, w.Code)

		var firstResult struct {
			AccessToken  string `json:"access_token"`
			RefreshToken string `json:"refresh_token"`
		}

		assert.NoError(ts.T(), json.NewDecoder(w.Result().Body).Decode(&firstResult))

		assert.NotEmpty(ts.T(), firstResult.AccessToken)
		assert.NotEmpty(ts.T(), firstResult.RefreshToken)
	}
}

func (ts *Web3TestSuite) TestHappyPath_MinimalMessage() {
	defer func() {
		ts.API.overrideTime = nil
	}()

	ts.API.overrideTime = func() time.Time {
		t, _ := time.Parse(time.RFC3339, "2025-03-29T00:09:59Z")
		return t
	}

	var buffer bytes.Buffer
	require.NoError(ts.T(), json.NewEncoder(&buffer).Encode(map[string]interface{}{
		"chain":     "solana",
		"message":   "linkly.id wants you to sign in with your Solana account:\nDYhiCjDjLBX6dNLaJ2JEv3r6HGLAvfZBDmt2UBC6wb3w\n\nStatement\n\nURI: https://linkly.id/\nVersion: 1\nIssued At: 2025-03-29T00:00:00Z",
		"signature": "W38rblT9Z5K8+pu10IATz0h85fdQBnBdYDtqdas6WT25vMy14dXdNxbtuJkFj4X/mzee1rPJCkv+Si55XddMCg==",
	}))

	req := httptest.NewRequest(http.MethodPost, "http://localhost/token?grant_type=web3", &buffer)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	ts.API.handler.ServeHTTP(w, req)

	assert.Equal(ts.T(), http.StatusOK, w.Code)

	var firstResult struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}

	assert.NoError(ts.T(), json.NewDecoder(w.Result().Body).Decode(&firstResult))

	assert.NotEmpty(ts.T(), firstResult.AccessToken)
	assert.NotEmpty(ts.T(), firstResult.RefreshToken)
}

func (ts *Web3TestSuite) TestValidationRules_URINotHTTPSButIsHTTP() {
	defer func() {
		ts.API.overrideTime = nil
	}()

	ts.API.overrideTime = func() time.Time {
		t, _ := time.Parse(time.RFC3339, "2025-03-29T00:00:00Z")
		return t
	}

	var buffer bytes.Buffer
	require.NoError(ts.T(), json.NewEncoder(&buffer).Encode(map[string]interface{}{
		"chain":     "solana",
		"message":   "linkly.id wants you to sign in with your Solana account:\nAMtzzkPVJ1V5pJg9JkYVpf9i4cmt5F527UyGv7RZy3Fg\n\nStatement\n\nURI: http://supaabse.com\nVersion: 1\nIssued At: 2025-03-29T00:00:00Z",
		"signature": "7yCIJKtiSUZJy35gCy871vSQZU2NvoTQYTwl4t/Y38WHYmyHbktplOALscnjiBuWrolar0z02dZw5p9IC9IdBA==",
	}))

	req := httptest.NewRequest(http.MethodPost, "http://localhost/token?grant_type=web3", &buffer)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	ts.API.handler.ServeHTTP(w, req)

	assert.Equal(ts.T(), http.StatusBadRequest, w.Code)

	var firstResult struct {
		Error            string `json:"error"`
		ErrorDescription string `json:"error_description"`
	}

	assert.NoError(ts.T(), json.NewDecoder(w.Result().Body).Decode(&firstResult))

	assert.Equal(ts.T(), firstResult.Error, "invalid_grant")
	assert.Equal(ts.T(), firstResult.ErrorDescription, "Signed Solana message is using URI which does not use HTTPS")
}

func (ts *Web3TestSuite) TestValidationRules_URINotAllowed() {
	defer func() {
		ts.API.overrideTime = nil
	}()

	ts.API.overrideTime = func() time.Time {
		t, _ := time.Parse(time.RFC3339, "2025-03-29T00:00:00Z")
		return t
	}

	var buffer bytes.Buffer
	require.NoError(ts.T(), json.NewEncoder(&buffer).Encode(map[string]interface{}{
		"chain":     "solana",
		"message":   "linkly.green wants you to sign in with your Solana account:\nAffS6ECf7VmBUYGymseFYTxmdaKRYpeteUmAmRHpzjZo\n\nStatement\n\nURI: https://linkly.green/\nVersion: 1\nIssued At: 2025-03-29T00:00:00Z\nExpiration Time: 2025-03-29T00:10:00Z",
		"signature": "ujavoNbw0WmQyMWjRGntIo1cfGE5lImKW6A037V6GgtX4sTYevn5hRpslpnmzXH7Ro2flG/pSoB40U6y9k0fAQ==",
	}))

	req := httptest.NewRequest(http.MethodPost, "http://localhost/token?grant_type=web3", &buffer)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	ts.API.handler.ServeHTTP(w, req)

	assert.Equal(ts.T(), http.StatusBadRequest, w.Code)

	var firstResult struct {
		Error            string `json:"error"`
		ErrorDescription string `json:"error_description"`
	}

	assert.NoError(ts.T(), json.NewDecoder(w.Result().Body).Decode(&firstResult))

	assert.Equal(ts.T(), "invalid_grant", firstResult.Error)
	assert.Equal(ts.T(), "Signed Solana message is using URI which is not allowed on this server, message was signed for another app", firstResult.ErrorDescription)
}

func (ts *Web3TestSuite) TestValidationRules_URINotHTTPS() {
	defer func() {
		ts.API.overrideTime = nil
	}()

	ts.API.overrideTime = func() time.Time {
		t, _ := time.Parse(time.RFC3339, "2025-03-29T00:00:00Z")
		return t
	}

	var buffer bytes.Buffer
	require.NoError(ts.T(), json.NewEncoder(&buffer).Encode(map[string]interface{}{
		"chain":     "solana",
		"message":   "linkly.id wants you to sign in with your Solana account:\nExVRDBoePUMgVGNYLXrxHzfA9ijhT3p7aJpSeVRt3BoM\n\nStatement\n\nURI: ftp://supaabse.com\nVersion: 1\nIssued At: 2025-03-29T00:00:00Z",
		"signature": "l3slXRnD4FdjrYFOa4SPnjRTVhljWGkpNN7ziqQR3ccUCLWLEVuzeo2LacoOpDtJtyNJ9CRbcUqvGBHf+t5bAA==",
	}))

	req := httptest.NewRequest(http.MethodPost, "http://localhost/token?grant_type=web3", &buffer)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	ts.API.handler.ServeHTTP(w, req)

	assert.Equal(ts.T(), http.StatusBadRequest, w.Code)

	var firstResult struct {
		Error            string `json:"error"`
		ErrorDescription string `json:"error_description"`
	}

	assert.NoError(ts.T(), json.NewDecoder(w.Result().Body).Decode(&firstResult))

	assert.Equal(ts.T(), "invalid_grant", firstResult.Error)
	assert.Equal(ts.T(), "Signed Solana message is using URI which does not use HTTPS", firstResult.ErrorDescription)
}

func (ts *Web3TestSuite) TestValidationRules_InvalidDomain() {
	defer func() {
		ts.API.overrideTime = nil
	}()

	ts.API.overrideTime = func() time.Time {
		t, _ := time.Parse(time.RFC3339, "2025-03-29T00:00:00Z")
		return t
	}

	var buffer bytes.Buffer
	require.NoError(ts.T(), json.NewEncoder(&buffer).Encode(map[string]interface{}{
		"chain":     "solana",
		"message":   "linkly.green wants you to sign in with your Solana account:\nFNgCGD658cgLLrfm9HUoFLBiKKw7K3vpr1o1cP3cG45Z\n\nStatement\n\nURI: https://linkly.id/\nVersion: 1\nIssued At: 2025-03-29T00:00:00Z",
		"signature": "v6ro07uwz+2ZrzZ5Zb8grtxzA19UGD+7SpQnpuVVJwPPozsyNxaFmzUiKWhCKMXE9rxp8nCBerNP6fA9YQTyAg==",
	}))

	req := httptest.NewRequest(http.MethodPost, "http://localhost/token?grant_type=web3", &buffer)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	ts.API.handler.ServeHTTP(w, req)

	assert.Equal(ts.T(), http.StatusBadRequest, w.Code)

	var firstResult struct {
		Error            string `json:"error"`
		ErrorDescription string `json:"error_description"`
	}

	assert.NoError(ts.T(), json.NewDecoder(w.Result().Body).Decode(&firstResult))

	assert.Equal(ts.T(), "invalid_grant", firstResult.Error)
	assert.Equal(ts.T(), "Signed Solana message is using a Domain that does not match the one in URI which is not allowed on this server", firstResult.ErrorDescription)
}

func (ts *Web3TestSuite) TestValidationRules_MismatchedDomainAndURIHostname() {
	defer func() {
		ts.API.overrideTime = nil
	}()

	ts.API.overrideTime = func() time.Time {
		t, _ := time.Parse(time.RFC3339, "2025-03-29T00:00:00Z")
		return t
	}

	var buffer bytes.Buffer
	require.NoError(ts.T(), json.NewEncoder(&buffer).Encode(map[string]interface{}{
		"chain":     "solana",
		"message":   "linkly.green wants you to sign in with your Solana account:\n3wzL76xxbEKbzd9NFG5vJMAEvoH5YgdfpLBa6nuBYSEd\n\nStatement\n\nURI: https://linkly.id/\nVersion: 1\nIssued At: 2025-03-29T00:00:00Z\nExpiration Time: 2025-03-29T00:10:00Z",
		"signature": "vEAURX4jjxqPtWEnfWRnG2kEyT45pulE3tZZi9AjO+13jIvQweRwufGsDJyMjhL23LXoskLIop6SxF3hOebQBg==",
	}))

	req := httptest.NewRequest(http.MethodPost, "http://localhost/token?grant_type=web3", &buffer)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	ts.API.handler.ServeHTTP(w, req)

	assert.Equal(ts.T(), http.StatusBadRequest, w.Code)

	var firstResult struct {
		Error            string `json:"error"`
		ErrorDescription string `json:"error_description"`
	}

	assert.NoError(ts.T(), json.NewDecoder(w.Result().Body).Decode(&firstResult))

	assert.Equal(ts.T(), "invalid_grant", firstResult.Error)
	assert.Equal(ts.T(), "Signed Solana message is using a Domain that does not match the one in URI which is not allowed on this server", firstResult.ErrorDescription)
}

func (ts *Web3TestSuite) TestValidationRules_ValidatedBeforeNotBefore() {
	defer func() {
		ts.API.overrideTime = nil
	}()

	ts.API.overrideTime = func() time.Time {
		t, _ := time.Parse(time.RFC3339, "2025-03-29T00:00:59Z")
		return t
	}

	var buffer bytes.Buffer
	require.NoError(ts.T(), json.NewEncoder(&buffer).Encode(map[string]interface{}{
		"chain":     "solana",
		"message":   "linkly.id wants you to sign in with your Solana account:\n86Z1BGU5MY1FDBBnAhgQKWpkVmjHs15EKeGYthe9As7w\n\nStatement\n\nURI: https://linkly.id/\nVersion: 1\nIssued At: 2025-03-29T00:00:00Z\nNot Before: 2025-03-29T00:01:00Z",
		"signature": "vfGoWanxBZ5U4zr2EZUZYBUc490RogrTUGeBfVIaApcPQtv+U06fW+Fuv2/pttfZVhk1Imsf8IjpCxN+Z7RqAA==",
	}))

	req := httptest.NewRequest(http.MethodPost, "http://localhost/token?grant_type=web3", &buffer)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	ts.API.handler.ServeHTTP(w, req)

	assert.Equal(ts.T(), http.StatusBadRequest, w.Code)

	var firstResult struct {
		Error            string `json:"error"`
		ErrorDescription string `json:"error_description"`
	}

	assert.NoError(ts.T(), json.NewDecoder(w.Result().Body).Decode(&firstResult))

	assert.Equal(ts.T(), "invalid_grant", firstResult.Error)
	assert.Equal(ts.T(), "Signed Solana message becomes valid in the future", firstResult.ErrorDescription)
}

func (ts *Web3TestSuite) TestValidationRules_Expired() {
	defer func() {
		ts.API.overrideTime = nil
	}()

	ts.API.overrideTime = func() time.Time {
		t, _ := time.Parse(time.RFC3339, "2025-03-29T00:10:01Z")
		return t
	}

	var buffer bytes.Buffer
	require.NoError(ts.T(), json.NewEncoder(&buffer).Encode(map[string]interface{}{
		"chain":     "solana",
		"message":   "linkly.id wants you to sign in with your Solana account:\n5okcGJLofeQPxoNA8x8yA5roGzDzEEjbpQ13VeDxM6S4\n\nStatement\n\nURI: https://linkly.id/\nVersion: 1\nIssued At: 2025-03-29T00:00:00Z\nExpiration Time: 2025-03-29T00:10:00Z\nNot Before: 2025-03-29T00:00:00Z",
		"signature": "EOOInDU+waiHhCi0KoArOTKF5x9KzNstoA9H8h3x/fd43FgeDSCIpFicn4oAtqQ7wkJgSfAGx3lh+mCw1yuWCg==",
	}))

	req := httptest.NewRequest(http.MethodPost, "http://localhost/token?grant_type=web3", &buffer)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	ts.API.handler.ServeHTTP(w, req)

	assert.Equal(ts.T(), http.StatusBadRequest, w.Code)

	var firstResult struct {
		Error            string `json:"error"`
		ErrorDescription string `json:"error_description"`
	}

	assert.NoError(ts.T(), json.NewDecoder(w.Result().Body).Decode(&firstResult))

	assert.Equal(ts.T(), "invalid_grant", firstResult.Error)
	assert.Equal(ts.T(), "Signed Solana message is expired", firstResult.ErrorDescription)
}

func (ts *Web3TestSuite) TestValidationRules_Future() {
	defer func() {
		ts.API.overrideTime = nil
	}()

	ts.API.overrideTime = func() time.Time {
		t, _ := time.Parse(time.RFC3339, "2025-03-28T23:49:59Z")
		return t
	}

	var buffer bytes.Buffer
	require.NoError(ts.T(), json.NewEncoder(&buffer).Encode(map[string]interface{}{
		"chain":     "solana",
		"message":   "linkly.id wants you to sign in with your Solana account:\n4uyfsjxvcegi9oWgeTGNC3JA487VNXJfNLfvSKh77RzC\n\nStatement\n\nURI: https://linkly.id/\nVersion: 1\nIssued At: 2025-03-29T00:00:00Z",
		"signature": "PzGvI8up/rGokrgyycsDZPhQru6W8MwThKzkTktamoQcfYQzgTXmRpHCkI2HXDW3sSv7M2/B+xL36ksl/7KrCQ==",
	}))

	req := httptest.NewRequest(http.MethodPost, "http://localhost/token?grant_type=web3", &buffer)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	ts.API.handler.ServeHTTP(w, req)

	assert.Equal(ts.T(), http.StatusBadRequest, w.Code)

	var firstResult struct {
		Error            string `json:"error"`
		ErrorDescription string `json:"error_description"`
	}

	assert.NoError(ts.T(), json.NewDecoder(w.Result().Body).Decode(&firstResult))

	assert.Equal(ts.T(), "invalid_grant", firstResult.Error)
	assert.Equal(ts.T(), "Solana message was issued too far in the future", firstResult.ErrorDescription)
}

func (ts *Web3TestSuite) TestValidationRules_IssedTooLongAgo() {
	defer func() {
		ts.API.overrideTime = nil
	}()

	ts.API.overrideTime = func() time.Time {
		t, _ := time.Parse(time.RFC3339, "2025-03-29T00:00:00Z")
		d, _ := time.ParseDuration("10m1s")

		return t.Add(d)
	}

	var buffer bytes.Buffer
	require.NoError(ts.T(), json.NewEncoder(&buffer).Encode(map[string]interface{}{
		"chain":     "solana",
		"message":   "linkly.id wants you to sign in with your Solana account:\n5E8as7E2HdYQbzxNs182dowRHBZV5jMHQp7BK37JRLCM\n\nStatement\n\nURI: https://linkly.id/\nVersion: 1\nIssued At: 2025-03-29T00:00:00Z\nNot Before: 2025-03-29T00:00:00Z",
		"signature": "9IqeyIN09tfuP4YpR9kHCj8YG+xDIWKi8ayEdiOWKfTSfA2dHnORyl1fQR5ogkN0MzsXQlw3yX2VKCk6VdT7Cg==",
	}))

	req := httptest.NewRequest(http.MethodPost, "http://localhost/token?grant_type=web3", &buffer)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	ts.API.handler.ServeHTTP(w, req)

	assert.Equal(ts.T(), http.StatusBadRequest, w.Code)

	var firstResult struct {
		Error            string `json:"error"`
		ErrorDescription string `json:"error_description"`
	}

	assert.NoError(ts.T(), json.NewDecoder(w.Result().Body).Decode(&firstResult))

	assert.Equal(ts.T(), firstResult.Error, "invalid_grant")
	assert.Equal(ts.T(), firstResult.ErrorDescription, "Solana message was issued too long ago")
}

func (ts *Web3TestSuite) TestValidationRules_InvalidSignature() {
	defer func() {
		ts.API.overrideTime = nil
	}()

	ts.API.overrideTime = func() time.Time {
		t, _ := time.Parse(time.RFC3339, "2025-03-29T00:00:00Z")
		return t
	}

	var buffer bytes.Buffer
	require.NoError(ts.T(), json.NewEncoder(&buffer).Encode(map[string]interface{}{
		"chain":     "solana",
		"message":   "linkly.id wants you to sign in with your Solana account:\n2EZEiBdw47VHT6SpZSW9VnuSvBe7DxuYHBTxj19gxvv8\n\nStatement\n\nURI: https://linkly.id/\nVersion: 1\nIssued At: 2025-03-29T00:00:00Z\nExpiration Time: 2025-03-29T00:10:00Z\nNot Before: 2025-03-29T00:00:00Z",
		"signature": "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx==",
	}))

	req := httptest.NewRequest(http.MethodPost, "http://localhost/token?grant_type=web3", &buffer)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	ts.API.handler.ServeHTTP(w, req)

	assert.Equal(ts.T(), http.StatusBadRequest, w.Code)

	var firstResult struct {
		Error            string `json:"error"`
		ErrorDescription string `json:"error_description"`
	}

	assert.NoError(ts.T(), json.NewDecoder(w.Result().Body).Decode(&firstResult))

	assert.Equal(ts.T(), "invalid_grant", firstResult.Error)
	assert.Equal(ts.T(), "Signature does not match address in message", firstResult.ErrorDescription)
}

func (ts *Web3TestSuite) TestValidationRules_BasicValidation() {
	var buffer bytes.Buffer
	require.NoError(ts.T(), json.NewEncoder(&buffer).Encode(map[string]interface{}{
		"chain":     "solana",
		"message":   strings.Repeat(" ", 63),
		"signature": "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx==",
	}))

	req := httptest.NewRequest(http.MethodPost, "http://localhost/token?grant_type=web3", &buffer)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	ts.API.handler.ServeHTTP(w, req)

	assert.Equal(ts.T(), http.StatusBadRequest, w.Code)

	require.NoError(ts.T(), json.NewEncoder(&buffer).Encode(map[string]interface{}{
		"chain":     "solana",
		"message":   strings.Repeat(" ", 64),
		"signature": strings.Repeat("x", 85),
	}))

	req = httptest.NewRequest(http.MethodPost, "http://localhost/token?grant_type=web3", &buffer)
	req.Header.Set("Content-Type", "application/json")

	w = httptest.NewRecorder()
	ts.API.handler.ServeHTTP(w, req)

	assert.Equal(ts.T(), http.StatusBadRequest, w.Code)

	require.NoError(ts.T(), json.NewEncoder(&buffer).Encode(map[string]interface{}{
		"chain":     "solana",
		"message":   strings.Repeat(" ", 64),
		"signature": strings.Repeat("x", 89),
	}))

	req = httptest.NewRequest(http.MethodPost, "http://localhost/token?grant_type=web3", &buffer)
	req.Header.Set("Content-Type", "application/json")

	w = httptest.NewRecorder()
	ts.API.handler.ServeHTTP(w, req)

	assert.Equal(ts.T(), http.StatusBadRequest, w.Code)

	require.NoError(ts.T(), json.NewEncoder(&buffer).Encode(map[string]interface{}{
		"chain":     "solana",
		"message":   strings.Repeat(" ", 20*1024+1),
		"signature": strings.Repeat("x", 86),
	}))

	req = httptest.NewRequest(http.MethodPost, "http://localhost/token?grant_type=web3", &buffer)
	req.Header.Set("Content-Type", "application/json")

	w = httptest.NewRecorder()
	ts.API.handler.ServeHTTP(w, req)

	assert.Equal(ts.T(), http.StatusBadRequest, w.Code)

	require.NoError(ts.T(), json.NewEncoder(&buffer).Encode(map[string]interface{}{
		"chain":     "solana",
		"message":   strings.Repeat(" ", 64),
		"signature": strings.Repeat("\x00", 86),
	}))

	req = httptest.NewRequest(http.MethodPost, "http://localhost/token?grant_type=web3", &buffer)
	req.Header.Set("Content-Type", "application/json")

	w = httptest.NewRecorder()
	ts.API.handler.ServeHTTP(w, req)

	assert.Equal(ts.T(), http.StatusBadRequest, w.Code)

	require.NoError(ts.T(), json.NewEncoder(&buffer).Encode(map[string]interface{}{
		"chain":     "solana",
		"message":   strings.Repeat(" ", 64),
		"signature": strings.Repeat("x", 86),
	}))

	req = httptest.NewRequest(http.MethodPost, "http://localhost/token?grant_type=web3", &buffer)
	req.Header.Set("Content-Type", "application/json")

	w = httptest.NewRecorder()
	ts.API.handler.ServeHTTP(w, req)

	assert.Equal(ts.T(), http.StatusBadRequest, w.Code)
}
