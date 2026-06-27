package main

import (
	"encoding/json"
	"fmt"
	"os"

	sdk "github.com/DataCube/consultasdeveiculos-sdk-golang"
)

func main() {
	// Inicializa a SDK em modo Sandbox para testes
	client, err := sdk.NewClient(sdk.Options{
		AuthToken: os.Getenv("API_TOKEN"),
		Sandbox:   true,
	})
	if err != nil {
		fmt.Printf("❌ Erro ao inicializar SDK: %v\n", err)
		return
	}

	// Consulta de Veículo - Agregados
	fmt.Println("🚗 Consultando veículo (agregados)...")
	response, err := client.Call("veiculos_agregados", map[string]interface{}{
		"placa": "LPH9883",
	})
	if err != nil {
		fmt.Printf("❌ Erro: %v\n", err)
		return
	}

	dataBytes, _ := json.MarshalIndent(response.Data, "", "  ")
	fmt.Println("Response:")
	fmt.Println(string(dataBytes))
	fmt.Println("")
}
