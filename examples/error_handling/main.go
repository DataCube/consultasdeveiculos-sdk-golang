package main

import (
	"encoding/json"
	"errors"
	"fmt"

	sdk "github.com/DataCube/consultasdeveiculos-sdk-golang"
	sdkErrors "github.com/DataCube/consultasdeveiculos-sdk-golang/errors"
)

func main() {
	fmt.Println("🔴 Exemplos de tratamento de erros")
	fmt.Println("")

	// ===================================
	// Exemplo 1: Inicialização sem token
	// ===================================
	fmt.Println("1️⃣ Tentando inicializar sem token...")
	_, err := sdk.NewClient(sdk.Options{})
	if err != nil {
		var authErr *sdkErrors.AuthenticationError
		if errors.As(err, &authErr) {
			fmt.Println("   ✅ AuthenticationError capturado:", authErr.Message)
		} else {
			fmt.Println("   ❌ Erro genérico capturado:", err)
		}
	}
	fmt.Println("")

	// Inicializa em sandbox para os próximos exemplos
	client, err := sdk.NewClient(sdk.Options{Sandbox: true})
	if err != nil {
		fmt.Printf("❌ Erro ao inicializar SDK: %v\n", err)
		return
	}

	// ===================================
	// Exemplo 2: Slug não encontrado
	// ===================================
	fmt.Println("2️⃣ Chamando slug inexistente...")
	_, err = client.Call("slug_que_nao_existe", map[string]interface{}{"foo": "bar"})
	if err != nil {
		var epErr *sdkErrors.EndpointNotFoundError
		if errors.As(err, &epErr) {
			fmt.Println("   ✅ EndpointNotFoundError capturado:", epErr.Message)
			fmt.Println("      Código:", epErr.Code)
		} else {
			fmt.Println("   ❌ Outro erro:", err)
		}
	}
	fmt.Println("")

	// ===================================
	// Exemplo 3: Tratamento genérico SDKError
	// ===================================
	fmt.Println("3️⃣ Tratamento genérico de SDKError...")
	simulatedErr := sdkErrors.NewSDKError("Erro simulado", "SIMULATED_ERROR", map[string]interface{}{"foo": "bar"})
	var sdkErr *sdkErrors.SDKError
	if errors.As(simulatedErr, &sdkErr) {
		fmt.Println("   ✅ SDKError capturado")
		fmt.Println("      Mensagem:", sdkErr.Message)
		fmt.Println("      Código:", sdkErr.Code)
		detailsBytes, _ := json.Marshal(sdkErr.Details)
		fmt.Println("      Detalhes:", string(detailsBytes))
		fmt.Println("      Timestamp:", sdkErr.Timestamp)
	}
	fmt.Println("")

	// ===================================
	// Exemplo 4: Padrão recomendado
	// ===================================
	fmt.Println("4️⃣ Padrão recomendado de tratamento...")
	_, err = client.Call("veiculos_agregados", map[string]interface{}{"placa": "ABC1234"})
	if err == nil {
		fmt.Println("   ✅ Requisição bem-sucedida")
	} else {
		var authErr *sdkErrors.AuthenticationError
		var valErr *sdkErrors.ValidationError
		var rlErr *sdkErrors.RateLimitError
		var epErr *sdkErrors.EndpointNotFoundError
		var sdkErr *sdkErrors.SDKError

		if errors.As(err, &authErr) {
			fmt.Println("   🔐 Erro de autenticação - verifique seu token")
		} else if errors.As(err, &valErr) {
			detailsBytes, _ := json.Marshal(valErr.Details)
			fmt.Println("   ⚠️ Dados inválidos:", string(detailsBytes))
		} else if errors.As(err, &rlErr) {
			fmt.Println("   ⏳ Rate limit - aguarde", rlErr.RetryAfter, "segundos")
		} else if errors.As(err, &epErr) {
			fmt.Println("   🔍 Slug não encontrado")
		} else if errors.As(err, &sdkErr) {
			fmt.Println("   ❌ Erro da SDK:", sdkErr.Message)
		} else {
			fmt.Println("   💥 Erro inesperado:", err)
		}
	}
	fmt.Println("")

	fmt.Println("✅ Exemplos de tratamento de erros concluídos")
}
