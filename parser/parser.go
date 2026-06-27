package parser

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

// PostmanCollection represents the Postman Collection v2.1 structure.
type PostmanCollection struct {
	Info     PostmanInfo       `json:"info"`
	Item     []PostmanItem     `json:"item"`
	Variable []PostmanVariable `json:"variable"`
}

type PostmanInfo struct {
	PostmanID   string `json:"_postman_id"`
	Name        string `json:"name"`
	Schema      string `json:"schema"`
	ExporterID  string `json:"_exporter_id"`
	Description string `json:"description"`
	Version     string `json:"version"`
}

type PostmanVariable struct {
	Key   string      `json:"key"`
	Value interface{} `json:"value"`
	Type  string      `json:"type"`
}

type PostmanItem struct {
	Name        string            `json:"name"`
	Description interface{}       `json:"description"`
	Request     *PostmanRequest   `json:"request"`
	Item        []PostmanItem     `json:"item"`
	Response    []PostmanResponse `json:"response"`
}

type PostmanRequest struct {
	Method      string          `json:"method"`
	Header      []PostmanHeader `json:"header"`
	Body        *PostmanBody    `json:"body"`
	URL         interface{}     `json:"url"` // string or PostmanURL
	Description interface{}     `json:"description"`
	Auth        interface{}     `json:"auth"`
}

type PostmanHeader struct {
	Key      string `json:"key"`
	Value    string `json:"value"`
	Disabled bool   `json:"disabled"`
}

type PostmanBody struct {
	Mode       string                 `json:"mode"`
	Raw        string                 `json:"raw"`
	Urlencoded []PostmanUrlencoded    `json:"urlencoded"`
	Formdata   []PostmanFormdata      `json:"formdata"`
	GraphQL    map[string]interface{} `json:"graphql"`
}

type PostmanUrlencoded struct {
	Key      string `json:"key"`
	Value    string `json:"value"`
	Disabled bool   `json:"disabled"`
}

type PostmanFormdata struct {
	Key      string `json:"key"`
	Value    string `json:"value"`
	Type     string `json:"type"`
	Src      string `json:"src"`
	Disabled bool   `json:"disabled"`
}

type PostmanURL struct {
	Raw      string               `json:"raw"`
	Protocol string               `json:"protocol"`
	Host     interface{}          `json:"host"` // string or []string
	Path     interface{}          `json:"path"` // string or []string
	Port     string               `json:"port"`
	Variable []PostmanURLVariable `json:"variable"`
}

type PostmanURLVariable struct {
	Key         string `json:"key"`
	Value       string `json:"value"`
	Description string `json:"description"`
}

type PostmanResponse struct {
	Name   string          `json:"name"`
	Code   int             `json:"code"`
	Header []PostmanHeader `json:"header"`
	Body   string          `json:"body"`
}

// Endpoint represents a parsed, flattened API endpoint.
type Endpoint struct {
	Key            string            `json:"key"`
	Name           string            `json:"name"`
	Namespace      string            `json:"namespace"`
	Method         string            `json:"method"`
	URL            string            `json:"url"`
	Headers        map[string]string `json:"headers"`
	Body           interface{}       `json:"body"`
	Auth           interface{}       `json:"auth"`
	Description    string            `json:"description"`
	Responses      []ResponseExample `json:"responses"`
	Variables      []VariableExample `json:"variables"`
	CollectionName string            `json:"collectionName,omitempty"`
	CollectionID   string            `json:"collectionId,omitempty"`
	Raw            interface{}       `json:"raw,omitempty"`
}

type ResponseExample struct {
	Name    string            `json:"name"`
	Status  int               `json:"status"`
	Headers map[string]string `json:"headers"`
	Body    interface{}       `json:"body"`
}

type VariableExample struct {
	Key         string `json:"key"`
	Value       string `json:"value"`
	Description string `json:"description"`
}

// PostmanParser parses a Postman collection.
type PostmanParser struct{}

func NewPostmanParser() *PostmanParser {
	return &PostmanParser{}
}

// Parse converts a PostmanCollection into a slice of Endpoints.
func (p *PostmanParser) Parse(collection *PostmanCollection) []Endpoint {
	if collection == nil {
		return nil
	}

	var endpoints []Endpoint
	for _, item := range collection.Item {
		if isFolder(item) {
			endpoints = append(endpoints, parseFolder(item, "")...)
		} else if isRequest(item) {
			endpoints = append(endpoints, parseRequest(item, ""))
		}
	}

	// Apply post processing (global variables replacement)
	return postProcess(endpoints, collection)
}

func isFolder(item PostmanItem) bool {
	return len(item.Item) > 0
}

func isRequest(item PostmanItem) bool {
	return item.Request != nil
}

func parseFolder(folder PostmanItem, parentNamespace string) []Endpoint {
	var endpoints []Endpoint
	ns := buildNamespace(folder.Name, parentNamespace)

	for _, item := range folder.Item {
		if isFolder(item) {
			endpoints = append(endpoints, parseFolder(item, ns)...)
		} else if isRequest(item) {
			endpoints = append(endpoints, parseRequest(item, ns))
		}
	}

	return endpoints
}

func buildNamespace(folderName string, parentNamespace string) string {
	normalized := normalizeFolderName(folderName)
	if normalized == "" {
		return parentNamespace
	}
	if parentNamespace == "" {
		return normalized
	}
	return parentNamespace + "." + normalized
}

func removeAccents(s string) string {
	var buf strings.Builder
	accents := map[rune]rune{
		'á': 'a', 'à': 'a', 'ã': 'a', 'â': 'a', 'ä': 'a',
		'é': 'e', 'è': 'e', 'ê': 'e', 'ë': 'e',
		'í': 'i', 'ì': 'i', 'î': 'i', 'ï': 'i',
		'ó': 'o', 'ò': 'o', 'õ': 'o', 'ô': 'o', 'ö': 'o',
		'ú': 'u', 'ù': 'u', 'û': 'u', 'ü': 'u',
		'ç': 'c', 'ñ': 'n',
		'Á': 'A', 'À': 'A', 'Ã': 'A', 'Â': 'A', 'Ä': 'A',
		'É': 'E', 'È': 'E', 'Ê': 'E', 'Ë': 'E',
		'Í': 'I', 'Ì': 'I', 'Î': 'I', 'Ï': 'I',
		'Ó': 'O', 'Ò': 'O', 'Õ': 'O', 'Ô': 'O', 'Ö': 'O',
		'Ú': 'U', 'Ù': 'U', 'Û': 'U', 'Ü': 'U',
		'Ç': 'C', 'Ñ': 'N',
	}
	for _, r := range s {
		if repl, ok := accents[r]; ok {
			buf.WriteRune(repl)
		} else {
			buf.WriteRune(r)
		}
	}
	return buf.String()
}

func normalizeFolderName(name string) string {
	s := removeAccents(name)

	var clean strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || unicode.IsSpace(r) || r == '_' {
			clean.WriteRune(r)
		}
	}

	words := strings.Fields(clean.String())
	if len(words) == 0 {
		return ""
	}

	var res strings.Builder
	for i, w := range words {
		if i == 0 {
			res.WriteString(strings.ToLower(w))
		} else {
			if len(w) > 0 {
				res.WriteString(strings.ToUpper(w[:1]) + strings.ToLower(w[1:]))
			}
		}
	}

	return res.String()
}

func normalizeMethodName(name string) string {
	var clean strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || unicode.IsSpace(r) || r == '_' {
			clean.WriteRune(r)
		}
	}

	words := strings.Fields(clean.String())
	if len(words) == 0 {
		return ""
	}

	var res strings.Builder
	for i, w := range words {
		if i == 0 {
			res.WriteString(strings.ToLower(w))
		} else {
			if len(w) > 0 {
				res.WriteString(strings.ToUpper(w[:1]) + strings.ToLower(w[1:]))
			}
		}
	}

	return res.String()
}

func parseRequest(item PostmanItem, namespace string) Endpoint {
	req := item.Request
	methodName := normalizeMethodName(item.Name)
	key := methodName
	if namespace != "" {
		key = namespace + "." + methodName
	}

	return Endpoint{
		Key:         key,
		Name:        item.Name,
		Namespace:   namespace,
		Method:      parseMethod(req),
		URL:         parseURL(req),
		Headers:     parseHeaders(req),
		Body:        parseBody(req.Body),
		Auth:        parseAuth(req.Auth),
		Description: parseDescription(item.Description),
		Responses:   parseResponses(item.Response),
		Variables:   parseURLVariables(req),
		Raw:         item,
	}
}

func parseMethod(req *PostmanRequest) string {
	if req.Method == "" {
		return "GET"
	}
	return strings.ToUpper(req.Method)
}

func parseURL(req *PostmanRequest) string {
	if req.URL == nil {
		return ""
	}

	if s, ok := req.URL.(string); ok {
		return s
	}

	// Try mapping to map first (raw JSON mapping)
	var urlObj PostmanURL
	data, err := json.Marshal(req.URL)
	if err == nil {
		if err := json.Unmarshal(data, &urlObj); err == nil && urlObj.Raw != "" {
			return urlObj.Raw
		}
	}

	return ""
}

func parseHeaders(req *PostmanRequest) map[string]string {
	headers := make(map[string]string)
	for _, h := range req.Header {
		if h.Disabled || h.Key == "" {
			continue
		}
		headers[h.Key] = h.Value
	}
	return headers
}

func parseBody(body *PostmanBody) interface{} {
	if body == nil {
		return nil
	}
	switch body.Mode {
	case "raw":
		var parsed interface{}
		err := json.Unmarshal([]byte(body.Raw), &parsed)
		if err == nil {
			return parsed
		}
		return body.Raw
	case "urlencoded":
		res := make(map[string]interface{})
		for _, item := range body.Urlencoded {
			if item.Disabled {
				continue
			}
			res[item.Key] = item.Value
		}
		return res
	case "formdata":
		res := make(map[string]interface{})
		for _, item := range body.Formdata {
			if item.Disabled {
				continue
			}
			if item.Type == "file" {
				res[item.Key] = "[FILE: " + item.Src + "]"
			} else {
				res[item.Key] = item.Value
			}
		}
		return res
	case "graphql":
		return body.GraphQL
	default:
		return nil
	}
}

func parseAuth(auth interface{}) interface{} {
	if auth == nil {
		return nil
	}
	// Similar to Node.js which returns { type, configured: true }
	if m, ok := auth.(map[string]interface{}); ok {
		return map[string]interface{}{
			"type":       m["type"],
			"configured": true,
		}
	}
	return map[string]interface{}{
		"configured": true,
	}
}

func parseDescription(desc interface{}) string {
	if desc == nil {
		return ""
	}
	if s, ok := desc.(string); ok {
		return s
	}
	if m, ok := desc.(map[string]interface{}); ok {
		if content, ok := m["content"].(string); ok {
			return content
		}
	}
	return ""
}

func parseResponses(responses []PostmanResponse) []ResponseExample {
	var examples []ResponseExample
	for _, resp := range responses {
		headers := make(map[string]string)
		for _, h := range resp.Header {
			headers[h.Key] = h.Value
		}

		var parsedBody interface{} = resp.Body
		var bodyObj interface{}
		if err := json.Unmarshal([]byte(resp.Body), &bodyObj); err == nil {
			parsedBody = bodyObj
		}

		examples = append(examples, ResponseExample{
			Name:    resp.Name,
			Status:  resp.Code,
			Headers: headers,
			Body:    parsedBody,
		})
	}
	return examples
}

func parseURLVariables(req *PostmanRequest) []VariableExample {
	if req.URL == nil {
		return nil
	}

	var urlObj PostmanURL
	data, err := json.Marshal(req.URL)
	if err != nil {
		return nil
	}

	if err := json.Unmarshal(data, &urlObj); err != nil {
		return nil
	}

	var variables []VariableExample
	for _, v := range urlObj.Variable {
		variables = append(variables, VariableExample{
			Key:         v.Key,
			Value:       v.Value,
			Description: v.Description,
		})
	}

	return variables
}

func postProcess(endpoints []Endpoint, collection *PostmanCollection) []Endpoint {
	variables := extractVariables(collection)

	for i := range endpoints {
		endpoints[i].URL = replaceVariables(endpoints[i].URL, variables)
		for k, v := range endpoints[i].Headers {
			endpoints[i].Headers[k] = replaceVariables(v, variables)
		}
		endpoints[i].CollectionName = collection.Info.Name
		endpoints[i].CollectionID = collection.Info.PostmanID
	}

	return endpoints
}

func extractVariables(collection *PostmanCollection) map[string]string {
	variables := make(map[string]string)
	for _, v := range collection.Variable {
		if v.Key != "" {
			if s, ok := v.Value.(string); ok {
				variables[v.Key] = s
			} else {
				variables[v.Key] = fmt.Sprintf("%v", v.Value)
			}
		}
	}
	return variables
}

func replaceVariables(str string, variables map[string]string) string {
	re := regexp.MustCompile(`\{\{(\w+)\}\}`)
	return re.ReplaceAllStringFunc(str, func(match string) string {
		key := match[2 : len(match)-2]
		if val, exists := variables[key]; exists {
			return val
		}
		return match
	})
}

type Stats struct {
	TotalEndpoints   int            `json:"totalEndpoints"`
	Namespaces       []string       `json:"namespaces"`
	NamespacesCount  int            `json:"namespacesCount"`
	MethodsCount     map[string]int `json:"methodsCount"`
	CollectionName   string         `json:"collectionName"`
	CollectionSchema string         `json:"collectionSchema"`
}

func (p *PostmanParser) GetStats(collection *PostmanCollection) Stats {
	endpoints := p.Parse(collection)
	namespacesSet := make(map[string]struct{})
	methodsCount := make(map[string]int)

	for _, ep := range endpoints {
		if ep.Namespace != "" {
			parts := strings.Split(ep.Namespace, ".")
			namespacesSet[parts[0]] = struct{}{}
		}

		method := ep.Method
		if method == "" {
			method = "GET"
		}
		methodsCount[method]++
	}

	var namespaces []string
	for k := range namespacesSet {
		namespaces = append(namespaces, k)
	}

	return Stats{
		TotalEndpoints:   len(endpoints),
		Namespaces:       namespaces,
		NamespacesCount:  len(namespaces),
		MethodsCount:     methodsCount,
		CollectionName:   collection.Info.Name,
		CollectionSchema: collection.Info.Schema,
	}
}
