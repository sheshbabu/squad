package gateways

import (
	"net/http"
	"squad/commons/llm"
	"squad/commons/utils"
)

func HandleGetGateways(w http.ResponseWriter, r *http.Request) {
	utils.SendJSONResponse(w, http.StatusOK, llm.GetGatewayNames())
}

func HandleGetModels(w http.ResponseWriter, r *http.Request) {
	gateway, ok := llm.GetGateway(r.PathValue("gateway"))
	if !ok {
		utils.SendErrorResponse(w, http.StatusNotFound, "GATEWAY_NOT_FOUND", "Gateway is not configured", nil)
		return
	}

	models, err := gateway.ListModels(r.Context())
	if err != nil {
		utils.SendErrorResponse(w, http.StatusBadGateway, "MODELS_READ_FAILED", "Failed to load models", err)
		return
	}

	utils.SendJSONResponse(w, http.StatusOK, models)
}

func HandleGetProviders(w http.ResponseWriter, r *http.Request) {
	gateway, ok := llm.GetGateway(r.PathValue("gateway"))
	if !ok {
		utils.SendErrorResponse(w, http.StatusNotFound, "GATEWAY_NOT_FOUND", "Gateway is not configured", nil)
		return
	}

	modelID := r.URL.Query().Get("model")
	if modelID == "" {
		utils.SendErrorResponse(w, http.StatusBadRequest, "INVALID_MODEL", "Model is required", nil)
		return
	}

	providers, err := gateway.ListProviders(r.Context(), modelID)
	if err != nil {
		utils.SendErrorResponse(w, http.StatusBadGateway, "PROVIDERS_READ_FAILED", "Failed to load providers", err)
		return
	}

	utils.SendJSONResponse(w, http.StatusOK, providers)
}

func HandleGetZDREndpoints(w http.ResponseWriter, r *http.Request) {
	gateway, ok := llm.GetGateway(r.PathValue("gateway"))
	if !ok {
		utils.SendErrorResponse(w, http.StatusNotFound, "GATEWAY_NOT_FOUND", "Gateway is not configured", nil)
		return
	}

	endpoints, err := gateway.ListZDREndpoints(r.Context())
	if err != nil {
		utils.SendErrorResponse(w, http.StatusBadGateway, "ZDR_ENDPOINTS_READ_FAILED", "Failed to load privacy information", err)
		return
	}

	utils.SendJSONResponse(w, http.StatusOK, endpoints)
}
