# ConsultadeveiculosSDK

SDK Go dinâmica para consultas de veículos, baseada em coleções Postman.

## 🚀 Visão Geral

Esta SDK funciona como um **Runtime Engine** que consome endpoints definidos em uma coleção Postman, sem necessidade de implementação manual de cada endpoint.

**Como o Go é uma linguagem estaticamente tipada, em vez de Proxy dinâmico, as chamadas são realizadas de forma robusta e segura através do método `client.Call("slug", params)`.**

### Características

- ✅ **Dynamic Endpoint Engine**: Zero funções hardcoded - tudo é derivado e atualizado através da coleção Postman
- ✅ **Slug-based API**: Chamadas simplificadas via `client.Call("veiculos_agregados", params)`
- ✅ **Modo Sandbox**: Teste e simulações rápidas sem custo e sem conexão com a API real
- ✅ **178 endpoints**: Gerados automaticamente e prontos para uso a partir da especificação Postman
- ✅ **CLI Completo**: Utilitário em linha de comando para listar, testar e atualizar a SDK

## 📦 Instalação

```bash
go get github.com/DataCube/consultasdeveiculos-sdk-golang
```

---

## 🏁 Início Rápido

### Modo Produção

```go
package main

import (
	"encoding/json"
	"fmt"
	"os"

	sdk "github.com/DataCube/consultasdeveiculos-sdk-golang"
)

func main() {
	// Inicializa com o token (carregado automaticamente do arquivo .env ou do ambiente)
	client, err := sdk.NewClient(sdk.Options{
		AuthToken: os.Getenv("API_TOKEN"),
	})
	if err != nil {
		fmt.Printf("Erro de inicialização: %v\n", err)
		return
	}

	// Consulta usando o slug do endpoint
	resultado, err := client.Call("veiculos_agregados", map[string]interface{}{
		"placa": "ABC1234",
	})
	if err != nil {
		fmt.Printf("Erro na requisição: %v\n", err)
		return
	}

	dataBytes, _ := json.MarshalIndent(resultado.Data, "", "  ")
	fmt.Println(string(dataBytes))
}
```

### Modo Sandbox

```go
package main

import (
	"encoding/json"
	"fmt"

	sdk "github.com/DataCube/consultasdeveiculos-sdk-golang"
)

func main() {
	// Inicializa em modo sandbox (sem token necessário)
	client, err := sdk.NewClient(sdk.Options{
		Sandbox: true,
	})
	if err != nil {
		fmt.Printf("Erro de inicialização: %v\n", err)
		return
	}

	// Retorna respostas simuladas instantaneamente
	resultado, err := client.Call("veiculos_agregados", map[string]interface{}{
		"placa": "ABC1234",
	})
	if err != nil {
		fmt.Printf("Erro na requisição: %v\n", err)
		return
	}

	dataBytes, _ := json.MarshalIndent(resultado.Data, "", "  ")
	fmt.Println("Resposta Sandbox:", string(dataBytes))
}
```

---

## 📖 API

### Opções de Inicialização

```go
client, err := sdk.NewClient(sdk.Options{
	AuthToken:           os.Getenv("API_TOKEN"), // Carregado do arquivo .env ou do ambiente (obrigatório em produção)
	Sandbox:             false,                  // Modo sandbox (padrão: false)
	BaseURL:             "URL",      // URL base customizada (opcional)
	Timeout:             30000,      // Timeout das requisições (padrão: 30s)
	MaxRetries:          3,          // Máximo de retries (padrão: 3)
	DisableCompression:  false,      // Desativa compressão Gzip (padrão: false)
})
```

### Como Chamar Endpoints

O slug é derivado diretamente da URL do endpoint na especificação Postman, convertendo `/` e `-` para `_`:

| URL do Endpoint | Slug para Chamar |
|-----------------|------------------|
| `/veiculos/agregados` | `client.Call("veiculos_agregados", params)` |
| `/veiculos/debitos-sp` | `client.Call("veiculos_debitos_sp", params)` |
| `/cnh/nacional/simples` | `client.Call("cnh_nacional_simples", params)` |
| `/pessoas/nome` | `client.Call("pessoas_nome", params)` |

```go
// Consulta de veículo
veiculo, err := client.Call("veiculos_agregados", map[string]interface{}{ "placa": "ABC1234" })

// Consulta de CNH
cnh, err := client.Call("cnh_nacional_simples", map[string]interface{}{ 
	"cnh":             "12345678901",
	"data_nascimento": "01/01/1990",
})

// Consulta de pessoa por CPF
pessoa, err := client.Call("pessoas_nome", map[string]interface{}{ "cpf": "123.456.789-00" })
```

---

### Métodos Auxiliares de Ajuda

A SDK inclui métodos idênticos para explorar e debugar os endpoints disponíveis:

```go
client, _ := sdk.NewClient(sdk.Options{ Sandbox: true })

// Exibe ajuda completa no console
client.Help("")

// Filtra endpoints por termo no console
client.Help("veiculos")
client.Help("cnh")

// Lista todos os endpoints agrupados por namespace
client.Endpoints()

// Informações detalhadas da SDK
info := client.GetInfo()
// info.RuntimeVersion, info.SpecVersion, info.Sandbox, info.EndpointsCount, info.Namespaces

// Busca endpoints por termo
results := client.Search("debitos")
```

---

## 🖥️ CLI (Command Line Interface)

A SDK possui um binário CLI em Go pronto para execução rápida, eliminando qualquer dependência externa.

### Comandos Disponíveis

```bash
# Compilar o CLI
go build -o consultas-cli ./cmd/consultas-de-veiculos-sdk

# Listar todos os endpoints
./consultas-cli endpoints

# Filtrar por namespace
./consultas-cli endpoints veiculos
./consultas-cli endpoints cnh

# Ver descrições e URLs detalhadas
./consultas-cli endpoints --verbose

# Saída em formato JSON
./consultas-cli endpoints --json

# Ver versão da SDK e da especificação local
./consultas-cli version

# Diagnóstico completo do ambiente (rede, caminhos, permissões, integridade)
./consultas-cli doctor

# Baixar e persistir a última especificação da API
./consultas-cli update

# Limpar cache local da especificação
./consultas-cli clear-cache --force
```

---

## 🛡️ Tratamento de Erros

A SDK expõe uma estrutura hierárquica baseada no padrão idiomatico do Go com `errors.As`.

```go
import (
	"errors"
	"fmt"
	
	sdk "github.com/DataCube/consultasdeveiculos-sdk-golang"
	sdkErrors "github.com/DataCube/consultasdeveiculos-sdk-golang/errors"
)

// ...
resultado, err := client.Call("veiculos_agregados", map[string]interface{}{"placa": "ABC1234"})
if err != nil {
	var authErr *sdkErrors.AuthenticationError
	var valErr *sdkErrors.ValidationError
	var rlErr *sdkErrors.RateLimitError
	var epErr *sdkErrors.EndpointNotFoundError
	var sdkErr *sdkErrors.SDKError

	if errors.As(err, &authErr) {
		fmt.Println("Erro de Autenticação - token inválido:", authErr.Message)
	} else if errors.As(err, &valErr) {
		fmt.Printf("Erro de Validação: %v\n", valErr.Details)
	} else if errors.As(err, &rlErr) {
		fmt.Printf("Limite atingido, aguarde %d segundos\n", rlErr.RetryAfter)
	} else if errors.As(err, &epErr) {
		fmt.Println("Endpoint / Slug não existe")
	} else if errors.As(err, &sdkErr) {
		fmt.Println("Erro genérico da SDK:", sdkErr.Message)
	}
}
```

### Hierarquia de Erros

```
SDKError (Estrutura base)
├── AuthenticationError   (Status 401, 403 - token inválido/expirado)
├── ValidationError       (Status 400, 422 - dados de envio inválidos)
├── RateLimitError        (Status 429 - limite de requisições por minuto atingido)
├── EndpointNotFoundError  (Slug solicitado não está presente na especificação)
└── SpecificationError     (Erro ao ler, verificar assinatura ou salvar postman.json)
```

---

## 📡 Namespaces Disponíveis

| Namespace | Endpoints | Descrição |
|-----------|-----------|-----------|
| conta | 2 | Informações da conta e consumo |
| assincrono | 3 | Consultas assíncronas (criar/buscar/recuperar tasks) |
| cadastros | 11 | Dados cadastrais de pessoas e empresas |
| orgaos | 18 | Consultas em órgãos públicos (Sintegra, Receita, etc) |
| credito | 6 | Análise de crédito e score |
| veiculos | 74 | Consultas de veículos (placa, agregados, débitos) |
| cnh | 25 | Consultas de CNH (nacional e estaduais) |
| inmetro | 2 | Dados do INMETRO |
| reclameAqui | 5 | Consultas no Reclame Aqui |

---

## 📁 Estrutura do Projeto Go

```
consultasdeveiculos-sdk-golang/
│
├── 📄 go.mod                     # Configurações do módulo Go
├── 📄 README.md                  # Esta documentação
├── 📄 .env.example               # Exemplo de configuração (.env opcional)
│
├── 📁 cmd/                       # Fontes dos binários executáveis
│   └── 📁 consultas-de-veiculos-sdk/
│       └── 📄 main.go            # Entrypoint e CLI da SDK
│
├── 📁 core/                      # Regras de Negócio e Componentes Core
│   ├── 📄 config.go              # Gerenciador de diretórios, cache e configurações
│   ├── 📄 env.go                 # Parser opcional e leve de arquivos .env
│   ├── 📄 loader.go              # Carregamento inteligente (Cache -> Disco -> Embedded)
│   ├── 📄 manifest.go            # Metadados e limites de compatibilidade da Spec
│   └── 📄 registry.go            # Armazenamento e indexador de endpoints
│
├── 📁 errors/                    # Pacote de erros customizados
│   └── 📄 errors.go              # Tipos estruturados, sanitização e suporte errors.As
│
├── 📁 parser/                    # Leitor de Coleções Postman
│   └── 📄 parser.go              # Mapeamento do JSON da Spec e cálculo de estatísticas
│
├── 📁 transport/                 # Camada de Rede (HTTP / Mock Sandbox)
│   ├── 📄 transport.go           # Interfaces de comunicação e construções de payload
│   ├── 📄 http.go                # HTTP real com Exponential Backoff
│   └── 📄 sandbox.go             # Mocking offline baseado nos exemplos da spec
│
├── 📁 spec/                      # Arquivos locais compilados da Especificação
│   ├── 📄 postman.json           # Cópia compilada embarcada da spec
│   └── 📄 manifest.json          # Metadados compilados embarcados
│
├── 📁 examples/                  # Diretório com exemplos práticos
│   ├── 📁 basic_usage/           # Exemplo básico de uso
│   ├── 📁 sandbox_mode/          # Demonstrações de execução offline (Sandbox)
│   ├── 📁 error_handling/        # Tratamento seguro de erros tipados
│   └── 📁 explore_endpoints/     # Exploração interativa de slugs
│
└── 📄 sdk.go                     # Fachada principal (Client, NewClient, Call)
```

---

## 🧪 Executando Exemplos e Testes

### Executar Testes
```bash
go test -v ./...
```

### Executar Exemplos
```bash
# Rodar exemplo de Sandbox
go run examples/sandbox_mode/main.go

# Rodar exemplo de Tratamento de Erros
go run examples/error_handling/main.go
```

## 📄 Licença

MIT
