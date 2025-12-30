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

func GenerateGangQuiz(prompt string) (*QuizResponse, error) {
	apiKey := os.Getenv("GROQ_API_KEY")
	url := "https://api.groq.com/openai/v1/chat/completions"

	payload := map[string]interface{}{
		"model": "llama-3.3-70b-versatile", // ✅ อัปเกรดเป็นตัวที่ฉลาดที่สุดเพื่อป้องกันการมโน
		"messages": []map[string]string{
			{
				"role":    "system",
				"content": "You are the 'Great Sage'. Return ONLY valid JSON. Your answers must be factually accurate and strictly follow the requested category.",
			},
			{
				"role":    "user",
				"content": prompt,
			},
		},
		"response_format": map[string]string{"type": "json_object"},
		"temperature":     0.2, // ✅ ลดความเพ้อเจ้อของ AI ให้ต่ำที่สุด เน้น Fact 100%
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
	var quiz QuizResponse
	if err := json.Unmarshal([]byte(aiContent), &quiz); err != nil {
		return nil, err
	}

	return &quiz, nil
}
