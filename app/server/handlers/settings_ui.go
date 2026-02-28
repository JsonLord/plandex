package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

type SettingStatus struct {
	IsSet   bool   `json:"is_set"`
	Message string `json:"message"`
}

type SettingsStatusResponse struct {
	BlabladorApiKey SettingStatus `json:"BLABLADOR_API_KEY"`
	AzureSqlConn    SettingStatus `json:"AZURE_SQL_CONNECTION_STRING"`
	PowerBiClientId SettingStatus `json:"POWERBI_CLIENT_ID"`
	OpenAiApiKey    SettingStatus `json:"OPENAI_API_KEY"`
}

func getStatus(key string) SettingStatus {
	val := os.Getenv(key)
	if val != "" {
		preview := ""
		if len(val) > 4 {
			preview = " (starts with " + val[:4] + "...)"
		}
		return SettingStatus{
			IsSet:   true,
			Message: "✅ Configured via Space Secrets" + preview,
		}
	}
	return SettingStatus{
		IsSet:   false,
		Message: "❌ Not Set",
	}
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

func UpdateEnvironmentSettingsHandler(w http.ResponseWriter, r *http.Request) {
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
