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

	fmt.Println("🧪 Modo Sandbox ativado")
	fmt.Println("")

	info := client.GetInfo()
	fmt.Println("📦 SDK Info:")
	fmt.Printf("   Runtime: v%s\n", info.RuntimeVersion)
	fmt.Printf("   Specification: v%s\n", info.SpecVersion)
	fmt.Printf("   Endpoints: %d\n", info.EndpointsCount)
	fmt.Printf("   Sandbox: %t\n", info.Sandbox)
	fmt.Println("")

	// 1. Consulta de Veículo - Agregados
	fmt.Println("1️⃣ Consultando veículo (Agregados)...")
	agregados, err := client.Call("veiculos_agregados", map[string]interface{}{"placa": "ABC1234"})
	if err != nil {
		fmt.Printf("   ❌ Erro: %v\n\n", err)
	} else {
		dataBytes, _ := json.MarshalIndent(agregados.Data, "", "  ")
		fmt.Printf("   Status: %d\n", agregados.Status)
		fmt.Printf("   Sandbox: %t\n", agregados.Sandbox)
		fmt.Printf("   Data: %s\n\n", string(dataBytes))
	}

	// 2. Consulta de Veículo - Base Estadual
	fmt.Println("2️⃣ Consultando veículo (Base Estadual)...")
	baseEstadual, err := client.Call("veiculos_bin_estadual", map[string]interface{}{"placa": "XYZ9876"})
	if err != nil {
		fmt.Printf("   ❌ Erro: %v\n\n", err)
	} else {
		dataBytes, _ := json.MarshalIndent(baseEstadual.Data, "", "  ")
		fmt.Printf("   Status: %d\n", baseEstadual.Status)
		fmt.Printf("   Sandbox: %t\n", baseEstadual.Sandbox)
		fmt.Printf("   Data: %s\n\n", string(dataBytes))
	}

	// 3. Consulta de Dados Cadastrais - Nome
	fmt.Println("3️⃣ Consultando nome pelo CPF...")
	nomeCpf, err := client.Call("pessoas_nome", map[string]interface{}{"cpf": "123.456.789-00"})
	if err != nil {
		fmt.Printf("   ❌ Erro: %v\n\n", err)
	} else {
		dataBytes, _ := json.MarshalIndent(nomeCpf.Data, "", "  ")
		fmt.Printf("   Status: %d\n", nomeCpf.Status)
		fmt.Printf("   Sandbox: %t\n", nomeCpf.Sandbox)
		fmt.Printf("   Data: %s\n\n", string(dataBytes))
	}

	// 4. Consulta de CNH Nacional
	fmt.Println("4️⃣ Consultando CNH Nacional...")
	cnh, err := client.Call("cnh_nacional_simples", map[string]interface{}{
		"cnh":             "12345678901",
		"data_nascimento": "01/01/1990",
	})
	if err != nil {
		fmt.Printf("   ❌ Erro: %v\n\n", err)
	} else {
		dataBytes, _ := json.MarshalIndent(cnh.Data, "", "  ")
		fmt.Printf("   Status: %d\n", cnh.Status)
		fmt.Printf("   Data: %s\n\n", string(dataBytes))
	}

	// 5. Consulta de Crédito - Score PF
	fmt.Println("5️⃣ Consultando crédito PF...")
	credito, err := client.Call("credito_credito_completa_pf", map[string]interface{}{"cpf": "123.456.789-00"})
	if err != nil {
		fmt.Printf("   ❌ Erro: %v\n\n", err)
	} else {
		dataBytes, _ := json.MarshalIndent(credito.Data, "", "  ")
		fmt.Printf("   Status: %d\n", credito.Status)
		fmt.Printf("   Data: %s\n\n", string(dataBytes))
	}

	// 6. Consulta de Conta
	fmt.Println("6️⃣ Consultando consumo da conta...")
	consumo, err := client.Call("conta_consultas", map[string]interface{}{"de": "2026-01-01", "ate": "2026-01-31"})
	if err != nil {
		fmt.Printf("   ❌ Erro: %v\n\n", err)
	} else {
		dataBytes, _ := json.MarshalIndent(consumo.Data, "", "  ")
		fmt.Printf("   Status: %d\n", consumo.Status)
		fmt.Printf("   Data: %s\n\n", string(dataBytes))
	}

	// 7. Listando slugs disponíveis
	fmt.Println("7️⃣ Slugs de veículos disponíveis:")
	slugs := client.ListSlugs()
	var veiculosSlugs []string
	for _, s := range slugs {
		if strings.HasPrefix(s, "veiculos_") {
			veiculosSlugs = append(veiculosSlugs, s)
		}
	}

	count := 0
	for _, s := range veiculosSlugs {
		fmt.Printf("   - client.Call(\"%s\", ...)\n", s)
		count++
		if count >= 5 {
			break
		}
	}
	if len(veiculosSlugs) > 5 {
		fmt.Printf("   ... e mais %d slugs\n", len(veiculosSlugs)-5)
	}
	fmt.Println("")

	fmt.Println("✅ Todas as chamadas realizadas com sucesso (sandbox)")
	fmt.Println("   Nenhuma requisição real foi feita à API.")
}
