package transport

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/DataCube/consultasdeveiculos-sdk-golang/parser"
)

type Response struct {
	Success  bool              `json:"success"`
	Status   int               `json:"status"`
	Data     interface{}       `json:"data"`
	Headers  map[string]string `json:"headers"`
	Sandbox  bool              `json:"sandbox,omitempty"`
	Endpoint string            `json:"endpoint,omitempty"`
}

type RequestOptions struct {
	Timeout    time.Duration
	MaxRetries *int
}

type Transport interface {
	Request(endpoint parser.Endpoint, params map[string]interface{}, options RequestOptions) (*Response, error)
}

type TransportOptions struct {
	Token              string
	BaseURL            string
	Timeout            time.Duration
	MaxRetries         int
	RetryDelay         time.Duration
	Headers            map[string]string
	DisableCompression bool
	SandboxDelay       time.Duration
	SandboxRandomErrors bool
	SandboxErrorRate    float64
}

func BuildUrl(endpointURL string, baseURL string, params map[string]interface{}) string {
	urlStr := endpointURL

	// Path params
	var pathParams map[string]interface{}
	if p, ok := params["path"]; ok {
		if m, ok := p.(map[string]interface{}); ok {
			pathParams = m
		}
	}

	for k, v := range pathParams {
		strVal := fmt.Sprintf("%v", v)
		escaped := url.QueryEscape(strVal)
		urlStr = strings.ReplaceAll(urlStr, "{{"+k+"}}", escaped)

		re := regexp.MustCompile(":" + regexp.QuoteMeta(k) + `(?=/|$)`)
		urlStr = re.ReplaceAllString(urlStr, escaped)
	}

	// Query params
	var queryParams map[string]interface{}
	if q, ok := params["query"]; ok {
		if m, ok := q.(map[string]interface{}); ok {
			queryParams = m
		}
	}

	if len(queryParams) > 0 {
		qVals := url.Values{}
		for k, v := range queryParams {
			if v != nil {
				qVals.Set(k, fmt.Sprintf("%v", v))
			}
		}
		qStr := qVals.Encode()
		if qStr != "" {
			if strings.Contains(urlStr, "?") {
				urlStr += "&" + qStr
			} else {
				urlStr += "?" + qStr
			}
		}
	}

	// Apply baseURL if urlStr is not absolute
	if baseURL != "" && !strings.HasPrefix(urlStr, "http") {
		urlStr = strings.TrimSuffix(baseURL, "/") + "/" + strings.TrimPrefix(urlStr, "/")
	}

	return urlStr
}

func BuildHeaders(defaultHeaders map[string]string, endpointHeaders map[string]string, params map[string]interface{}) map[string]string {
	res := make(map[string]string)
	for k, v := range defaultHeaders {
		res[k] = v
	}
	for k, v := range endpointHeaders {
		res[k] = v
	}
	if h, ok := params["headers"]; ok {
		if m, ok := h.(map[string]string); ok {
			for k, v := range m {
				res[k] = v
			}
		} else if m, ok := h.(map[string]interface{}); ok {
			for k, v := range m {
				res[k] = fmt.Sprintf("%v", v)
			}
		}
	}
	return res
}

func BuildBody(endpointBody interface{}, params map[string]interface{}) map[string]interface{} {
	var bodyData map[string]interface{}

	if b, ok := params["body"]; ok {
		if m, ok := b.(map[string]interface{}); ok {
			bodyData = make(map[string]interface{})
			for k, v := range m {
				bodyData[k] = v
			}
		}
	} else {
		bodyData = make(map[string]interface{})
		for k, v := range params {
			if k == "path" || k == "query" || k == "headers" {
				continue
			}
			bodyData[k] = v
		}
	}

	if endpointBody != nil {
		if epMap, ok := endpointBody.(map[string]interface{}); ok {
			merged := make(map[string]interface{})
			for k, v := range epMap {
				merged[k] = v
			}
			for k, v := range bodyData {
				merged[k] = v
			}
			return merged
		}
	}

	if len(bodyData) > 0 {
		return bodyData
	}
	return nil
}
