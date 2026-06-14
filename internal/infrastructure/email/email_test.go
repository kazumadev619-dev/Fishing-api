package email

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestClient は resend の BaseURL をテストサーバーに差し替えた EmailClient を返す
func newTestClient(t *testing.T, serverURL string) *EmailClient {
	t.Helper()
	ec := NewEmailClient("test-api-key", "noreply@example.com")
	base, err := url.Parse(serverURL + "/")
	require.NoError(t, err)
	ec.client.BaseURL = base
	return ec
}

func TestSendVerificationEmail_Success(t *testing.T) {
	var captured struct {
		From    string   `json:"from"`
		To      []string `json:"to"`
		Subject string   `json:"subject"`
		Html    string   `json:"html"`
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/emails", r.URL.Path)
		assert.Equal(t, "Bearer test-api-key", r.Header.Get("Authorization"))

		body, _ := io.ReadAll(r.Body)
		require.NoError(t, json.Unmarshal(body, &captured))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"email-123"}`))
	}))
	defer srv.Close()

	ec := newTestClient(t, srv.URL)
	err := ec.SendVerificationEmail("user@example.com", "tok-abc", "https://app.example.com")
	require.NoError(t, err)

	assert.Equal(t, "noreply@example.com", captured.From)
	assert.Equal(t, []string{"user@example.com"}, captured.To)
	assert.Contains(t, captured.Subject, "メールアドレスの確認")
	// 検証URLにトークンとベースURLが正しく組み込まれていること
	assert.Contains(t, captured.Html, "https://app.example.com/api/auth/verify-email?token=tok-abc")
}

func TestSendVerificationEmail_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{"statusCode":422,"message":"invalid from address","name":"validation_error"}`))
	}))
	defer srv.Close()

	ec := newTestClient(t, srv.URL)
	err := ec.SendVerificationEmail("user@example.com", "tok", "https://app.example.com")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "sending verification email to user@example.com")
}

func TestNewEmailClient_SetsFields(t *testing.T) {
	ec := NewEmailClient("key-xyz", "from@example.com")
	require.NotNil(t, ec)
	assert.Equal(t, "from@example.com", ec.fromAddress)
	require.NotNil(t, ec.client)
	// デフォルト BaseURL は resend 本番エンドポイント
	assert.True(t, strings.Contains(ec.client.BaseURL.String(), "resend.com"))
}
