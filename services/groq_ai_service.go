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

func GenerateQuizFromGroq(prompt string) (*QuizResponse, error) {
	apiKey := os.Getenv("GROQ_API_KEY")
	url := "https://api.groq.com/openai/v1/chat/completions"

	payload := map[string]interface{}{
		"model": "llama-3.1-8b-instant",
		"messages": []map[string]string{
			{
				"role":    "system",
				"content": "You are a quiz master. Return ONLY a single JSON object. No arrays, no lists, no intro. Ensure it is valid JSON.",
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
	if resp.StatusCode != http.StatusOK {
		fmt.Printf("❌ GROQ ERROR DETAIL: %s\n", string(body))
		return nil, fmt.Errorf("Groq API Error: %d", resp.StatusCode)
	}

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
		return nil, fmt.Errorf("AI choice is empty")
	}

	aiContent := strings.TrimSpace(result.Choices[0].Message.Content)

	// ✅ แก้ไขปัญหา Array: ถ้า AI ส่ง [ {...} ] มา ให้ตัดก้ามปูออก
	if strings.HasPrefix(aiContent, "[") && strings.HasSuffix(aiContent, "]") {
		aiContent = strings.TrimPrefix(aiContent, "[")
		aiContent = strings.TrimSuffix(aiContent, "]")
	}

	var quiz QuizResponse
	if err := json.Unmarshal([]byte(aiContent), &quiz); err != nil {
		fmt.Printf("❌ Raw JSON Error Content: %s\n", aiContent)
		return nil, fmt.Errorf("failed to parse AI response: %v", err)
	}

	return &quiz, nil
}
