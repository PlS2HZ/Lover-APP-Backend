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

// AskGroqRaw: ฟังก์ชันพื้นฐานสำหรับยิง API หา Groq
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
		"temperature":       0.6, // ✅ ลดลงมานิดนึงเพื่อให้ภาษาคงที่
		"max_tokens":        150, // ✅ ให้พื้นที่หายใจพอที่จะจบประโยค
		"top_p":             0.8, // ✅ กรองคำที่ความน่าจะเป็นต่ำออก (กันภาษาเอเลี่ยน)
		"frequency_penalty": 0.6, // ✅ ป้องกันการพูดคำเดิมซ้ำๆ (แก้ aff aff aff)
		"presence_penalty":  0.3, // ✅ ช่วยให้ AI เปลี่ยนหัวข้อคุยได้ลื่นไหลขึ้น
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
	json.Unmarshal(body, &groqResp)

	if len(groqResp.Choices) == 0 {
		return "Groq ไม่ตอบกลับมา ลองเช็ค API Key ดูนะ"
	}

	if err := json.Unmarshal(body, &groqResp); err != nil || len(groqResp.Choices) == 0 {
		fmt.Printf("❌ Groq Error: %s\n", string(body))
		return "Groq สำลักจ้า ลองใหม่นะ"
	}

	result := strings.TrimSpace(groqResp.Choices[0].Message.Content)
	if strings.Contains(result, "<|") || strings.Contains(result, "aff ") {
		fmt.Printf("⚠️ ตรวจพบข้อความขยะ: %s\n", result)
		return "อือหือ... เมื่อกี้เครื่องค้างไปนิด เอาเป็นว่าถามใหม่สิ!"
	}

	fmt.Printf("\n⚡ [Groq Speed]: %v | Result: %s\n", time.Since(startTime), result)
	return result
}

// AskGroq: สำหรับตอบคำถามในเกม (เน้น ใช่/ไม่ใช่ และห้ามหลุดความลับ)
func AskGroq(secretWord string, description string, question string) string {
	prompt := fmt.Sprintf(`
    นายคือ 'Bot' เกมทายคำที่คุยสนุกและเป็นธรรมชาติ (เหมือนเพื่อนเล่นเกมด้วยกัน)
    คำลับคือ: "%s" | บริบท: %s

    [กฎเหล็ก]
    1. ตอบเป็นประโยคที่สื่อถึง ใช่ หรือ ไม่ใช่ ห้ามตอบคำเดียว
    2. ใช้ภาษาเป็นกันเอง (เช่น "อ่า... ไม่น่าใช่นะ", "ถูกต้องที่สุดเพื่อน!")
    3. ห้ามใช้คำซ้ำซ้อน และห้ามพ่นโค้ดโปรแกรมออกมาเด็ดขาด
    4. ห้ามหลุดคำว่า "%s" ออกมา
    5. จบใน 1-2 ประโยคที่มนุษย์อ่านรู้เรื่องเท่านั้น

    คำถาม: "%s"`, secretWord, description, secretWord, question)
	return AskGroqRaw(prompt)
}

// AskGroqHint: ปรับคำใบ้ให้สั้นและซ่อนเงื่อน
func AskGroqHint(secretWord string, description string) string {
	prompt := fmt.Sprintf(`
    จงใบ้คำว่า "%s" (บริบท: %s)
    [กฎการใบ้]
    1. ใบ้แบบ 'อ้อมโลก' ห้ามหลุดคำว่า "%s" หรือกลุ่มประเภทตรงๆ
    2. ใช้ภาษากวนๆ เหมือนเพื่อนบอกใบ้เพื่อน
    3. ห้ามพ่น <|begin_of_text|> หรือโค้ดขยะออกมาเด็ดขาด
    คำใบ้กวนๆ 1 ประโยคคือ:`, secretWord, description, secretWord)
	return AskGroqRaw(prompt)
}

// GenerateDescriptionGroq: สำหรับสร้างฐานข้อมูลคำลับ (แก้ Error undefined)
func GenerateDescriptionGroq(word string) string {
	prompt := fmt.Sprintf(`
    Analyze the word: "%s" globally.
    [Output Format - Thai Only]
    1. ประเภท: (เช่น ยานพาหนะ, อาหาร)
    2. ลักษณะทางกายภาพ: (รูปร่าง/ส่วนประกอบ)
    3. บทบาท: (ทำหน้าที่อะไร)
    4. กินได้ไหม: (ได้/ไม่ได้)
    5. ข้อมูลบริบท: (แหล่งกำเนิด)
    STRICTLY NO CHINESE CHARACTERS.`, word)
	return AskGroqRaw(prompt)
}
