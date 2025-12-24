package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
)

type GeminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
}

var (
	keyIndex int
	mu       sync.Mutex
)

func AskGeminiRaw(prompt string) string {
	keys := []string{
		os.Getenv("GEMINI_KEY_1"),
		os.Getenv("GEMINI_KEY_2"),
		os.Getenv("GEMINI_KEY_3"),
	}

	validKeys := []string{}
	for _, k := range keys {
		if k != "" {
			validKeys = append(validKeys, k)
		}
	}

	if len(validKeys) == 0 {
		return "เออ... ลืมตั้ง API Key ว่ะ ถามไม่ได้ (ไปเช็คไฟล์ .env ด่วน)"
	}

	for i := 0; i < len(validKeys); i++ {
		mu.Lock()
		apiKey := validKeys[keyIndex]
		currentIndex := keyIndex
		keyIndex = (keyIndex + 1) % len(validKeys)
		mu.Unlock()

		// ✅ เปลี่ยนชื่อรุ่นเป็น gemma-3-12b-it (ระบุ IT เพื่อให้รองรับ generateContent)
		// หรือหากยังไม่เจอ ให้ใช้รุ่นที่ชัวร์ที่สุดคือ gemini-1.5-flash-latest (โควต้า 1,500/วัน)
		url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/gemma-3-12b-it:generateContent?key=%s", apiKey)

		payload := map[string]interface{}{
			"contents": []interface{}{
				map[string]interface{}{
					"parts": []interface{}{
						map[string]interface{}{"text": prompt},
					},
				},
			},
		}

		jsonData, _ := json.Marshal(payload)
		resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
		if err != nil {
			fmt.Printf("⚠️ Network Error (Key %d): %v\n", currentIndex+1, err)
			continue
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)

		if resp.StatusCode == 429 {
			fmt.Printf("⚠️ API Key %d ติดลิมิต Quota, กำลังลองคีย์ถัดไป...\n", currentIndex+1)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			fmt.Printf("❌ API Error (Key %d) Status: %d, Body: %s\n", currentIndex+1, resp.StatusCode, string(body))
			continue
		}

		var geminiResp GeminiResponse
		if err := json.Unmarshal(body, &geminiResp); err != nil {
			continue
		}

		if len(geminiResp.Candidates) > 0 && len(geminiResp.Candidates[0].Content.Parts) > 0 {
			return strings.TrimSpace(geminiResp.Candidates[0].Content.Parts[0].Text)
		}
	}

	return "ทุกบัญชีติดลิมิต พักสัก 1-2 นาทีแล้วลองใหม่นะ"
}

func GenerateDescription(word string) string {
	prompt := fmt.Sprintf(`คุณคือผู้ช่วยสร้างคำอธิบายในเกมทายคำ หน้าที่ของคุณคืออธิบาย '%s'
    โดยใช้รูปแบบเป๊ะๆ ดังนี้:
    "คำในใจคือ '%s' เป็นสิ่งของ ([ระบุประเภท]) ไม่สามารถกินได้/กินได้ [ระบุลักษณะ 3 อย่าง] ไม่ใช่สถานที่"`, word, word)
	return AskGeminiRaw(prompt)
}

func AskGemini(secretWord string, description string, question string) string {
	// ✅ รักษา Logic กวนประสาทแบบประโยคเดียวจบ
	prompt := fmt.Sprintf(`
    นายคือ 'Rubssarb Bot' เพื่อนสนิทจอมกวนที่กำลังเล่นเกมทายคำกับเพื่อน
    
    [โจทย์ลับ]
    - คำที่ต้องทาย: "%s"
    - ข้อมูล: %s

    [กฎเหล็กในการตอบ]
    1. ตอบจบใน "1 ประโยค" เท่านั้น
    2. กวนได้แต่ประโยคสุดท้าย "ต้องเป็นความจริง" (ใช่/ไม่ใช่/เกือบถูกแล้ว)
    3. ห้ามหลุดคำว่า "%s" ออกมาเด็ดขาด
    4. ถ้าเพื่อนทายถูกเป๊ะๆ ว่า "%s" ให้ตอบคำเดียวว่า "ถูกต้อง"

    คำถามเพื่อน: "%s"`, secretWord, description, secretWord, secretWord, question)

	return AskGeminiRaw(prompt)
}
