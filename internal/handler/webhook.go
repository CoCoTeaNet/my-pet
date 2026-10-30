package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

func PostWebhook(url string, payload map[string]interface{}) error {
	return postWebhook(url, payload)
}

func postWebhook(url string, payload map[string]interface{}) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(url, "application/json", bytesReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	if resp.StatusCode >= 300 {
		return fmt.Errorf("对方返回状态码 %d", resp.StatusCode)
	}
	return nil
}
