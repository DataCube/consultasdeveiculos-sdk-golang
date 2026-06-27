package core

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"time"

	sdkErrors "github.com/DataCube/consultasdeveiculos-sdk-golang/errors"
	"github.com/DataCube/consultasdeveiculos-sdk-golang/parser"
)

type Manifest struct {
	SpecVersion       string `json:"specVersion"`
	MinRuntimeVersion string `json:"minRuntimeVersion"`
	GeneratedAt       string `json:"generatedAt"`
}

type LoaderResult struct {
	Postman  *parser.PostmanCollection
	Manifest *Manifest
	Source   string
}

type PostmanLoader struct {
	configManager   *ConfigManager
	defaultSpec     []byte
	defaultManifest []byte
	specDir         string
}

func NewPostmanLoader(cm *ConfigManager, defaultSpec []byte, defaultManifest []byte, specDir string) *PostmanLoader {
	return &PostmanLoader{
		configManager:   cm,
		defaultSpec:     defaultSpec,
		defaultManifest: defaultManifest,
		specDir:         specDir,
	}
}

func (l *PostmanLoader) Load() (*LoaderResult, error) {
	if l.configManager.HasLocalCache() {
		res, err := l.loadFromCache()
		if err == nil {
			return res, nil
		}
	}
	return l.loadFromPackage()
}

func (l *PostmanLoader) loadFromCache() (*LoaderResult, error) {
	postmanPath := l.configManager.GetCachedPostmanPath()
	manifestPath := l.configManager.GetCachedManifestPath()

	postmanBytes, err := os.ReadFile(postmanPath)
	if err != nil {
		return nil, sdkErrors.NewSpecificationError("Arquivo postman.json não encontrado no cache", nil)
	}

	manifestBytes, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, sdkErrors.NewSpecificationError("Arquivo manifest.json não encontrado no cache", nil)
	}

	var postman parser.PostmanCollection
	if err := json.Unmarshal(postmanBytes, &postman); err != nil {
		return nil, sdkErrors.NewSpecificationError("Erro ao fazer parse do postman.json no cache", nil)
	}

	var manifest Manifest
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		return nil, sdkErrors.NewSpecificationError("Erro ao fazer parse do manifest.json no cache", nil)
	}

	if err := l.validatePostman(&postman); err != nil {
		return nil, err
	}
	if err := l.validateManifest(&manifest); err != nil {
		return nil, err
	}

	return &LoaderResult{
		Postman:  &postman,
		Manifest: &manifest,
		Source:   "cache",
	}, nil
}

func (l *PostmanLoader) loadFromPackage() (*LoaderResult, error) {
	specDir := l.specDir
	if specDir == "" {
		specDir = "spec"
	}

	postmanPath := l.configManager.FindPostmanFile(specDir)
	manifestPath := filepath.Join(specDir, "manifest.json")

	var postmanBytes []byte
	var err error
	var filename string

	if postmanPath != "" {
		postmanBytes, err = os.ReadFile(postmanPath)
		filename = filepath.Base(postmanPath)
	} else if len(l.defaultSpec) > 0 {
		postmanBytes = l.defaultSpec
		filename = "postman.json"
	} else {
		return nil, sdkErrors.NewSpecificationError("Arquivo Postman não encontrado. Execute update para baixar a especificação.", nil)
	}

	if err != nil {
		return nil, sdkErrors.NewSpecificationError(fmt.Sprintf("Erro ao ler especificação: %s", err.Error()), nil)
	}

	var postman parser.PostmanCollection
	if err := json.Unmarshal(postmanBytes, &postman); err != nil {
		return nil, sdkErrors.NewSpecificationError("Erro ao fazer parse do postman.json", nil)
	}

	var manifest Manifest
	loadedManifest := false
	if _, errStat := os.Stat(manifestPath); errStat == nil {
		var manifestBytes []byte
		manifestBytes, err = os.ReadFile(manifestPath)
		if err == nil {
			if errUnmarshal := json.Unmarshal(manifestBytes, &manifest); errUnmarshal == nil {
				loadedManifest = true
			}
		}
	} else if len(l.defaultManifest) > 0 {
		if errUnmarshal := json.Unmarshal(l.defaultManifest, &manifest); errUnmarshal == nil {
			loadedManifest = true
		}
	}

	if !loadedManifest {
		specVersion := "1.0.0"
		if filename != "" {
			extracted := l.configManager.ExtractVersionFromFilename(filename)
			if extracted != "" {
				specVersion = extracted
			}
		}
		if postman.Info.Version != "" {
			specVersion = postman.Info.Version
		} else {
			re := regexp.MustCompile(`(?i)V([\d.]+)`)
			match := re.FindStringSubmatch(postman.Info.Name)
			if len(match) > 1 {
				specVersion = match[1]
			}
		}

		manifest = Manifest{
			SpecVersion:       specVersion,
			MinRuntimeVersion: "1.0.0",
			GeneratedAt:       time.Now().UTC().Format(time.RFC3339),
		}
	}

	if manifest.SpecVersion == "" && filename != "" {
		manifest.SpecVersion = l.configManager.ExtractVersionFromFilename(filename)
		if manifest.SpecVersion == "" {
			manifest.SpecVersion = "1.0.0"
		}
	}

	if err := l.validatePostman(&postman); err != nil {
		return nil, err
	}
	if err := l.validateManifest(&manifest); err != nil {
		return nil, err
	}

	return &LoaderResult{
		Postman:  &postman,
		Manifest: &manifest,
		Source:   "package",
	}, nil
}

func (l *PostmanLoader) validatePostman(postman *parser.PostmanCollection) error {
	if postman == nil {
		return sdkErrors.NewSpecificationError("Coleção Postman vazia", nil)
	}
	if postman.Info.Name == "" {
		return sdkErrors.NewSpecificationError("Coleção Postman sem informações (info)", nil)
	}
	if len(postman.Item) == 0 {
		return sdkErrors.NewSpecificationError("Coleção Postman sem itens", nil)
	}
	return nil
}

func (l *PostmanLoader) validateManifest(manifest *Manifest) error {
	if manifest == nil {
		return sdkErrors.NewSpecificationError("Manifest vazio", nil)
	}
	if manifest.SpecVersion == "" {
		return sdkErrors.NewSpecificationError("Manifest sem versão da especificação (specVersion)", nil)
	}
	if manifest.MinRuntimeVersion == "" {
		return sdkErrors.NewSpecificationError("Manifest sem versão mínima do runtime (minRuntimeVersion)", nil)
	}
	return nil
}

func (l *PostmanLoader) SaveToCache(postman *parser.PostmanCollection, manifest *Manifest) error {
	postmanPath := l.configManager.GetCachedPostmanPath()
	manifestPath := l.configManager.GetCachedManifestPath()

	postmanBytes, err := json.MarshalIndent(postman, "", "  ")
	if err != nil {
		return err
	}

	manifestBytes, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(postmanPath, postmanBytes, 0644); err != nil {
		return err
	}

	return os.WriteFile(manifestPath, manifestBytes, 0644)
}
