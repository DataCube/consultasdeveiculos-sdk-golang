package consultasdeveiculos

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/DataCube/consultasdeveiculos-sdk-golang/core"
	sdkErrors "github.com/DataCube/consultasdeveiculos-sdk-golang/errors"
	"github.com/DataCube/consultasdeveiculos-sdk-golang/parser"
)

func TestSDK_Inicializacao(t *testing.T) {
	t.Run("deve lancar erro sem token em modo producao", func(t *testing.T) {
		_, err := NewClient(Options{})
		if err == nil {
			t.Fatal("esperava erro ao inicializar sem token em modo producao")
		}
		var authErr *sdkErrors.AuthenticationError
		if !errors.As(err, &authErr) {
			t.Fatalf("esperava erro do tipo AuthenticationError, obteve: %v", err)
		}
	})

	t.Run("deve inicializar em modo sandbox sem token", func(t *testing.T) {
		client, err := NewClient(Options{Sandbox: true})
		if err != nil {
			t.Fatalf("erro ao inicializar em modo sandbox: %v", err)
		}
		if !client.Sandbox {
			t.Fatal("esperava que o cliente estivesse em modo sandbox")
		}
	})

	t.Run("deve ter versao definida", func(t *testing.T) {
		if VERSION == "" {
			t.Fatal("versao nao deve ser vazia")
		}
		parts := strings.Split(VERSION, ".")
		if len(parts) != 3 {
			t.Fatalf("versao no formato incorreto: %s", VERSION)
		}
	})
}

func TestSDK_SandboxMode(t *testing.T) {
	client, err := NewClient(Options{Sandbox: true})
	if err != nil {
		t.Fatalf("erro ao inicializar client: %v", err)
	}

	t.Run("deve listar endpoints", func(t *testing.T) {
		endpoints := client.ListEndpoints()
		if len(endpoints) == 0 {
			t.Fatal("esperava pelo menos um endpoint listado")
		}
	})

	t.Run("deve ter namespaces definidos", func(t *testing.T) {
		info := client.GetInfo()
		if len(info.Namespaces) == 0 {
			t.Fatal("esperava pelo menos um namespace")
		}
	})

	t.Run("deve retornar resposta de exemplo", func(t *testing.T) {
		res, err := client.Call("veiculos_agregados", map[string]interface{}{
			"placa": "ABC1234",
		})
		if err != nil {
			t.Fatalf("erro na chamada: %v", err)
		}
		if !res.Success {
			t.Fatal("resposta nao sucedida")
		}
		if !res.Sandbox {
			t.Fatal("resposta deveria ser sandbox")
		}
	})
}

func TestSDK_PostmanParser(t *testing.T) {
	mockCollection := &parser.PostmanCollection{
		Info: parser.PostmanInfo{
			Name: "Test Collection",
		},
		Item: []parser.PostmanItem{
			{
				Name: "Categoria",
				Item: []parser.PostmanItem{
					{
						Name: "Endpoint Teste",
						Request: &parser.PostmanRequest{
							Method: "GET",
							URL: parser.PostmanURL{
								Raw: "https://api.test.com/endpoint",
							},
						},
					},
				},
			},
		},
	}

	p := parser.NewPostmanParser()
	endpoints := p.Parse(mockCollection)

	if len(endpoints) != 1 {
		t.Fatalf("esperava 1 endpoint, obteve %d", len(endpoints))
	}
	if endpoints[0].Key != "categoria.endpointTeste" {
		t.Fatalf("esperava key 'categoria.endpointTeste', obteve '%s'", endpoints[0].Key)
	}

	stats := p.GetStats(mockCollection)
	if stats.TotalEndpoints != 1 {
		t.Fatalf("esperava 1 endpoint nas estatisticas, obteve %d", stats.TotalEndpoints)
	}
	foundNamespace := false
	for _, ns := range stats.Namespaces {
		if ns == "categoria" {
			foundNamespace = true
			break
		}
	}
	if !foundNamespace {
		t.Fatal("esperava namespace 'categoria'")
	}
}

func TestSDK_Errors(t *testing.T) {
	t.Run("SDKError deve ter propriedades corretas", func(t *testing.T) {
		err := sdkErrors.NewSDKError("Test error", "TEST_CODE", map[string]interface{}{"foo": "bar"})
		if err.Message != "Test error" {
			t.Fatalf("mensagem incorreta: %s", err.Message)
		}
		if err.Code != "TEST_CODE" {
			t.Fatalf("codigo incorreto: %s", err.Code)
		}

		detailsMap, ok := err.Details.(map[string]interface{})
		if !ok {
			t.Fatalf("esperava map[string]interface{}, obteve %T", err.Details)
		}
		if detailsMap["foo"] != "bar" {
			t.Fatalf("detalhes incorretos: %v", err.Details)
		}
		if err.Timestamp == "" {
			t.Fatal("timestamp nao deve ser vazio")
		}
	})

	t.Run("AuthenticationError deve ser assinalavel", func(t *testing.T) {
		err := sdkErrors.NewAuthenticationError("Invalid token", nil)
		var sdkErr *sdkErrors.SDKError
		if !errors.As(err, &sdkErr) {
			t.Fatal("AuthenticationError deve ser assecuravel como SDKError")
		}
		if sdkErr.Code != "AUTHENTICATION_ERROR" {
			t.Fatalf("codigo incorreto: %s", sdkErr.Code)
		}
	})
}

func TestSDK_LoadEnv(t *testing.T) {
	err := os.WriteFile(".env", []byte("TEST_ENV_VAR=hello_world\n# Comment\nAPI_TOKEN=mocked_token"), 0644)
	if err != nil {
		t.Fatalf("erro ao criar arquivo .env: %v", err)
	}
	defer os.Remove(".env")

	core.LoadEnv()

	if os.Getenv("TEST_ENV_VAR") != "hello_world" {
		t.Fatalf("esperava TEST_ENV_VAR=hello_world, obteve '%s'", os.Getenv("TEST_ENV_VAR"))
	}
	if os.Getenv("API_TOKEN") != "mocked_token" {
		t.Fatalf("esperava API_TOKEN=mocked_token, obteve '%s'", os.Getenv("API_TOKEN"))
	}
}
