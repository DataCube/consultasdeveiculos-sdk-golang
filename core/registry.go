package core

import (
	"regexp"
	"strings"

	"github.com/DataCube/consultasdeveiculos-sdk-golang/errors"
	"github.com/DataCube/consultasdeveiculos-sdk-golang/parser"
)

// EndpointRegistry manages and stores all parsed endpoints.
type EndpointRegistry struct {
	Endpoints     map[string]parser.Endpoint
	NamespaceTree map[string]interface{}
}

func NewEndpointRegistry() *EndpointRegistry {
	return &EndpointRegistry{
		Endpoints:     make(map[string]parser.Endpoint),
		NamespaceTree: make(map[string]interface{}),
	}
}

// Register registers an endpoint in the registry.
func (r *EndpointRegistry) Register(endpoint parser.Endpoint) {
	key := endpoint.Key
	if key == "" {
		panic("Endpoint deve ter uma key")
	}

	r.Endpoints[key] = endpoint
	r.addToNamespaceTree(key, endpoint)
}

// Get retrieves an endpoint by key.
func (r *EndpointRegistry) Get(key string) (parser.Endpoint, error) {
	ep, exists := r.Endpoints[key]
	if !exists {
		return parser.Endpoint{}, errors.NewEndpointNotFoundError("Endpoint \""+key+"\" não encontrado", map[string]interface{}{"key": key})
	}
	return ep, nil
}

// Has checks if an endpoint exists.
func (r *EndpointRegistry) Has(key string) bool {
	_, exists := r.Endpoints[key]
	return exists
}

// List returns all registered endpoints as a slice.
func (r *EndpointRegistry) List() []parser.Endpoint {
	var list []parser.Endpoint
	for _, ep := range r.Endpoints {
		list = append(list, ep)
	}
	return list
}

// ListByNamespace returns all endpoints under a given namespace.
func (r *EndpointRegistry) ListByNamespace(namespace string) []parser.Endpoint {
	var list []parser.Endpoint
	prefix := namespace + "."
	for k, ep := range r.Endpoints {
		if strings.HasPrefix(k, prefix) {
			list = append(list, ep)
		}
	}
	return list
}

// GetNamespaceTree returns the structured namespace tree.
func (r *EndpointRegistry) GetNamespaceTree() map[string]interface{} {
	return r.NamespaceTree
}

// GetNamespaces returns all first-level namespaces.
func (r *EndpointRegistry) GetNamespaces() []string {
	var list []string
	for k := range r.NamespaceTree {
		list = append(list, k)
	}
	return list
}

// Clear clears all endpoints and namespaces.
func (r *EndpointRegistry) Clear() {
	r.Endpoints = make(map[string]parser.Endpoint)
	r.NamespaceTree = make(map[string]interface{})
}

// Size returns the count of registered endpoints.
func (r *EndpointRegistry) Size() int {
	return len(r.Endpoints)
}

func (r *EndpointRegistry) addToNamespaceTree(key string, ep parser.Endpoint) {
	parts := strings.Split(key, ".")
	current := r.NamespaceTree

	for i := 0; i < len(parts)-1; i++ {
		part := parts[i]
		if _, exists := current[part]; !exists {
			newNode := make(map[string]interface{})
			newNode["_methods"] = make(map[string]parser.Endpoint)
			current[part] = newNode
		}

		nextNode, ok := current[part].(map[string]interface{})
		if !ok {
			nextNode = make(map[string]interface{})
			nextNode["_methods"] = make(map[string]parser.Endpoint)
			current[part] = nextNode
		}
		current = nextNode
	}

	methodName := parts[len(parts)-1]
	methods, exists := current["_methods"]
	if !exists {
		methods = make(map[string]parser.Endpoint)
		current["_methods"] = methods
	}

	methodsMap, ok := methods.(map[string]parser.Endpoint)
	if !ok {
		methodsMap = make(map[string]parser.Endpoint)
		current["_methods"] = methodsMap
	}
	methodsMap[methodName] = ep
	current["_methods"] = methodsMap
}

// Search searches for endpoints matching a pattern.
func (r *EndpointRegistry) Search(pattern string) []parser.Endpoint {
	re, err := regexp.Compile("(?i)" + regexp.QuoteMeta(pattern))
	if err != nil {
		return nil
	}

	var results []parser.Endpoint
	for _, ep := range r.Endpoints {
		if re.MatchString(ep.Key) || re.MatchString(ep.Name) || re.MatchString(ep.Description) {
			results = append(results, ep)
		}
	}
	return results
}
