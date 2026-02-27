package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

type SettingsStatusResponse struct {
	BlabladorApiKey string `json:"BLABLADOR_API_KEY"`
	AzureSqlConn    string `json:"AZURE_SQL_CONNECTION_STRING"`
	PowerBiClientId string `json:"POWERBI_CLIENT_ID"`
	OpenAiApiKey    string `json:"OPENAI_API_KEY"`
}

func getStatus(key string) string {
	val := os.Getenv(key)
	if val != "" {
		if len(val) > 4 {
			return "SET (starts with " + val[:4] + "...)"
		}
		return "SET"
	}
	return "NOT SET"
}

func GetSettingsStatusHandler(w http.ResponseWriter, r *http.Request) {
	status := SettingsStatusResponse{
		BlabladorApiKey: getStatus("BLABLADOR_API_KEY"),
		AzureSqlConn:    getStatus("AZURE_SQL_CONNECTION_STRING"),
		PowerBiClientId: getStatus("POWERBI_CLIENT_ID"),
		OpenAiApiKey:    getStatus("OPENAI_API_KEY"),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

func UpdateSettingsHandler(w http.ResponseWriter, r *http.Request) {
	var req map[string]string
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Note: Setting os.Setenv affects the running process but might not persist across restarts in Docker/Space environment
	// unless handled by platform configuration.
	// For MVP, we set it in the process so immediate calls work.

	updated := []string{}
	for k, v := range req {
		if v != "" {
			os.Setenv(k, v)
			updated = append(updated, k)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": fmt.Sprintf("Updated %d settings in process memory.", len(updated)),
		"updated": updated,
	})
}
