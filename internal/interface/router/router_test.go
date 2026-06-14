package router

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/kazumadev619-dev/fishing-api/internal/interface/handler"
	jwtutil "github.com/kazumadev619-dev/fishing-api/pkg/jwtutil"
	"github.com/stretchr/testify/assert"
)

// buildRouter はルート登録の検証用に最小構成のルーターを組み立てる。
// 各ハンドラーは usecase に依存せずルート登録のみ検証するため nil 依存で生成する。
func buildRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	handlers := &Handlers{
		Auth:     handler.NewAuthHandler(nil),
		Weather:  handler.NewWeatherHandler(nil),
		Tide:     handler.NewTideHandler(nil),
		Location: handler.NewLocationHandler(nil),
		Favorite: handler.NewFavoriteHandler(nil),
	}
	jwtManager := jwtutil.NewManager("access-secret", "refresh-secret")
	return New(handlers, jwtManager)
}

func TestNew_RegistersAllRoutes(t *testing.T) {
	r := buildRouter()

	got := make(map[string]bool)
	for _, ri := range r.Routes() {
		got[ri.Method+" "+ri.Path] = true
	}

	want := []string{
		"GET /health",
		"POST /api/auth/register",
		"POST /api/auth/login",
		"POST /api/auth/refresh",
		"GET /api/auth/verify-email",
		"GET /api/weather",
		"GET /api/conditions/tide",
		"GET /api/locations/search",
		"GET /api/favorites",
		"POST /api/favorites",
		"DELETE /api/favorites/:id",
	}
	for _, route := range want {
		assert.True(t, got[route], "route %q should be registered", route)
	}
	assert.Len(t, r.Routes(), len(want), "登録ルート数が想定と一致すること")
}

func TestNew_HealthEndpoint(t *testing.T) {
	r := buildRouter()

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/health", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestNew_ProtectedRouteRequiresAuth(t *testing.T) {
	r := buildRouter()

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/favorites", nil)
	r.ServeHTTP(w, req)

	// JWT ミドルウェアがトークン無しのリクエストを 401 で弾くこと
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
