package transport

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"regexp"
	"strings"
	"time"

	sdkErrors "github.com/DataCube/consultasdeveiculos-sdk-golang/errors"
	"github.com/DataCube/consultasdeveiculos-sdk-golang/parser"
)

type SandboxTransport struct {
	options TransportOptions
}

func NewSandboxTransport(opts TransportOptions) *SandboxTransport {
	if opts.SandboxDelay == 0 {
		opts.SandboxDelay = 100 * time.Millisecond
	}
	return &SandboxTransport{
		options: opts,
	}
}

func (t *SandboxTransport) Request(endpoint parser.Endpoint, params map[string]interface{}, options RequestOptions) (*Response, error) {
	t.simulateDelay()

	if t.options.SandboxRandomErrors && rand.Float64() < t.options.SandboxErrorRate {
		return nil, sdkErrors.NewSDKError(
			"Erro simulado de sandbox",
			"SANDBOX_SIMULATED_ERROR",
			map[string]interface{}{"endpoint": endpoint.Key},
		)
	}

	return t.findExampleResponse(endpoint, params)
}

func (t *SandboxTransport) simulateDelay() {
	if t.options.SandboxDelay <= 0 {
		return
	}
	jitter := rand.Float64() * float64(t.options.SandboxDelay) * 0.5
	totalDelay := float64(t.options.SandboxDelay) + jitter
	time.Sleep(time.Duration(totalDelay))
}

func (t *SandboxTransport) findExampleResponse(endpoint parser.Endpoint, params map[string]interface{}) (*Response, error) {
	responses := endpoint.Responses

	if len(responses) == 0 {
		data := map[string]interface{}{
			"success":  true,
			"message":  fmt.Sprintf("Sandbox response for %s", endpoint.Name),
			"endpoint": endpoint.Key,
			"params":   sanitizeParams(params),
		}
		return &Response{
			Success:  true,
			Status:   200,
			Data:     data,
			Headers:  map[string]string{},
			Sandbox:  true,
			Endpoint: endpoint.Key,
		}, nil
	}

	var selectedResp *parser.ResponseExample
	for i := range responses {
		if responses[i].Status >= 200 && responses[i].Status < 300 {
			selectedResp = &responses[i]
			break
		}
	}

	if selectedResp == nil {
		selectedResp = &responses[0]
	}

	data := selectedResp.Body
	if str, ok := data.(string); ok {
		var parsed interface{}
		if err := json.Unmarshal([]byte(str), &parsed); err == nil {
			data = parsed
		}
	}

	data = replacePlaceholders(data, params)

	return &Response{
		Success:  true,
		Status:   selectedResp.Status,
		Data:     data,
		Headers:  selectedResp.Headers,
		Sandbox:  true,
		Endpoint: endpoint.Key,
	}, nil
}

func replacePlaceholders(data interface{}, params map[string]interface{}) interface{} {
	switch val := data.(type) {
	case string:
		return replaceInString(val, params)
	case []interface{}:
		res := make([]interface{}, len(val))
		for i, item := range val {
			res[i] = replacePlaceholders(item, params)
		}
		return res
	case map[string]interface{}:
		res := make(map[string]interface{})
		for k, v := range val {
			res[k] = replacePlaceholders(v, params)
		}
		return res
	default:
		return val
	}
}

func replaceInString(str string, params map[string]interface{}) string {
	allParams := make(map[string]interface{})

	for k, v := range params {
		if k != "path" && k != "query" && k != "headers" && k != "body" {
			allParams[k] = v
		}
	}

	if b, ok := params["body"]; ok {
		if m, ok := b.(map[string]interface{}); ok {
			for k, v := range m {
				allParams[k] = v
			}
		}
	}
	if q, ok := params["query"]; ok {
		if m, ok := q.(map[string]interface{}); ok {
			for k, v := range m {
				allParams[k] = v
			}
		}
	}
	if p, ok := params["path"]; ok {
		if m, ok := p.(map[string]interface{}); ok {
			for k, v := range m {
				allParams[k] = v
			}
		}
	}

	re := regexp.MustCompile(`\{\{(\w+)\}\}`)
	return re.ReplaceAllStringFunc(str, func(match string) string {
		key := match[2 : len(match)-2]
		if v, exists := allParams[key]; exists {
			return fmt.Sprintf("%v", v)
		}
		return match
	})
}

func sanitizeParams(params map[string]interface{}) map[string]interface{} {
	sanitized := make(map[string]interface{})
	for k, v := range params {
		if k == "headers" {
			if m, ok := v.(map[string]interface{}); ok {
				newH := make(map[string]interface{})
				for hk, hv := range m {
					if strings.ToLower(hk) == "authorization" {
						continue
					}
					newH[hk] = hv
				}
				sanitized[k] = newH
			} else if m, ok := v.(map[string]string); ok {
				newH := make(map[string]string)
				for hk, hv := range m {
					if strings.ToLower(hk) == "authorization" {
						continue
					}
					newH[hk] = hv
				}
				sanitized[k] = newH
			} else {
				sanitized[k] = v
			}
		} else {
			sanitized[k] = v
		}
	}
	return sanitized
}
