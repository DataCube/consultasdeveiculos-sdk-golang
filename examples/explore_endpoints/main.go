package main

import (
	"encoding/json"
	"fmt"
	"strings"

	sdk "github.com/DataCube/consultasdeveiculos-sdk-golang"
)

func main() {
	// Inicializa a SDK em modo Sandbox
	client, err := sdk.NewClient(sdk.Options{
		Sandbox: true,
	})
	if err != nil {
		fmt.Printf("❌ Erro ao inicializar SDK: %v\n", err)
		return
	}

	fmt.Println("🔍 Explorando a SDK")
	fmt.Println("")

	// 1. Informações gerais
	fmt.Println("📊 Informações da SDK:")
	info := client.GetInfo()
	fmt.Printf("   Runtime: v%s\n", info.RuntimeVersion)
	fmt.Printf("   Specification: v%s\n", info.SpecVersion)
	fmt.Printf("   Total de endpoints: %d\n", info.EndpointsCount)
	fmt.Printf("   Total de slugs: %d\n", info.SlugsCount)
	fmt.Printf("   Namespaces: %s\n", strings.Join(info.Namespaces, ", "))
	fmt.Println("")

	// 2. Listar slugs por namespace (prefixo)
	fmt.Println("📋 Slugs por namespace:")
	slugs := client.ListSlugs()
	byPrefix := make(map[string][]string)

	for _, slug := range slugs {
		prefix := strings.Split(slug, "_")[0]
		byPrefix[prefix] = append(byPrefix[prefix], slug)
	}

	for prefix, prefixSlugs := range byPrefix {
		fmt.Printf("\n   📁 %s (%d slugs)\n", prefix, len(prefixSlugs))
		count := 0
		for _, s := range prefixSlugs {
			fmt.Printf("      client.Call(\"%s\", ...)\n", s)
			count++
			if count >= 3 {
				break
			}
		}
		if len(prefixSlugs) > 3 {
			fmt.Printf("      ... e mais %d slugs\n", len(prefixSlugs)-3)
		}
	}
	fmt.Println("")

	// 3. Buscar endpoints de veículos
	fmt.Println("🔎 Busca por \"agregados\":")
	agregadosEndpoints := client.SearchEndpoints("agregados")
	for _, ep := range agregadosEndpoints {
		fmt.Printf("   client.Call(\"%s\", ...) - %s\n", ep.Slug, ep.Name)
	}
	fmt.Println("")

	fmt.Println("🔎 Busca por \"cpf\":")
	cpfEndpoints := client.SearchEndpoints("cpf")
	count := 0
	for _, ep := range cpfEndpoints {
		fmt.Printf("   client.Call(\"%s\", ...)\n", ep.Slug)
		count++
		if count >= 5 {
			break
		}
	}
	if len(cpfEndpoints) > 5 {
		fmt.Printf("   ... e mais %d endpoints\n", len(cpfEndpoints)-5)
	}
	fmt.Println("")

	// 4. Obter detalhes de um endpoint
	fmt.Println("📖 Detalhes do slug \"veiculos_agregados\":")
	agregadosResults := client.SearchEndpoints("veiculos_agregados")
	if len(agregadosResults) > 0 {
		endpoint := agregadosResults[0]
		data, _ := json.MarshalIndent(endpoint, "", "  ")
		fmt.Println(string(data))
	}
	fmt.Println("")

	// 5. Listar todos os endpoints com detalhes
	fmt.Println("📋 Lista completa de endpoints (primeiros 10):")
	endpoints := client.ListEndpoints()
	for i, ep := range endpoints {
		if i >= 10 {
			break
		}
		fmt.Printf("   📌 %s\n", ep.Slug)
		fmt.Printf("      Nome: %s\n", ep.Name)
	}
	fmt.Printf("   ... e mais %d endpoints\n", len(endpoints)-10)
	fmt.Println("")

	fmt.Println("✅ Exploração concluída")
}
