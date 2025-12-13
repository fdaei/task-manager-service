package http

import (
	_ "embed"
	nethttp "net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

var openapiSpec []byte

const swaggerUIPage = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <title>Task Service API Docs</title>
  <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5/swagger-ui.css">
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
  <script src="https://cdn.jsdelivr.net/npm/swagger-ui-dist@5/swagger-ui-standalone-preset.js"></script>
  <script>
    window.onload = () => {
      SwaggerUIBundle({
        url: '/openapi.yaml',
        dom_id: '#swagger-ui',
        presets: [
          SwaggerUIBundle.presets.apis,
          SwaggerUIStandalonePreset,
        ],
      });
    };
  </script>
</body>
</html>`

func RegisterOpenAPI(router *gin.Engine) {
	router.GET("/openapi.yaml", func(c *gin.Context) {
		spec := openapiSpec
		if len(spec) == 0 {
			if data, err := os.ReadFile(filepath.Join("internal", "delivery", "http", "openapi.yaml")); err == nil {
				spec = data
			}
		}
		c.Data(nethttp.StatusOK, "application/yaml", spec)
	})
	router.GET("/swagger", serveSwaggerUI)
	router.GET("/swagger/*any", serveSwaggerUI)
}

func serveSwaggerUI(c *gin.Context) {
	c.Data(nethttp.StatusOK, "text/html; charset=utf-8", []byte(swaggerUIPage))
}
