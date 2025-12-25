package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

func AskGroqRaw(prompt string) string {
	startTime := time.Now()
	apiKey := os.Getenv("GROQ_API_KEY")
	if apiKey == "" {
		return "เออ... ลืมตั้ง GROQ_API_KEY ใน .env นะนาย"
	}

	url := "https://api.groq.com/openai/v1/chat/completions"
	payload := map[string]interface{}{
		"model":             "llama-3.3-70b-versatile",
		"messages":          []map[string]interface{}{{"role": "user", "content": prompt}},
		"temperature":       0.6,
		"max_tokens":        100,
		"top_p":             0.8,
		"frequency_penalty": 0.6,
		"presence_penalty":  0.3,
	}

	jsonData, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "Network Error กับ Groq จ้า"
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var groqResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.Unmarshal(body, &groqResp); err != nil || len(groqResp.Choices) == 0 {
		return "Groq สำลักจ้า ลองใหม่นะ"
	}

	result := strings.TrimSpace(groqResp.Choices[0].Message.Content)
	if strings.Contains(result, "<|") || strings.Contains(result, "aff ") {
		return "อือหือ... เมื่อกี้เครื่องค้างไปนิด เอาเป็นว่าถามใหม่สิ!"
	}

	fmt.Printf("\n⚡ [Groq Speed]: %v | Result: %s\n", time.Since(startTime), result)
	return result
}

func AskGroq(secretWord string, description string, question string) string {
	prompt := fmt.Sprintf(`
    นายคือ 'Bot' เกมทายคำที่คุยสนุก (เหมือนเพื่อนแชทกัน)
    คำลับคือ: "%s" | บริบท: %s

    [กฎเหล็กการตอบ]
    1. ห้ามสโคปคำตอบจนแคบเกินไป (เช่น ถ้าเขาถามว่าสิ่งของไหม ห้ามชิงตอบว่าเป็นสัตว์เลี้ยงทันที ให้ตอบแค่ ใช่/ไม่ใช่ ตามความจริง)
    2. ห้ามเริ่มประโยคด้วย "อ่า..." หรือ Pattern ซ้ำๆ ให้เข้าเนื้อหาทันที
    3. ตอบสั้น กระชับ จบใน 1 ประโยค และต้องสื่อถึง ใช่ หรือ ไม่ใช่ ห้ามตอบคำเดียวทื่อๆ
    4. หากถูกถามเรื่อง "จำนวนพยางค์" หรือ "ตัวอักษร" ให้ตอบว่า "บอกไม่ได้หรอก"
    5. ห้ามหลุดคำว่า "%s" หรือส่วนประกอบของคำออกมา
    6. หากผู้ใช้ทายใกล้เคียงมาก ให้ยอมรับว่าใกล้เคียงสุดๆ

    คำถาม: "%s"`, secretWord, description, secretWord, question)
	return AskGroqRaw(prompt)
}

func AskGroqHint(secretWord string, description string) string {
	prompt := fmt.Sprintf(`
    จงใบ้คำว่า "%s" โดยห้ามหลุดคำว่า "%s"
    [กฎ] ใบ้แบบอ้อมโลกที่สุด (เช่น ใบ้เรื่องความรู้สึกหรือสถานการณ์ที่เจอ) ห้ามบอกประเภทตรงๆ
    คำใบ้กวนๆ 1 ประโยคคือ:`, secretWord, secretWord)
	return AskGroqRaw(prompt)
}

func GenerateDescriptionGroq(word string) string {
	prompt := fmt.Sprintf(`Analyze the word: "%s" globally. Thai only output. 5 key points. Short.`, word)
	return AskGroqRaw(prompt)
}
