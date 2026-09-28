package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
)

func (h *handler) callDeepseek(body []byte) ([]byte, error) {
	req, err := http.NewRequest("POST", "https://api.deepseek.com/chat/completions", bytes.NewReader(body))
	if err != nil {
		return []byte{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+h.key)

	resp, err := h.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("deepseek 返回状态码 %d: %s", resp.StatusCode, respBody)
	}
	return respBody, nil
}
