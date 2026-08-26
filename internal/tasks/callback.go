package tasks

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/Ceruvia/grader/internal/config"
	log "github.com/sirupsen/logrus"
)

var callbackConfig *config.CallbackConfig

var httpClient = &http.Client{
	Timeout: 30 * time.Second,
}

func InitCallback(cfg *config.CallbackConfig) {
	callbackConfig = cfg
}

type CallbackPayload struct {
	GradingResult json.RawMessage `json:"grading_result"`
}

func SendCallback(submissionId string, gradingResultJSON string) error {
	if callbackConfig == nil || callbackConfig.BaseURL == "" {
		return fmt.Errorf("callback config not initialized")
	}

	url := fmt.Sprintf("%s/v1/submissions/%s/programming/grade-callback", callbackConfig.BaseURL, submissionId)

	payload := CallbackPayload{
		GradingResult: json.RawMessage(gradingResultJSON),
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal callback payload: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create callback request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", callbackConfig.APIKey)

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send callback request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("callback request failed with status: %d", resp.StatusCode)
	}

	log.WithFields(log.Fields{
		"submissionId": submissionId,
		"statusCode":   resp.StatusCode,
	}).Debug("Callback sent successfully")

	return nil
}
