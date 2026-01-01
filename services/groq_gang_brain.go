package services

import (
	"bytes"         // ใช้สำหรับจัดการ Buffer ข้อมูล (Payload)
	"encoding/json" // ใช้สำหรับแปลงข้อมูล JSON
	"fmt"           // ใช้สำหรับจัดการ Error Formatting
	"io"            // ใช้สำหรับอ่าน Body ของ Response
	"net/http"      // ใช้สำหรับยิง HTTP Request
	"os"            // ใช้สำหรับอ่าน API Key จาก Env
	"strings"       // ใช้สำหรับจัดการ String (TrimSpace)
)

// QuizResponse: โครงสร้างข้อมูลที่คาดหวังจาก AI (ต้องตรงกับ JSON Format ที่สั่งไปใน Prompt)
type QuizResponse struct {
	Question     string   `json:"question"`      // คำถาม
	Options      []string `json:"options"`       // ตัวเลือก 4 ข้อ
	AnswerIndex  int      `json:"answer_index"`  // ดัชนีข้อที่ถูก (0-3)
	SweetComment string   `json:"sweet_comment"` // คอมเมนต์กวนๆ หรือน่ารักท้ายข้อ
}

// GenerateGangQuiz: ฟังก์ชันยิง API ไปหา Groq เพื่อสร้างโจทย์ตาม Prompt
func GenerateGangQuiz(prompt string) (*QuizResponse, error) {
	apiKey := os.Getenv("GROQ_API_KEY")                      // อ่าน Key จาก Environment
	url := "https://api.groq.com/openai/v1/chat/completions" // Endpoint ของ Groq

	// เตรียม Payload ที่จะส่งไปหา AI
	payload := map[string]interface{}{
		"model": "llama-3.3-70b-versatile", // ✅ อัปเกรดเป็นตัวที่ฉลาดที่สุดในตระกูล Llama 3 เพื่อป้องกันการมโน (Hallucination)
		"messages": []map[string]string{
			{
				"role": "system",
				// System Prompt: กำหนดบทบาทให้เป็น "มหาปราชญ์" และบังคับให้ตอบเป็น JSON เท่านั้น
				"content": "You are the 'Great Sage'. Return ONLY valid JSON. Your answers must be factually accurate and strictly follow the requested category.",
			},
			{
				"role":    "user",
				"content": prompt, // ใส่ Prompt ยาวๆ (Super Prompt) ที่จัดเตรียมมาจาก Handler
			},
		},
		"response_format": map[string]string{"type": "json_object"}, // บังคับ Output เป็น JSON Object ชัดเจน
		"temperature":     0.2,                                      // ✅ ตั้งค่าต่ำๆ (0.2) เพื่อลดความเพ้อเจ้อของ AI เน้นข้อมูลจริง (Fact) 100%
	}

	// แปลง Payload เป็น JSON Bytes
	jsonData, _ := json.Marshal(payload)

	// สร้าง HTTP Request
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	// สร้าง Client และส่ง Request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err // ส่ง Error กลับถ้าเน็ตหลุดหรือยิงไม่ไป
	}
	defer resp.Body.Close()

	// อ่าน Response Body
	body, _ := io.ReadAll(resp.Body)

	// โครงสร้างสำหรับแกะซอง Response ของ Groq (OpenAI Compatible Format)
	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	// แกะ JSON ชั้นแรก (Response Wrapper)
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	// เช็คว่า AI ตอบกลับมาไหม
	if len(result.Choices) == 0 {
		return nil, fmt.Errorf("AI response is empty")
	}

	// เอาเนื้อหาข้างใน (Content) มาทำความสะอาด (ตัดช่องว่างหน้าหลัง)
	aiContent := strings.TrimSpace(result.Choices[0].Message.Content)

	// แปลง JSON String ที่ AI ตอบมา ให้กลายเป็น Struct QuizResponse ของ Go
	var quiz QuizResponse
	if err := json.Unmarshal([]byte(aiContent), &quiz); err != nil {
		return nil, err // ถ้า AI ตอบ JSON ผิด Format จะ Error ตรงนี้
	}

	return &quiz, nil // ส่งคืน Pointer ของ QuizResponse
}
