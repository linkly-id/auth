package api

import (
	"net/http"
	"testing"

	"net/http/httptest"

	"github.com/linkly-id/auth/internal/conf"
	"github.com/linkly-id/auth/internal/hooks/hookserrors"
	"github.com/linkly-id/auth/internal/hooks/v0hooks"
	"github.com/linkly-id/auth/internal/models"
	"github.com/linkly-id/auth/internal/storage"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"gopkg.in/h2non/gock.v1"
)

var handleApiRequest func(*http.Request) (*http.Response, error)

type HooksTestSuite struct {
	suite.Suite
	API      *API
	Config   *conf.GlobalConfiguration
	TestUser *models.User
}

type MockHttpClient struct {
	mock.Mock
}

func (m *MockHttpClient) Do(req *http.Request) (*http.Response, error) {
	return handleApiRequest(req)
}

func TestHooks(t *testing.T) {
	api, config, err := setupAPIForTest()
	require.NoError(t, err)

	ts := &HooksTestSuite{
		API:    api,
		Config: config,
	}
	defer api.db.Close()

	suite.Run(t, ts)

	t.Run("CheckTX", func(t *testing.T) {
		require.NoError(t, checkTX(api.db))

		err := api.db.Transaction(func(tx *storage.Connection) error {
			require.Error(t, checkTX(tx))

			err := checkTX(tx)
			require.Error(t, err)
			return nil
		})
		require.NoError(t, err)
	})
}

func (ts *HooksTestSuite) SetupTest() {
	models.TruncateAll(ts.API.db)
	u, err := models.NewUser("123456789", "testemail@gmail.com", "securetestpassword", ts.Config.JWT.Aud, nil)
	require.NoError(ts.T(), err, "Error creating test user model")
	require.NoError(ts.T(), ts.API.db.Create(u), "Error saving new test user")
	ts.TestUser = u
}

func (ts *HooksTestSuite) TestRunHTTPHook() {
	// setup mock requests for hooks
	defer gock.OffAll()

	input := v0hooks.SendSMSInput{
		User: ts.TestUser,
		SMS: v0hooks.SMS{
			OTP: "123456",
		},
	}
	testURL := "http://localhost:54321/functions/v1/custom-sms-sender"
	ts.Config.Hook.SendSMS.URI = testURL

	unsuccessfulResponse := hookserrors.Error{
		HTTPCode: http.StatusUnprocessableEntity,
		Message:  "test error",
	}

	testCases := []struct {
		description  string
		expectError  bool
		mockResponse hookserrors.Error
	}{
		{
			description:  "Hook returns success",
			expectError:  false,
			mockResponse: hookserrors.Error{},
		},
		{
			description:  "Hook returns error",
			expectError:  true,
			mockResponse: unsuccessfulResponse,
		},
	}

	gock.New(ts.Config.Hook.SendSMS.URI).
		Post("/").
		MatchType("json").
		Reply(http.StatusOK).
		JSON(v0hooks.SendSMSOutput{})

	gock.New(ts.Config.Hook.SendSMS.URI).
		Post("/").
		MatchType("json").
		Reply(http.StatusUnprocessableEntity).
		JSON(struct {
			Error *hookserrors.Error `json:"error,omitempty"`
		}{Error: &unsuccessfulResponse})

	for _, tc := range testCases {
		ts.Run(tc.description, func() {
			req, _ := http.NewRequest("POST", ts.Config.Hook.SendSMS.URI, nil)

			var output v0hooks.SendSMSOutput
			err := ts.API.hooksMgr.InvokeHook(ts.API.db, req, &input, &output)

			if !tc.expectError {
				require.NoError(ts.T(), err)
			} else {
				require.Error(ts.T(), err)
				require.Equal(ts.T(), output, v0hooks.SendSMSOutput{})
			}
		})
	}
	require.True(ts.T(), gock.IsDone())
}

func (ts *HooksTestSuite) TestShouldRetryWithRetryAfterHeader() {
	defer gock.OffAll()

	input := v0hooks.SendSMSInput{
		User: ts.TestUser,
		SMS: v0hooks.SMS{
			OTP: "123456",
		},
	}
	testURL := "http://localhost:54321/functions/v1/custom-sms-sender"
	ts.Config.Hook.SendSMS.URI = testURL

	gock.New(testURL).
		Post("/").
		MatchType("json").
		Reply(http.StatusTooManyRequests).
		SetHeader("retry-after", "true").SetHeader("content-type", "application/json")

	// Simulate an additional response for the retry attempt
	gock.New(testURL).
		Post("/").
		MatchType("json").
		Reply(http.StatusOK).
		JSON(v0hooks.SendSMSOutput{}).SetHeader("content-type", "application/json")

	// Simulate the original HTTP request which triggered the hook
	req, err := http.NewRequest("POST", "http://localhost:9998/otp", nil)
	require.NoError(ts.T(), err)

	var output v0hooks.SendSMSOutput
	err = ts.API.hooksMgr.InvokeHook(ts.API.db, req, &input, &output)
	require.NoError(ts.T(), err)

	// Ensure that all expected HTTP interactions (mocks) have been called
	require.True(ts.T(), gock.IsDone(), "Expected all mocks to have been called including retry")
}

func (ts *HooksTestSuite) TestShouldReturnErrorForNonJSONContentType() {
	defer gock.OffAll()

	input := v0hooks.SendSMSInput{
		User: ts.TestUser,
		SMS: v0hooks.SMS{
			OTP: "123456",
		},
	}
	testURL := "http://localhost:54321/functions/v1/custom-sms-sender"
	ts.Config.Hook.SendSMS.URI = testURL

	gock.New(testURL).
		Post("/").
		MatchType("json").
		Reply(http.StatusOK).
		SetHeader("content-type", "text/plain")

	req, err := http.NewRequest("POST", "http://localhost:9999/otp", nil)
	require.NoError(ts.T(), err)

	var output v0hooks.SendSMSOutput
	err = ts.API.hooksMgr.InvokeHook(ts.API.db, req, &input, &output)
	require.Error(ts.T(), err, "Expected an error due to wrong content type")
	require.Contains(ts.T(), err.Error(), "Invalid JSON response.")
	require.True(ts.T(), gock.IsDone(), "Expected all mocks to have been called")
}

func (ts *HooksTestSuite) TestInvokeHookIntegration() {
	// We use the Send Email Hook as illustration
	defer gock.OffAll()
	hookFunctionSQL := `
        create or replace function invoke_test(input jsonb)
        returns json as $$
        begin
            return input;
        end; $$ language plpgsql;`
	require.NoError(ts.T(), ts.API.db.RawQuery(hookFunctionSQL).Exec())

	testHTTPUri := "http://myauthservice.com/signup"
	testHTTPSUri := "https://myauthservice.com/signup"
	testPGUri := "pg-functions://postgres/auth/invoke_test"
	successOutput := map[string]interface{}{}
	authEndpoint := "https://app.myapp.com/otp"
	gock.New(testHTTPUri).
		Post("/").
		MatchType("json").
		Reply(http.StatusOK).
		JSON(successOutput).SetHeader("content-type", "application/json")

	gock.New(testHTTPSUri).
		Post("/").
		MatchType("json").
		Reply(http.StatusOK).
		JSON(successOutput).SetHeader("content-type", "application/json")

	tests := []struct {
		description   string
		conn          *storage.Connection
		request       *http.Request
		input         any
		output        any
		uri           string
		expectedError error
	}{
		{
			description: "HTTP endpoint success",
			conn:        nil,
			request:     httptest.NewRequest("POST", authEndpoint, nil),
			input:       &v0hooks.SendEmailInput{},
			output:      &v0hooks.SendEmailOutput{},
			uri:         testHTTPUri,
		},
		{
			description: "HTTPS endpoint success",
			conn:        nil,
			request:     httptest.NewRequest("POST", authEndpoint, nil),
			input:       &v0hooks.SendEmailInput{},
			output:      &v0hooks.SendEmailOutput{},
			uri:         testHTTPSUri,
		},
		{
			description: "PostgreSQL function success",
			conn:        ts.API.db,
			request:     httptest.NewRequest("POST", authEndpoint, nil),
			input:       &v0hooks.SendEmailInput{},
			output:      &v0hooks.SendEmailOutput{},
			uri:         testPGUri,
		},
		{
			description:   "Unsupported protocol error",
			conn:          nil,
			request:       httptest.NewRequest("POST", authEndpoint, nil),
			input:         &v0hooks.SendEmailInput{},
			output:        &v0hooks.SendEmailOutput{},
			uri:           "ftp://example.com/path",
			expectedError: errors.New("unsupported protocol: \"ftp://example.com/path\" only postgres hooks and HTTPS functions are supported at the moment"),
		},
	}

	var err error
	for _, tc := range tests {
		// Set up hook config
		ts.Config.Hook.SendEmail.Enabled = true
		ts.Config.Hook.SendEmail.URI = tc.uri
		require.NoError(ts.T(), ts.Config.Hook.SendEmail.PopulateExtensibilityPoint())

		ts.Run(tc.description, func() {
			err = ts.API.hooksMgr.InvokeHook(tc.conn, tc.request, tc.input, tc.output)
			if tc.expectedError != nil {
				require.EqualError(ts.T(), err, tc.expectedError.Error())
			} else {
				require.NoError(ts.T(), err)
			}
		})

	}
	// Ensure that all expected HTTP interactions (mocks) have been called
	require.True(ts.T(), gock.IsDone(), "Expected all mocks to have been called including retry")
}
