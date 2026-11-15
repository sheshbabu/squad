package llm

import (
	"context"
	"fmt"
	"slices"
	"sort"
	"sync"
	"time"
)

type Model struct {
	ModelID                string   `json:"modelId"`
	Name                   string   `json:"name"`
	ContextLength          int      `json:"contextLength"`
	InputModalities        []string `json:"inputModalities"`
	PromptPrice            string   `json:"promptPrice"`
	CompletionPrice        string   `json:"completionPrice"`
	ReasoningEfforts       []string `json:"reasoningEfforts"`
	DefaultReasoningEffort string   `json:"defaultReasoningEffort"`
}

type Provider struct {
	ProviderID string `json:"providerId"`
	Name       string `json:"name"`
}

// ProviderID is the same tag bots pin to, so endpoints compare directly against a bot's model and provider.
type Endpoint struct {
	ModelID    string `json:"modelId"`
	ProviderID string `json:"providerId"`
}

const modelsCacheDuration = time.Hour

var modelsCacheMutex sync.Mutex
var modelsCache = map[string][]Model{}
var modelsCachedAt = map[string]time.Time{}

func (g Gateway) ListModels(ctx context.Context) ([]Model, error) {
	modelsCacheMutex.Lock()
	defer modelsCacheMutex.Unlock()

	if cached, ok := modelsCache[g.Name]; ok && time.Since(modelsCachedAt[g.Name]) < modelsCacheDuration {
		return cached, nil
	}

	var body struct {
		Data []struct {
			ID            string `json:"id"`
			Name          string `json:"name"`
			ContextLength int    `json:"context_length"`
			Architecture  struct {
				InputModalities []string `json:"input_modalities"`
			} `json:"architecture"`
			Pricing struct {
				Prompt     string `json:"prompt"`
				Completion string `json:"completion"`
			} `json:"pricing"`
			SupportedParameters []string `json:"supported_parameters"`
			Reasoning           struct {
				SupportedEfforts []string `json:"supported_efforts"`
				DefaultEffort    string   `json:"default_effort"`
			} `json:"reasoning"`
		} `json:"data"`
	}

	if err := g.get(ctx, "/models", &body); err != nil {
		return nil, fmt.Errorf("error listing models: %w", err)
	}

	models := []Model{}
	for _, item := range body.Data {
		// Bots are agents, so a model that can't call tools is no use here.
		if g.SupportsProviderRouting && !slices.Contains(item.SupportedParameters, "tools") {
			continue
		}

		models = append(models, Model{
			ModelID:                item.ID,
			Name:                   item.Name,
			ContextLength:          item.ContextLength,
			InputModalities:        item.Architecture.InputModalities,
			PromptPrice:            item.Pricing.Prompt,
			CompletionPrice:        item.Pricing.Completion,
			ReasoningEfforts:       item.Reasoning.SupportedEfforts,
			DefaultReasoningEffort: item.Reasoning.DefaultEffort,
		})
	}

	sort.Slice(models, func(i, j int) bool {
		return models[i].Name < models[j].Name
	})

	modelsCache[g.Name] = models
	modelsCachedAt[g.Name] = time.Now()

	return models, nil
}

var zdrEndpointsCache = map[string][]Endpoint{}
var zdrEndpointsCachedAt = map[string]time.Time{}

func (g Gateway) ListZDREndpoints(ctx context.Context) ([]Endpoint, error) {
	endpoints := []Endpoint{}

	if !g.SupportsProviderRouting {
		return endpoints, nil
	}

	modelsCacheMutex.Lock()
	defer modelsCacheMutex.Unlock()

	if cached, ok := zdrEndpointsCache[g.Name]; ok && time.Since(zdrEndpointsCachedAt[g.Name]) < modelsCacheDuration {
		return cached, nil
	}

	var body struct {
		Data []struct {
			ModelID string `json:"model_id"`
			Tag     string `json:"tag"`
		} `json:"data"`
	}

	if err := g.get(ctx, "/endpoints/zdr", &body); err != nil {
		return nil, fmt.Errorf("error listing zdr endpoints: %w", err)
	}

	for _, item := range body.Data {
		endpoints = append(endpoints, Endpoint{ModelID: item.ModelID, ProviderID: item.Tag})
	}

	zdrEndpointsCache[g.Name] = endpoints
	zdrEndpointsCachedAt[g.Name] = time.Now()

	return endpoints, nil
}

func (g Gateway) ListProviders(ctx context.Context, modelID string) ([]Provider, error) {
	providers := []Provider{}

	if !g.SupportsProviderRouting {
		return providers, nil
	}

	var body struct {
		Data struct {
			Endpoints []struct {
				ProviderName string `json:"provider_name"`
				Tag          string `json:"tag"`
			} `json:"endpoints"`
		} `json:"data"`
	}

	if err := g.get(ctx, "/models/"+modelID+"/endpoints", &body); err != nil {
		return nil, fmt.Errorf("error listing providers: %w", err)
	}

	for _, endpoint := range body.Data.Endpoints {
		providers = append(providers, Provider{ProviderID: endpoint.Tag, Name: endpoint.ProviderName})
	}

	sort.Slice(providers, func(i, j int) bool {
		return providers[i].ProviderID < providers[j].ProviderID
	})

	return providers, nil
}
