package yamlintegration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// HTTPResponse envuelve el status code y el body raw de una respuesta HTTP.
type HTTPResponse struct {
	StatusCode int
	Body       []byte
}

// httpClientTimeout devuelve el timeout para las peticiones de integración.
// Por defecto 30 segundos. Se sobrescribe con la variable de entorno
// INTEGRATION_HTTP_TIMEOUT (formato time.ParseDuration, ej. "90s", "3m").
func httpClientTimeout() time.Duration {
	if s := strings.TrimSpace(os.Getenv("INTEGRATION_HTTP_TIMEOUT")); s != "" {
		if d, err := time.ParseDuration(s); err == nil && d > 0 {
			return d
		}
	}
	return 30 * time.Second
}

// ExecuteHTTPRequest realiza una petición HTTP real y devuelve la respuesta.
func ExecuteHTTPRequest(baseURL, method, endpoint string, headers map[string]string, body map[string]interface{}) (*HTTPResponse, error) {
	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reqBody = bytes.NewBuffer(data)
	}

	url := fmt.Sprintf("%s%s", strings.TrimSuffix(baseURL, "/"), endpoint)
	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	client := &http.Client{Timeout: httpClientTimeout()}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return &HTTPResponse{
		StatusCode: resp.StatusCode,
		Body:       respData,
	}, nil
}
