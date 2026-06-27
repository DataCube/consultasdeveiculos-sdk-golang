package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	sdk "github.com/DataCube/consultasdeveiculos-sdk-golang"
	"github.com/DataCube/consultasdeveiculos-sdk-golang/core"
	"github.com/DataCube/consultasdeveiculos-sdk-golang/parser"
)

func main() {
	args := os.Args[1:]

	if len(args) == 0 || contains(args, "--help") || contains(args, "-h") {
		showHelp()
		os.Exit(0)
	}

	command := args[0]
	cmdArgs := args[1:]

	switch command {
	case "endpoints":
		endpointsCmd(cmdArgs)
	case "update":
		updateCmd(cmdArgs)
	case "version":
		versionCmd(cmdArgs)
	case "doctor":
		doctorCmd(cmdArgs)
	case "clear-cache":
		clearCacheCmd(cmdArgs)
	default:
		fmt.Fprintf(os.Stderr, "❌ Comando desconhecido: %s\n\n", command)
		showHelp()
		os.Exit(1)
	}
}

func showHelp() {
	fmt.Println(`
📦 consultas-de-veiculos-sdk CLI

Comandos disponíveis:

  endpoints     Lista todos os endpoints disponíveis (gerados do Postman)
  update        Atualiza a especificação da API
  version       Exibe versões do Runtime e Specification
  doctor        Executa diagnóstico do ambiente
  clear-cache   Limpa o cache local

Uso:
  consultas-de-veiculos-sdk <comando>

Exemplos:
  consultas-de-veiculos-sdk endpoints                 # Lista todos os endpoints
  consultas-de-veiculos-sdk endpoints veiculos        # Endpoints de veículos
  consultas-de-veiculos-sdk endpoints --verbose       # Com descrições
  consultas-de-veiculos-sdk update
  consultas-de-veiculos-sdk version
  consultas-de-veiculos-sdk doctor

Para mais informações sobre um comando específico:
  consultas-de-veiculos-sdk <comando> --help`)
}

func endpointsCmd(args []string) {
	if contains(args, "--help") || contains(args, "-h") {
		fmt.Println(`
📡 consultas-de-veiculos-sdk endpoints

Lista todos os endpoints disponíveis gerados a partir da coleção Postman.

Uso:
  consultas-de-veiculos-sdk endpoints [namespace] [opções]

Argumentos:
  namespace     Filtra endpoints por namespace (ex: veiculos, cadastros, cnh)

Opções:
  -v, --verbose   Exibe descrições e URLs dos endpoints
  --json          Saída em formato JSON
  -h, --help      Exibe esta ajuda

Exemplos:
  consultas-de-veiculos-sdk endpoints                    # Lista todos
  consultas-de-veiculos-sdk endpoints veiculos           # Apenas veículos
  consultas-de-veiculos-sdk endpoints cnh --verbose      # CNH com detalhes
  consultas-de-veiculos-sdk endpoints --json             # Saída JSON`)
		return
	}

	jsonOutput := contains(args, "--json")
	verbose := contains(args, "--verbose") || contains(args, "-v")

	var namespace string
	for _, arg := range args {
		if !strings.HasPrefix(arg, "-") {
			namespace = arg
			break
		}
	}

	client, err := sdk.NewClient(sdk.Options{Sandbox: true})
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Erro ao inicializar SDK: %s\n", err.Error())
		os.Exit(1)
	}

	allEndpoints := client.ListEndpoints()
	info := client.GetInfo()

	if jsonOutput {
		data, _ := json.MarshalIndent(allEndpoints, "", "  ")
		fmt.Println(string(data))
		return
	}

	fmt.Println("")
	fmt.Println("📡 Endpoints Disponíveis")
	fmt.Println("")
	fmt.Printf("   Specification: v%s\n", info.SpecVersion)
	fmt.Printf("   Total: %d endpoints\n", len(allEndpoints))
	fmt.Printf("   Namespaces: %s\n", strings.Join(info.Namespaces, ", "))
	fmt.Println("")
	fmt.Println("   💡 Use o SLUG para chamar: client.Call(\"<slug>\", { params })")
	fmt.Println("")

	// Group by namespace
	byNamespace := make(map[string][]sdk.EndpointDetail)
	for _, ep := range allEndpoints {
		parts := strings.Split(ep.Slug, "_")
		ns := parts[0]
		if ns == "" {
			ns = "outros"
		}
		byNamespace[ns] = append(byNamespace[ns], ep)
	}

	// Filter by namespace if specified
	var namespacesToShow []string
	if namespace != "" {
		if _, exists := byNamespace[namespace]; !exists {
			fmt.Printf("   ❌ Namespace \"%s\" não encontrado.\n", namespace)
			var keys []string
			for k := range byNamespace {
				keys = append(keys, k)
			}
			fmt.Printf("   Namespaces disponíveis: %s\n\n", strings.Join(keys, ", "))
			return
		}
		namespacesToShow = []string{namespace}
	} else {
		for _, ns := range info.Namespaces {
			if _, exists := byNamespace[ns]; exists {
				namespacesToShow = append(namespacesToShow, ns)
			}
		}
		// add remaining
		for k := range byNamespace {
			found := false
			for _, ns := range info.Namespaces {
				if ns == k {
					found = true
					break
				}
			}
			if !found {
				namespacesToShow = append(namespacesToShow, k)
			}
		}
	}

	for _, ns := range namespacesToShow {
		eps := byNamespace[ns]
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Printf("📁 %s (%d endpoints)\n", strings.ToUpper(ns), len(eps))
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Println("")

		for _, ep := range eps {
			paramsStr := ""
			if len(ep.Params) > 0 {
				paramsStr = fmt.Sprintf("{ %s }", strings.Join(ep.Params, ", "))
			}
			fmt.Printf("   📌 client.Call(\"%s\", %s)\n", ep.Slug, paramsStr)
			fmt.Printf("      Nome: %s\n", ep.Name)

			if verbose && ep.Description != "" {
				desc := ep.Description
				if len(desc) > 70 {
					desc = desc[:70] + "..."
				}
				fmt.Printf("      Desc: %s\n", desc)
			}

			if verbose && ep.URL != "" {
				fmt.Printf("      URL:  %s\n", ep.URL)
			}

			fmt.Println("")
		}
	}

	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("")
	fmt.Println("📝 Exemplo de uso:")
	fmt.Println("")
	fmt.Println("   client, _ := consultasdeveiculos.NewClient(consultasdeveiculos.Options{ AuthToken: \"TOKEN\" });")
	if len(allEndpoints) > 0 {
		example := allEndpoints[0]
		paramsStr := "{}"
		if len(example.Params) > 0 {
			paramsStr = fmt.Sprintf("{ \"%s\": \"valor\" }", example.Params[0])
		}
		fmt.Printf("   result, _ := client.Call(\"%s\", %s);\n", example.Slug, paramsStr)
	}
	fmt.Println("")
	fmt.Println("💡 Dicas:")
	fmt.Println("   • Use --verbose para ver descrições e URLs")
	fmt.Println("   • Use --json para saída em JSON")
	fmt.Println("   • Filtre por namespace: consultas-de-veiculos-sdk endpoints veiculos")
	fmt.Println("")
}

func updateCmd(args []string) {
	if contains(args, "--help") || contains(args, "-h") {
		fmt.Println(`
📦 consultas-de-veiculos-sdk update

Atualiza a especificação da API baixando a versão mais recente do servidor.
Sempre baixa e sobrescreve a versão local.

Uso:
  consultas-de-veiculos-sdk update

Opções:
  -h, --help    Exibe esta ajuda

Exemplos:
  consultas-de-veiculos-sdk update`)
		return
	}

	cm := core.NewConfigManager(core.Config{})
	postmanPath := cm.GetCachedPostmanPath()

	fmt.Println("🔄 Atualizando especificação da API...")
	fmt.Println("")

	var currentVersion string
	if _, err := os.Stat(postmanPath); err == nil {
		postmanBytes, errReadFile := os.ReadFile(postmanPath)
		if errReadFile == nil {
			var postman parser.PostmanCollection
			if errUnmarshal := json.Unmarshal(postmanBytes, &postman); errUnmarshal == nil {
				currentVersion = extractVersion(&postman)
				fmt.Printf("   Versão atual: %s\n", currentVersion)
			} else {
				fmt.Println("   Versão atual: corrompida")
			}
		}
	} else {
		// check in local project directory if exists
		localPostman := filepath.Join("spec", "postman.json")
		if _, errLocal := os.Stat(localPostman); errLocal == nil {
			postmanBytes, errReadFile := os.ReadFile(localPostman)
			if errReadFile == nil {
				var postman parser.PostmanCollection
				if errUnmarshal := json.Unmarshal(postmanBytes, &postman); errUnmarshal == nil {
					currentVersion = extractVersion(&postman)
					fmt.Printf("   Versão atual: %s (local)\n", currentVersion)
				}
			}
		} else {
			fmt.Println("   Versão atual: nenhuma")
		}
	}

	downloadURL := cm.GetDownloadURL()
	fmt.Println("   Baixando do servidor...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", downloadURL, nil)
	if err != nil {
		fmt.Printf("❌ Erro ao criar requisição: %s\n", err.Error())
		os.Exit(1)
	}
	req.Header.Set("Accept", "application/json, */*")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println("")
		if strings.Contains(err.Error(), "no such host") || strings.Contains(err.Error(), "connection refused") {
			fmt.Println("❌ Não foi possível conectar ao servidor.")
			fmt.Println("   Verifique sua conexão com a internet.")
		} else {
			fmt.Printf("❌ Erro: %s\n", err.Error())
		}
		fmt.Println("")
		fmt.Printf("   URL: %s\n", downloadURL)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		fmt.Printf("\n❌ Erro: HTTP %s\n\n", resp.Status)
		os.Exit(1)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("\n❌ Erro ao ler resposta: %s\n\n", err.Error())
		os.Exit(1)
	}

	var postman parser.PostmanCollection
	if err := json.Unmarshal(bodyBytes, &postman); err != nil {
		fmt.Printf("\n❌ Resposta não é JSON válido\n\n")
		os.Exit(1)
	}

	fmt.Println("   Validando estrutura...")
	if postman.Info.Name == "" || len(postman.Item) == 0 {
		fmt.Printf("\n❌ Estrutura Postman inválida (faltando info ou item)\n\n")
		os.Exit(1)
	}

	newVersion := extractVersion(&postman)
	fmt.Printf("   Nova versão: %s\n", newVersion)

	fmt.Println("   Salvando...")
	// Save to local cache dir using loader
	loader := core.NewPostmanLoader(cm, nil, nil, "")
	manifest := &core.Manifest{
		SpecVersion:       newVersion,
		MinRuntimeVersion: "1.0.0",
		GeneratedAt:       time.Now().UTC().Format(time.RFC3339),
	}
	errSave := loader.SaveToCache(&postman, manifest)
	if errSave != nil {
		fmt.Printf("\n❌ Erro ao salvar cache: %s\n\n", errSave.Error())
		os.Exit(1)
	}

	// Save to spec/postman.json if local directory exists (for development updates)
	if _, errStat := os.Stat("spec"); errStat == nil {
		postmanIndent, _ := json.MarshalIndent(&postman, "", "  ")
		_ = os.WriteFile(filepath.Join("spec", "postman.json"), postmanIndent, 0644)
		manifestIndent, _ := json.MarshalIndent(manifest, "", "  ")
		_ = os.WriteFile(filepath.Join("spec", "manifest.json"), manifestIndent, 0644)
	}

	fmt.Println("")
	fmt.Println("✅ Especificação atualizada com sucesso!")
	fmt.Println("")
	prevVer := currentVersion
	if prevVer == "" {
		prevVer = "nenhuma"
	}
	fmt.Printf("   Versão anterior: %s\n", prevVer)
	fmt.Printf("   Versão atual: %s\n", newVersion)

	// Count endpoints
	parserInstance := parser.NewPostmanParser()
	endpoints := parserInstance.Parse(&postman)
	fmt.Printf("   Endpoints: %d\n\n", len(endpoints))
}

func extractVersion(postman *parser.PostmanCollection) string {
	name := postman.Info.Name
	re := regexp.MustCompile(`(?i)V([\d.]+)`)
	matches := re.FindStringSubmatch(name)
	if len(matches) > 1 {
		return matches[1]
	}
	if postman.Info.Version != "" {
		return postman.Info.Version
	}
	return "unknown"
}

func versionCmd(args []string) {
	if contains(args, "--help") || contains(args, "-h") {
		fmt.Println(`
📦 consultas-de-veiculos-sdk version

Exibe as versões do Runtime e da Specification.

Uso:
  consultas-de-veiculos-sdk version [opções]

Opções:
  -v, --verbose   Exibe informações detalhadas
  --json          Saída em formato JSON
  -h, --help      Exibe esta ajuda

Exemplos:
  consultas-de-veiculos-sdk version
  consultas-de-veiculos-sdk version --verbose
  consultas-de-veiculos-sdk version --json`)
		return
	}

	verbose := contains(args, "--verbose") || contains(args, "-v")
	jsonOutput := contains(args, "--json")

	cm := core.NewConfigManager(core.Config{})

	specVersion := "não instalada"
	var specGeneratedAt string
	var specSource string
	var minRuntimeVersion string

	cachedManifestPath := cm.GetCachedManifestPath()
	localManifestPath := filepath.Join("spec", "manifest.json")

	if _, err := os.Stat(cachedManifestPath); err == nil {
		bytes, errReadFile := os.ReadFile(cachedManifestPath)
		if errReadFile == nil {
			var manifest core.Manifest
			if errUnmarshal := json.Unmarshal(bytes, &manifest); errUnmarshal == nil {
				specVersion = manifest.SpecVersion
				specGeneratedAt = manifest.GeneratedAt
				minRuntimeVersion = manifest.MinRuntimeVersion
				specSource = "cache"
			}
		}
	} else if _, err := os.Stat(localManifestPath); err == nil {
		bytes, errReadFile := os.ReadFile(localManifestPath)
		if errReadFile == nil {
			var manifest core.Manifest
			if errUnmarshal := json.Unmarshal(bytes, &manifest); errUnmarshal == nil {
				specVersion = manifest.SpecVersion
				specGeneratedAt = manifest.GeneratedAt
				minRuntimeVersion = manifest.MinRuntimeVersion
				specSource = "package"
			}
		}
	} else {
		// try loading from loader default spec
		client, err := sdk.NewClient(sdk.Options{Sandbox: true})
		if err == nil {
			info := client.GetInfo()
			specVersion = info.SpecVersion
			specGeneratedAt = info.GeneratedAt
			minRuntimeVersion = "1.0.0"
			specSource = "embedded"
		}
	}

	if jsonOutput {
		output := map[string]interface{}{
			"runtime": map[string]string{
				"version": sdk.VERSION,
			},
			"specification": map[string]interface{}{
				"version":           specVersion,
				"generatedAt":       specGeneratedAt,
				"source":            specSource,
				"minRuntimeVersion": minRuntimeVersion,
			},
		}
		data, _ := json.MarshalIndent(output, "", "  ")
		fmt.Println(string(data))
		return
	}

	fmt.Println("")
	fmt.Println("📦 consultas-de-veiculos-sdk")
	fmt.Println("")
	fmt.Printf("   Runtime:       %s\n", sdk.VERSION)
	fmt.Printf("   Specification: %s\n", specVersion)

	if verbose {
		fmt.Println("")
		fmt.Println("   Detalhes:")
		if specGeneratedAt != "" {
			fmt.Printf("     Gerado em:     %s\n", specGeneratedAt)
		}
		if specSource != "" {
			fmt.Printf("     Fonte:         %s\n", specSource)
		}
		if minRuntimeVersion != "" {
			fmt.Printf("     Min Runtime:   %s\n", minRuntimeVersion)
		}
		fmt.Printf("     Cache:         %s\n", cm.GetCacheDir())
	}
	fmt.Println("")
}

type CheckResult struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

func doctorCmd(args []string) {
	if contains(args, "--help") || contains(args, "-h") {
		fmt.Println(`
🩺 consultas-de-veiculos-sdk doctor

Executa diagnóstico completo do ambiente da SDK.

Uso:
  consultas-de-veiculos-sdk doctor [opções]

Opções:
  -v, --verbose   Exibe informações detalhadas
  -n, --network   Testa conectividade com o servidor
  -h, --help      Exibe esta ajuda

Verificações realizadas:
  • Versão do Go
  • Permissões do diretório de cache
  • Presença do arquivo postman.json
  • Presença do arquivo manifest.json
  • Compatibilidade de versões
  • Estrutura da coleção Postman
  • Contagem de endpoints

Exemplos:
  consultas-de-veiculos-sdk doctor
  consultas-de-veiculos-sdk doctor --verbose
  consultas-de-veiculos-sdk doctor --network`)
		return
	}

	verbose := contains(args, "--verbose") || contains(args, "-v")
	checkNetwork := contains(args, "--network") || contains(args, "-n")

	var checks []CheckResult

	fmt.Println("")
	fmt.Println("🩺 Diagnóstico consultas-de-veiculos-sdk")
	fmt.Println("")

	// 1. Go version check
	goVersion := runtime.Version()
	checks = append(checks, CheckResult{
		Name:    "Go Runtime",
		Status:  "ok",
		Message: fmt.Sprintf("%s (mínimo: go1.18)", goVersion),
	})

	// 2. Cache dir permission check
	cm := core.NewConfigManager(core.Config{})
	cacheDir := cm.GetCacheDir()
	canWrite := true
	testFile := filepath.Join(cacheDir, ".doctor_test")
	errTest := os.WriteFile(testFile, []byte("test"), 0644)
	if errTest != nil {
		canWrite = false
	} else {
		_ = os.Remove(testFile)
	}

	if canWrite {
		checks = append(checks, CheckResult{
			Name:    "Diretório de cache",
			Status:  "ok",
			Message: cacheDir,
		})
	} else {
		checks = append(checks, CheckResult{
			Name:    "Diretório de cache",
			Status:  "warning",
			Message: fmt.Sprintf("Sem permissão de escrita em %s", cacheDir),
		})
	}

	// 3. Postman spec check
	cachedPostmanPath := cm.GetCachedPostmanPath()
	localPostmanPath := filepath.Join("spec", "postman.json")
	var postmanBytes []byte
	var postmanSource string
	var postmanFilename string

	if _, err := os.Stat(cachedPostmanPath); err == nil {
		postmanBytes, _ = os.ReadFile(cachedPostmanPath)
		postmanSource = "cache"
		postmanFilename = "postman.json"
	} else if _, err := os.Stat(localPostmanPath); err == nil {
		postmanBytes, _ = os.ReadFile(localPostmanPath)
		postmanSource = "package"
		postmanFilename = "postman.json"
	} else {
		// check embedded
		defaultSpec, _ := sdk.DefaultSpecFS.ReadFile("spec/postman.json")
		if len(defaultSpec) > 0 {
			postmanBytes = defaultSpec
			postmanSource = "embedded"
			postmanFilename = "postman.json"
		}
	}

	postmanExists := len(postmanBytes) > 0
	var postman parser.PostmanCollection
	if postmanExists {
		_ = json.Unmarshal(postmanBytes, &postman)
		checks = append(checks, CheckResult{
			Name:    "Postman Collection",
			Status:  "ok",
			Message: fmt.Sprintf("%s (fonte: %s)", postmanFilename, postmanSource),
		})
	} else {
		checks = append(checks, CheckResult{
			Name:    "Postman Collection",
			Status:  "error",
			Message: "Não encontrado (postman.json). Execute: consultas-de-veiculos-sdk update",
		})
	}

	// 4. Manifest check
	cachedManifestPath := cm.GetCachedManifestPath()
	localManifestPath := filepath.Join("spec", "manifest.json")
	var manifestBytes []byte
	var manifestSource string

	if _, err := os.Stat(cachedManifestPath); err == nil {
		manifestBytes, _ = os.ReadFile(cachedManifestPath)
		manifestSource = "cache"
	} else if _, err := os.Stat(localManifestPath); err == nil {
		manifestBytes, _ = os.ReadFile(localManifestPath)
		manifestSource = "package"
	}

	manifestExists := len(manifestBytes) > 0
	var manifest core.Manifest
	if manifestExists {
		_ = json.Unmarshal(manifestBytes, &manifest)
		checks = append(checks, CheckResult{
			Name:    "Manifest",
			Status:  "ok",
			Message: fmt.Sprintf("Encontrado (fonte: %s)", manifestSource),
		})
	} else {
		// If manifest doesn't exist on disk but we have embedded manifest, or default
		if postmanExists {
			checks = append(checks, CheckResult{
				Name:    "Manifest",
				Status:  "ok",
				Message: "Criado em runtime a partir do Postman",
			})
			manifest = core.Manifest{
				SpecVersion:       extractVersion(&postman),
				MinRuntimeVersion: "1.0.0",
			}
			manifestExists = true
		} else {
			checks = append(checks, CheckResult{
				Name:    "Manifest",
				Status:  "error",
				Message: "Não encontrado. Execute: consultas-de-veiculos-sdk update",
			})
		}
	}

	// 5. Version compatibility check
	if manifestExists {
		isCompatible := isVersionCompatible(sdk.VERSION, manifest.MinRuntimeVersion)
		if isCompatible {
			checks = append(checks, CheckResult{
				Name:    "Compatibilidade",
				Status:  "ok",
				Message: fmt.Sprintf("Runtime %s >= Mínimo %s", sdk.VERSION, manifest.MinRuntimeVersion),
			})
		} else {
			checks = append(checks, CheckResult{
				Name:    "Compatibilidade",
				Status:  "error",
				Message: fmt.Sprintf("Runtime %s < Mínimo %s. Atualize a SDK.", sdk.VERSION, manifest.MinRuntimeVersion),
			})
		}
	}

	// 6. Postman structure check
	if postmanExists {
		hasInfo := postman.Info.Name != ""
		hasItems := len(postman.Item) > 0

		if hasInfo && hasItems {
			checks = append(checks, CheckResult{
				Name:    "Estrutura Postman",
				Status:  "ok",
				Message: fmt.Sprintf("%s - %d item(s)", postman.Info.Name, len(postman.Item)),
			})
		} else {
			checks = append(checks, CheckResult{
				Name:    "Estrutura Postman",
				Status:  "warning",
				Message: "Estrutura incompleta ou vazia",
			})
		}

		// 7. Endpoints count check
		parserInstance := parser.NewPostmanParser()
		endpoints := parserInstance.Parse(&postman)
		if len(endpoints) > 0 {
			checks = append(checks, CheckResult{
				Name:    "Endpoints",
				Status:  "ok",
				Message: fmt.Sprintf("%d endpoint(s) disponível(is)", len(endpoints)),
			})

			if verbose {
				stats := parserInstance.GetStats(&postman)
				checks = append(checks, CheckResult{
					Name:    "Namespaces",
					Status:  "info",
					Message: strings.Join(stats.Namespaces, ", "),
				})
				var mCountStrs []string
				for m, c := range stats.MethodsCount {
					mCountStrs = append(mCountStrs, fmt.Sprintf("%s: %d", m, c))
				}
				checks = append(checks, CheckResult{
					Name:    "Métodos HTTP",
					Status:  "info",
					Message: strings.Join(mCountStrs, ", "),
				})
			}
		} else {
			checks = append(checks, CheckResult{
				Name:    "Endpoints",
				Status:  "warning",
				Message: "Nenhum endpoint disponível",
			})
		}
	}

	// 8. Network check (optional)
	if checkNetwork {
		downloadURL := cm.GetDownloadURL()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, "HEAD", downloadURL, nil)
		var resp *http.Response
		if err == nil {
			resp, err = http.DefaultClient.Do(req)
		}

		if err == nil && resp.StatusCode == 200 {
			checks = append(checks, CheckResult{
				Name:    "Conectividade",
				Status:  "ok",
				Message: fmt.Sprintf("Servidor acessível (%s)", downloadURL),
			})
		} else {
			errMsg := "Não foi possível conectar ao servidor"
			if err != nil {
				errMsg = fmt.Sprintf("Não foi possível conectar ao servidor: %s", err.Error())
			} else if resp != nil {
				errMsg = fmt.Sprintf("Servidor retornou status %d", resp.StatusCode)
			}
			checks = append(checks, CheckResult{
				Name:    "Conectividade",
				Status:  "warning",
				Message: errMsg,
			})
		}
		if resp != nil {
			resp.Body.Close()
		}
	}

	// Render output
	statusIcons := map[string]string{
		"ok":      "✅",
		"warning": "⚠️",
		"error":   "❌",
		"info":    "ℹ️",
	}

	for _, check := range checks {
		icon := statusIcons[check.Status]
		fmt.Printf("   %s %s: %s\n", icon, check.Name, check.Message)
	}

	fmt.Println("")

	errorsCount := 0
	warningsCount := 0
	for _, check := range checks {
		if check.Status == "error" {
			errorsCount++
		} else if check.Status == "warning" {
			warningsCount++
		}
	}

	if errorsCount > 0 {
		fmt.Printf("❌ %d erro(s) encontrado(s). Corrija os problemas acima.\n", errorsCount)
		os.Exit(1)
	} else if warningsCount > 0 {
		fmt.Printf("⚠️  %d aviso(s). A SDK pode funcionar com limitações.\n", warningsCount)
	} else {
		fmt.Println("✅ Tudo ok! A SDK está pronta para uso.")
	}
	fmt.Println("")
}

func clearCacheCmd(args []string) {
	if contains(args, "--help") || contains(args, "-h") {
		fmt.Println(`
🗑️  consultas-de-veiculos-sdk clear-cache

Limpa o cache local da SDK, removendo arquivos baixados.

Uso:
  consultas-de-veiculos-sdk clear-cache [opções]

Opções:
  -f, --force   Confirma a limpeza do cache
  -h, --help    Exibe esta ajuda

O que é removido:
  • postman.json - Cópia em cache da especificação
  • manifest.json - Cópia em cache do manifest
  • cache/ - Cache de respostas

Exemplos:
  consultas-de-veiculos-sdk clear-cache
  consultas-de-veiculos-sdk clear-cache --force`)
		return
	}

	force := contains(args, "--force") || contains(args, "-f")
	cm := core.NewConfigManager(core.Config{})
	cacheDir := cm.GetCacheDir()

	fmt.Println("")
	fmt.Println("🗑️  Limpando cache da SDK...")
	fmt.Println("")
	fmt.Printf("   Diretório: %s\n", cacheDir)

	if !force {
		fmt.Println("")
		fmt.Println("   ⚠️  Isso irá remover:")
		fmt.Println("      • postman.json (cópia em cache)")
		fmt.Println("      • manifest.json (cópia em cache)")
		fmt.Println("      • Cache de respostas")
		fmt.Println("")
		fmt.Println("   Use --force para confirmar a limpeza.")
		fmt.Println("")
		fmt.Println("   💡 Após limpar o cache, execute:")
		fmt.Println("      consultas-de-veiculos-sdk update")
		fmt.Println("")
		return
	}

	cm.ClearCache()
	fmt.Println("")
	fmt.Println("✅ Cache limpo com sucesso!")
	fmt.Println("")
	fmt.Println("   💡 Para baixar a especificação novamente, execute:")
	fmt.Println("      consultas-de-veiculos-sdk update")
	fmt.Println("")
}

func contains(slice []string, val string) bool {
	for _, item := range slice {
		if item == val {
			return true
		}
	}
	return false
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
