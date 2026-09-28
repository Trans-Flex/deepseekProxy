package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"golang.org/x/sync/singleflight"
)

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
}

type handler struct {
	cache  *Cache
	key    string
	sf     *singleflight.Group
	client *http.Client
}

func (h *handler) handleChat(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to read request body")
		log.Println(err)
		return
	}

	var req ChatRequest
	err = json.Unmarshal(body, &req)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		log.Println(err)
		return
	}

	system := Message{Role: "system", Content: "你是学习辅助AI, 只做分布引导和学习计划分析, 其余拒绝"}
	req.Messages = append([]Message{system}, req.Messages...)
	req.Model = "deepseek-flash"
	finalBody, err := json.Marshal(req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		log.Println(err)
		return
	}

	tmp := sha256.Sum256(finalBody)
	cacheKey := hex.EncodeToString(tmp[:])
	if data, ok := h.cache.get(cacheKey); ok {
		w.Header().Set("Content-Type", "application/json")
		w.Write(data)
		return
	}

	result, err, _ := h.sf.Do(cacheKey, func() (any, error) {
		if data, ok := h.cache.get(cacheKey); ok {
			return data, nil
		}
		data, err := h.callDeepseek(finalBody)
		if err != nil {
			return nil, err
		}
		err = h.cache.put(cacheKey, data)
		if err != nil {
			log.Println(err)
		}
		return data, nil
	})
	if err != nil {
		log.Println(err)
		writeError(w, http.StatusBadGateway, "upstream failed")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(result.([]byte))
}

func writeError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write([]byte(fmt.Sprintf(`{"error":%q}`, msg)))
}
