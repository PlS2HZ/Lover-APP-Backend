package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

type QuizResponse struct {
	Question     string   `json:"question"`
	Options      []string `json:"options"`
	AnswerIndex  int      `json:"answer_index"`
	SweetComment string   `json:"sweet_comment"`
}

// ตรวจสอบชื่อฟังก์ชันตรงนี้: ต้องเป็น GenerateGangQuiz
func GenerateGangQuiz(prompt string) (*QuizResponse, error) {
	apiKey := os.Getenv("GROQ_API_KEY")
	url := "https://api.groq.com/openai/v1/chat/completions"

	payload := map[string]interface{}{
		"model": "llama-3.1-8b-instant",
		"messages": []map[string]string{
			{
				"role":    "system",
				"content": "You are the 'Great Sage', an omniscient and wise entity. Return ONLY a single valid JSON object.",
			},
			{
				"role":    "user",
				"content": prompt,
			},
		},
		"response_format": map[string]string{"type": "json_object"},
		"temperature":     0.5,
	}

	jsonData, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	if len(result.Choices) == 0 {
		return nil, fmt.Errorf("AI response is empty")
	}

	aiContent := strings.TrimSpace(result.Choices[0].Message.Content)
	aiContent = strings.TrimPrefix(aiContent, "```json")
	aiContent = strings.TrimSuffix(aiContent, "```")

	var quiz QuizResponse
	if err := json.Unmarshal([]byte(aiContent), &quiz); err != nil {
		return nil, err
	}

	return &quiz, nil
}
