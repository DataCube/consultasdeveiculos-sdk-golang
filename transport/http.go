package transport

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	sdkErrors "github.com/DataCube/consultasdeveiculos-sdk-golang/errors"
	"github.com/DataCube/consultasdeveiculos-sdk-golang/parser"
)

type HttpTransport struct {
	options TransportOptions
	client  *http.Client
}

func NewHttpTransport(opts TransportOptions) *HttpTransport {
	tr := &http.Transport{
		DisableCompression: opts.DisableCompression,
	}
	return &HttpTransport{
		options: opts,
		client: &http.Client{
			Transport: tr,
		},
	}
}

func (t *HttpTransport) Request(endpoint parser.Endpoint, params map[string]interface{}, options RequestOptions) (*Response, error) {
	urlStr := BuildUrl(endpoint.URL, t.options.BaseURL, params)
	headers := BuildHeaders(t.options.Headers, endpoint.Headers, params)
	body := BuildBody(endpoint.Body, params)
	method := "GET"
	if endpoint.Method != "" {
		method = strings.ToUpper(endpoint.Method)
	}

	// Add auth_token to body
	if t.options.Token != "" {
		if body == nil {
			body = make(map[string]interface{})
		}
		// Merge token
		newBody := map[string]interface{}{
			"auth_token": t.options.Token,
		}
		for k, v := range body {
			if k == "auth_token" && v == "{{api_token}}" {
				continue
			}
			newBody[k] = v
		}
		body = newBody
	}

	if !t.options.DisableCompression {
		headers["Accept-Encoding"] = "gzip, deflate, br"
	}

	var jsonBytes []byte
	if body != nil && (method == "POST" || method == "PUT" || method == "PATCH" || method == "DELETE") {
		var err error
		jsonBytes, err = json.Marshal(body)
		if err != nil {
			return nil, err
		}
	}

	// Timeout configuration
	timeout := t.options.Timeout
	if options.Timeout != 0 {
		timeout = options.Timeout
	}

	// Retries configuration
	maxRetries := t.options.MaxRetries
	if options.MaxRetries != nil {
		maxRetries = *options.MaxRetries
	}

	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		var bodyReader io.Reader
		if len(jsonBytes) > 0 {
			bodyReader = bytes.NewReader(jsonBytes)
		}

		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		req, err := http.NewRequestWithContext(ctx, method, urlStr, bodyReader)
		if err != nil {
			cancel()
			return nil, err
		}

		for k, v := range headers {
			req.Header.Set(k, v)
		}

		resp, err := t.client.Do(req)
		if err != nil {
			cancel()
			lastErr = err
			// Backoff sleep
			if attempt < maxRetries {
				delay := time.Duration(float64(t.options.RetryDelay) * math.Pow(2, float64(attempt)))
				time.Sleep(delay)
				continue
			}
			break
		}

		res, errHandler := t.handleResponse(resp)
		cancel()

		if errHandler != nil {
			lastErr = errHandler

			// Do not retry for Auth or Validation errors
			_, isAuth := errHandler.(*sdkErrors.AuthenticationError)
			_, isVal := errHandler.(*sdkErrors.ValidationError)
			if isAuth || isVal {
				return nil, errHandler
			}

			// Rate limit error - sleep retryAfter
			if rlErr, ok := errHandler.(*sdkErrors.RateLimitError); ok && rlErr.RetryAfter > 0 {
				if attempt < maxRetries {
					time.Sleep(time.Duration(rlErr.RetryAfter) * time.Second)
					continue
				}
			}

			if attempt < maxRetries {
				delay := time.Duration(float64(t.options.RetryDelay) * math.Pow(2, float64(attempt)))
				time.Sleep(delay)
				continue
			}
			break
		}

		return res, nil
	}

	if lastErr != nil {
		return nil, lastErr
	}
	return nil, sdkErrors.NewSDKError("Falha na requisição após múltiplas tentativas", "SDK_ERROR", nil)
}

func (t *HttpTransport) handleResponse(resp *http.Response) (*Response, error) {
	defer resp.Body.Close()

	contentType := resp.Header.Get("Content-Type")
	var data interface{}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if strings.Contains(contentType, "application/json") {
		var parsed interface{}
		if err := json.Unmarshal(bodyBytes, &parsed); err == nil {
			data = parsed
		} else {
			data = string(bodyBytes)
		}
	} else {
		data = string(bodyBytes)
	}

	headers := make(map[string]string)
	for k, v := range resp.Header {
		if len(v) > 0 {
			headers[k] = v[0]
		}
	}

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return &Response{
			Success: true,
			Status:  resp.StatusCode,
			Data:    data,
			Headers: headers,
		}, nil
	}

	var message string = fmt.Sprintf("Erro HTTP %d", resp.StatusCode)
	var details map[string]interface{}

	if m, ok := data.(map[string]interface{}); ok {
		if msg, ok := m["message"].(string); ok {
			message = msg
		} else if msg, ok := m["msg"].(string); ok {
			message = msg
		}
		details = m
	} else {
		details = map[string]interface{}{
			"raw": data,
		}
	}

	switch resp.StatusCode {
	case 401, 403:
		return nil, sdkErrors.NewAuthenticationError(message, details)
	case 400, 422:
		return nil, sdkErrors.NewValidationError(message, details)
	case 429:
		retryAfterSecs := 60
		if val, ok := headers["retry-after"]; ok {
			if parsedVal, err := strconv.Atoi(val); err == nil {
				retryAfterSecs = parsedVal
			}
		}
		return nil, sdkErrors.NewRateLimitError(message, retryAfterSecs, details)
	case 404:
		return nil, sdkErrors.NewSDKError(message, "NOT_FOUND", details)
	case 500, 502, 503, 504:
		return nil, sdkErrors.NewSDKError(message, "SERVER_ERROR", details)
	default:
		return nil, sdkErrors.NewSDKError(message, "HTTP_ERROR", details)
	}
}
