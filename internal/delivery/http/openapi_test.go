package http

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestOpenAPIRouteServesSpec(t *testing.T) {
	router := gin.New()
	RegisterOpenAPI(router)

	req := httptest.NewRequest(http.MethodGet, "/openapi.yaml", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusOK, resp.Code)
	require.Equal(t, "application/yaml", resp.Header().Get("Content-Type"))
	require.Equal(t, openapiSpec, resp.Body.Bytes())
}

func TestSwaggerRouteServesUI(t *testing.T) {
	router := gin.New()
	RegisterOpenAPI(router)

	req := httptest.NewRequest(http.MethodGet, "/swagger", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusOK, resp.Code)
	require.Equal(t, "text/html; charset=utf-8", resp.Header().Get("Content-Type"))
	body := resp.Body.String()
	require.Contains(t, body, "SwaggerUIBundle")
	require.Contains(t, body, "/openapi.yaml")
	require.Contains(t, body, `<div id="swagger-ui"></div>`)
}
