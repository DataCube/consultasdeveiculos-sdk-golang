package consultasdeveiculos

import (
	"embed"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/DataCube/consultasdeveiculos-sdk-golang/core"
	sdkErrors "github.com/DataCube/consultasdeveiculos-sdk-golang/errors"
	"github.com/DataCube/consultasdeveiculos-sdk-golang/parser"
	"github.com/DataCube/consultasdeveiculos-sdk-golang/transport"
)

//go:embed spec/postman.json
var DefaultSpecFS embed.FS

const VERSION = "1.0.0"

func init() {
	core.LoadEnv()
}

type Options struct {
	AuthToken           string
	Sandbox             bool
	BaseURL             string
	Timeout             time.Duration
	MaxRetries          int
	RetryDelay          time.Duration
	Headers             map[string]string
	DisableCompression  bool
	CacheDir            string
	DownloadURL         string
	SandboxDelay        time.Duration
	SandboxRandomErrors bool
	SandboxErrorRate    float64
}

type Client struct {
	Options       Options
	Sandbox       bool
	Initialized   bool
	configManager *core.ConfigManager
	registry      *core.EndpointRegistry
	transport     transport.Transport
	manifest      *core.Manifest
	postman       *parser.PostmanCollection
	slugMap       map[string]parser.Endpoint
}

func NewClient(opts Options) (*Client, error) {
	if !opts.Sandbox && opts.AuthToken == "" {
		return nil, sdkErrors.NewAuthenticationError("auth_token é obrigatório. Use Options{ Sandbox: true } para modo de teste.", nil)
	}

	userCfg := core.Config{
		CacheDir:           opts.CacheDir,
		DownloadURL:        opts.DownloadURL,
		Timeout:            opts.Timeout,
		MaxRetries:         opts.MaxRetries,
		RetryDelay:         opts.RetryDelay,
		DisableCompression: opts.DisableCompression,
		DefaultHeaders:     opts.Headers,
	}
	cm := core.NewConfigManager(userCfg)

	defaultPostman, _ := DefaultSpecFS.ReadFile("spec/postman.json")
	var defaultManifest []byte // manifest is auto-generated if missing

	loader := core.NewPostmanLoader(cm, defaultPostman, defaultManifest, "")
	specRes, err := loader.Load()
	if err != nil {
		return nil, err
	}

	if !isVersionCompatible(VERSION, specRes.Manifest.MinRuntimeVersion) {
		return nil, sdkErrors.NewSpecificationError(
			fmt.Sprintf("Atualize a SDK para continuar utilizando esta versão da API Specification. Runtime atual: %s, Mínimo requerido: %s", VERSION, specRes.Manifest.MinRuntimeVersion),
			nil,
		)
	}

	registry := core.NewEndpointRegistry()
	parserInstance := parser.NewPostmanParser()
	endpoints := parserInstance.Parse(specRes.Postman)

	slugMap := make(map[string]parser.Endpoint)
	for _, ep := range endpoints {
		registry.Register(ep)
		slug := urlToSlug(ep.URL)
		if slug != "" {
			slugMap[slug] = ep
		}
	}

	transportOpts := transport.TransportOptions{
		Token:               opts.AuthToken,
		BaseURL:             opts.BaseURL,
		Timeout:             cm.GetTimeout(),
		MaxRetries:          cm.GetMaxRetries(),
		RetryDelay:          cm.GetRetryDelay(),
		Headers:             cm.GetDefaultHeaders(),
		DisableCompression:  opts.DisableCompression,
		SandboxDelay:        opts.SandboxDelay,
		SandboxRandomErrors: opts.SandboxRandomErrors,
		SandboxErrorRate:    opts.SandboxErrorRate,
	}

	var transportInstance transport.Transport
	if opts.Sandbox {
		transportInstance = transport.NewSandboxTransport(transportOpts)
	} else {
		transportInstance = transport.NewHttpTransport(transportOpts)
	}

	return &Client{
		Options:       opts,
		Sandbox:       opts.Sandbox,
		Initialized:   true,
		configManager: cm,
		registry:      registry,
		transport:     transportInstance,
		manifest:      specRes.Manifest,
		postman:       specRes.Postman,
		slugMap:       slugMap,
	}, nil
}

func urlToSlug(endpointURL string) string {
	if endpointURL == "" {
		return ""
	}

	path := endpointURL
	if strings.Contains(endpointURL, "://") {
		u, err := url.Parse(endpointURL)
		if err == nil {
			path = u.Path
		}
	}

	path = strings.Trim(path, "/")

	reVars := regexp.MustCompile(`\{\{[^}]+\}\}`)
	path = reVars.ReplaceAllString(path, "")
	path = strings.Trim(path, "/")

	path = strings.ReplaceAll(path, "/", "_")
	path = strings.ReplaceAll(path, "-", "_")

	reUnderscores := regexp.MustCompile(`_+`)
	path = reUnderscores.ReplaceAllString(path, "_")

	path = strings.Trim(path, "_")

	return strings.ToLower(path)
}

func isVersionCompatible(current, minimum string) bool {
	parseVersion := func(v string) []int {
		parts := strings.Split(v, ".")
		res := make([]int, 3)
		for i := 0; i < len(parts) && i < 3; i++ {
			var val int
			fmt.Sscanf(parts[i], "%d", &val)
			res[i] = val
		}
		return res
	}

	curr := parseVersion(current)
	min := parseVersion(minimum)

	if curr[0] > min[0] {
		return true
	}
	if curr[0] < min[0] {
		return false
	}
	if curr[1] > min[1] {
		return true
	}
	if curr[1] < min[1] {
		return false
	}
	return curr[2] >= min[2]
}

func (c *Client) ExecuteBySlug(slug string, params map[string]interface{}) (*transport.Response, error) {
	ep, exists := c.slugMap[slug]
	if !exists {
		var available []string
		count := 0
		for k := range c.slugMap {
			available = append(available, k)
			count++
			if count >= 5 {
				break
			}
		}
		return nil, sdkErrors.NewEndpointNotFoundError(
			fmt.Sprintf("Endpoint \"%s\" não encontrado. Exemplos disponíveis: %s...", slug, strings.Join(available, ", ")),
			map[string]interface{}{"slug": slug, "availableExamples": available},
		)
	}

	return c.transport.Request(ep, params, transport.RequestOptions{})
}

func (c *Client) Call(slug string, params map[string]interface{}) (*transport.Response, error) {
	return c.ExecuteBySlug(slug, params)
}

type SDKInfo struct {
	RuntimeVersion string   `json:"runtimeVersion"`
	SpecVersion    string   `json:"specVersion"`
	GeneratedAt    string   `json:"generatedAt"`
	Sandbox        bool     `json:"sandbox"`
	EndpointsCount int      `json:"endpointsCount"`
	Namespaces     []string `json:"namespaces"`
	SlugsCount     int      `json:"slugsCount"`
}

func (c *Client) GetInfo() SDKInfo {
	return SDKInfo{
		RuntimeVersion: VERSION,
		SpecVersion:    c.manifest.SpecVersion,
		GeneratedAt:    c.manifest.GeneratedAt,
		Sandbox:        c.Sandbox,
		EndpointsCount: c.registry.Size(),
		Namespaces:     c.registry.GetNamespaces(),
		SlugsCount:     len(c.slugMap),
	}
}

type EndpointDetail struct {
	Slug        string   `json:"slug"`
	Key         string   `json:"key"`
	Name        string   `json:"name"`
	Method      string   `json:"method"`
	URL         string   `json:"url"`
	Description string   `json:"description"`
	Params      []string `json:"params"`
}

func (c *Client) ListEndpoints() []EndpointDetail {
	var list []EndpointDetail
	for slug, ep := range c.slugMap {
		var params []string
		if ep.Body != nil {
			if m, ok := ep.Body.(map[string]interface{}); ok {
				for k := range m {
					params = append(params, k)
				}
			}
		}
		list = append(list, EndpointDetail{
			Slug:        slug,
			Key:         ep.Key,
			Name:        ep.Name,
			Method:      ep.Method,
			URL:         ep.URL,
			Description: ep.Description,
			Params:      params,
		})
	}
	return list
}

func (c *Client) ListSlugs() []string {
	var list []string
	for k := range c.slugMap {
		list = append(list, k)
	}
	return list
}

func (c *Client) SearchEndpoints(pattern string) []EndpointDetail {
	re, err := regexp.Compile("(?i)" + regexp.QuoteMeta(pattern))
	if err != nil {
		return nil
	}

	var results []EndpointDetail
	for slug, ep := range c.slugMap {
		if re.MatchString(slug) || re.MatchString(ep.Name) {
			var params []string
			if ep.Body != nil {
				if m, ok := ep.Body.(map[string]interface{}); ok {
					for k := range m {
						params = append(params, k)
					}
				}
			}
			results = append(results, EndpointDetail{
				Slug:        slug,
				Key:         ep.Key,
				Name:        ep.Name,
				Method:      ep.Method,
				URL:         ep.URL,
				Description: ep.Description,
				Params:      params,
			})
		}
	}
	return results
}

func (c *Client) Help(filter string) {
	info := c.GetInfo()
	fmt.Println("")
	fmt.Println("📦 ConsultadeveiculosSDK - Help")
	fmt.Println("════════════════════════════════════════════════════════════════")
	fmt.Println("")
	fmt.Printf("   Runtime: v%s\n", info.RuntimeVersion)
	fmt.Printf("   Spec: v%s\n", info.SpecVersion)
	modeStr := "🔴 PRODUÇÃO"
	if info.Sandbox {
		modeStr = "🧪 SANDBOX"
	}
	fmt.Printf("   Modo: %s\n", modeStr)
	fmt.Printf("   Endpoints: %d\n", info.EndpointsCount)
	fmt.Printf("   Namespaces: %s\n", strings.Join(info.Namespaces, ", "))
	fmt.Println("")
	fmt.Println("────────────────────────────────────────────────────────────────")
	fmt.Println("📖 USO BÁSICO")
	fmt.Println("────────────────────────────────────────────────────────────────")
	fmt.Println("")
	fmt.Println("   // Criar cliente")
	fmt.Println("   client, _ := consultasdeveiculos.NewClient(consultasdeveiculos.Options{ AuthToken: \"SEU_TOKEN\" })")
	fmt.Println("")
	fmt.Println("   // Chamar endpoint")
	fmt.Println("   result, _ := client.Call(\"SLUG\", map[string]interface{}{\"param\": \"valor\"})")
	fmt.Println("")
	fmt.Println("────────────────────────────────────────────────────────────────")
	fmt.Println("🔧 MÉTODOS AUXILIARES")
	fmt.Println("────────────────────────────────────────────────────────────────")
	fmt.Println("")
	fmt.Println("   client.Help(\"\")            Exibe esta ajuda")
	fmt.Println("   client.Help(\"veiculos\")    Filtra endpoints por termo")
	fmt.Println("   client.Endpoints()         Lista todos os endpoints")
	fmt.Println("   client.Info()              Informações do SDK")
	fmt.Println("   client.Search(\"placa\")     Busca endpoints")
	fmt.Println("")

	if filter != "" {
		fmt.Println("────────────────────────────────────────────────────────────────")
		fmt.Printf("🔍 ENDPOINTS COM \"%s\"\n", strings.ToUpper(filter))
		fmt.Println("────────────────────────────────────────────────────────────────")
		fmt.Println("")

		filtered := c.SearchEndpoints(filter)
		if len(filtered) == 0 {
			fmt.Printf("   Nenhum endpoint encontrado para \"%s\"\n", filter)
		} else {
			for _, ep := range filtered {
				paramsStr := ""
				if len(ep.Params) > 0 {
					paramsStr = fmt.Sprintf("{ %s }", strings.Join(ep.Params, ", "))
				}
				fmt.Printf("   📌 client.Call(\"%s\", %s)\n", ep.Slug, paramsStr)
				fmt.Printf("      %s\n", ep.Name)
				fmt.Println("")
			}
		}
	} else {
		fmt.Println("────────────────────────────────────────────────────────────────")
		fmt.Println("📌 ENDPOINTS (por namespace)")
		fmt.Println("────────────────────────────────────────────────────────────────")
		fmt.Println("")

		allEndpoints := c.ListEndpoints()
		grouped := make(map[string][]EndpointDetail)
		for _, ep := range allEndpoints {
			ns := strings.Split(ep.Slug, "_")[0]
			if ns == "" {
				ns = "outros"
			}
			grouped[ns] = append(grouped[ns], ep)
		}

		for _, ns := range info.Namespaces {
			eps := grouped[ns]
			if len(eps) == 0 {
				continue
			}
			fmt.Printf("   %s:\n", strings.ToUpper(ns))
			for _, ep := range eps {
				paramsStr := ""
				if len(ep.Params) > 0 {
					paramsStr = fmt.Sprintf("{ %s }", strings.Join(ep.Params, ", "))
				}
				fmt.Printf("     • client.Call(\"%s\", %s)\n", ep.Slug, paramsStr)
			}
			fmt.Println("")
		}
	}
	fmt.Println("════════════════════════════════════════════════════════════════")
	fmt.Println("")
}

func (c *Client) Endpoints() []EndpointDetail {
	list := c.ListEndpoints()
	fmt.Println("")
	fmt.Printf("📡 %d Endpoints Disponíveis\n", len(list))
	fmt.Println("")

	grouped := make(map[string][]EndpointDetail)
	for _, ep := range list {
		ns := strings.Split(ep.Slug, "_")[0]
		if ns == "" {
			ns = "outros"
		}
		grouped[ns] = append(grouped[ns], ep)
	}

	for ns, eps := range grouped {
		fmt.Printf("   %s (%d):\n", strings.ToUpper(ns), len(eps))
		for _, ep := range eps {
			paramsStr := ""
			if len(ep.Params) > 0 {
				paramsStr = fmt.Sprintf("{ %s }", strings.Join(ep.Params, ", "))
			}
			fmt.Printf("     • %s(%s)\n", ep.Slug, paramsStr)
		}
		fmt.Println("")
	}

	return list
}

func (c *Client) Info() SDKInfo {
	data := c.GetInfo()
	fmt.Println("")
	fmt.Println("ℹ️  SDK Info")
	fmt.Println("")
	fmt.Printf("   Runtime: v%s\n", data.RuntimeVersion)
	fmt.Printf("   Spec: v%s\n", data.SpecVersion)
	modeStr := "Produção"
	if data.Sandbox {
		modeStr = "Sandbox"
	}
	fmt.Printf("   Modo: %s\n", modeStr)
	fmt.Printf("   Endpoints: %d\n", data.EndpointsCount)
	fmt.Printf("   Namespaces: %s\n", strings.Join(data.Namespaces, ", "))
	fmt.Println("")
	return data
}

func (c *Client) Search(pattern string) []EndpointDetail {
	results := c.SearchEndpoints(pattern)
	fmt.Println("")
	fmt.Printf("🔍 Busca: \"%s\" (%d resultados)\n", pattern, len(results))
	fmt.Println("")

	for _, ep := range results {
		paramsStr := ""
		if len(ep.Params) > 0 {
			paramsStr = fmt.Sprintf("{ %s }", strings.Join(ep.Params, ", "))
		}
		fmt.Printf("   📌 client.Call(\"%s\", %s)\n", ep.Slug, paramsStr)
		fmt.Printf("      %s\n", ep.Name)
		fmt.Println("")
	}
	return results
}
