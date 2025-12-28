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

// ... (AskGroqRaw, AskGroqCustomWithTemp, AskGroqCustom คงเดิม) ...
func AskGroqRaw(prompt string) string {
	return AskGroqCustomWithTemp(prompt, 100, 1.2)
}

func AskGroqCustomWithTemp(prompt string, maxTokens int, temp float64) string {
	apiKey := os.Getenv("GROQ_API_KEY")
	url := "https://api.groq.com/openai/v1/chat/completions"
	payload := map[string]interface{}{
		"model":       "llama-3.3-70b-versatile",
		"messages":    []map[string]interface{}{{"role": "user", "content": prompt}},
		"temperature": temp,
		"max_tokens":  maxTokens,
		"top_p":       0.9,
	}
	jsonData, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)
	client := &http.Client{Timeout: 15 * time.Second}
	resp, _ := client.Do(req)
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
		return ""
	}
	return strings.TrimSpace(groqResp.Choices[0].Message.Content)
}

func AskGroqCustom(prompt string, maxTokens int) string {
	return AskGroqCustomWithTemp(prompt, maxTokens, 0.5)
}

func GenerateDescriptionGroq(word string) string {
	prompt := fmt.Sprintf(`
    [System Role] นายคือ "มหาปราชญ์ผู้รวบรวมความรู้" 
    [โจทย์] วิเคราะห์ "%s" โดยดึง "อัตลักษณ์ที่เด่นชัดที่สุด" ที่คนเห็นแล้วต้องนึกถึงทันที

    [หัวข้อการวิเคราะห์]
    1. นิยามและประเภท: (เช่น พืชผักที่มีรสชาติเฉพาะตัว, ตัวละครที่มีนิสัย...)
    2. รูปลักษณ์ที่เป็นเอกลักษณ์: (ต้องระบุลักษณะผิว, สี, หรือรูปร่างที่ "แตกต่าง" จากสิ่งอื่นในหมวดเดียวกัน)
    3. คุณสมบัติเด่น/รสชาติ/พลัง: (ระบุสิ่งที่ทำให้คนจดจำได้ทันที เช่น ความขม, ความเร็ว, พลังไฟ)
    4. บริบทความสัมพันธ์: (แหล่งกำเนิด หรือเมนูอาหารที่คนนึกถึงบ่อยที่สุด)
    5. ความลับ/Trivia: (ข้อมูลสั้นๆ ที่เป็น Fact จริงของสิ่งนี้)

    [กฎ] ห้ามใช้คำว่า "%s" ในคำตอบ และ "ห้ามตอบกว้างเกินไป" จนใช้กับสิ่งอื่นได้`, word, word)

	return AskGroqCustom(prompt, 1000)
}

// ✅ [ฉบับปรับปรุง] AskGroq: ตอบฉลาด มีเหตุผล และช่วย Scope ให้แฟนเล่นสนุกขึ้น
func AskGroq(secretWord string, description string, question string) string {
	prompt := fmt.Sprintf(`
    [System Role] นายคือ "ผู้ช่วยมหาปราชญ์" ในเกมทายคำ หน้าที่ของนายคือตอบคำถาม Yes/No อย่างฉลาดที่สุด
    เป้าหมาย: ตอบเพื่อให้ผู้เล่น "บีบวงคำตอบได้แคบลง" ในทุกๆ คำถาม แต่ห้ามเฉลยชื่อตรงๆ

    [ฐานข้อมูลคำลับ]
    - คำลับ: "%s"
    - ข้อมูลลักษณะเฉพาะ: %s

    [ตรรกะการตอบ - ต้องทำตามนี้]
    1. ตอบ "ใช่" หรือ "ไม่ใช่" ให้ชัดเจนที่ต้นประโยคเสมอ
    2. **ช่วยผู้เล่น Scope (สำคัญมาก)**:
       - ถ้าผู้เล่นทาย "ประเภท" ผิด: ให้บอกว่าไม่ใช่ และระบุประเภทที่ถูกสั้นๆ (เช่น "ไม่ใช่สัตว์ครับ สิ่งนี้เป็นพืชผัก")
       - ถ้าผู้เล่นทาย "ลักษณะ" ถูก: ให้ยืนยันและเสริมความมั่นใจ (เช่น "ใช่ครับ และผิวของมันยังไม่เรียบด้วย")
       - ถ้าผู้เล่นทาย "ใกล้เคียง": ให้บอกว่า "ใกล้เคียงมากครับ" หรือ "มาถูกทางแล้ว"
    3. **ใช้ความรู้รอบตัว**: หากผู้เล่นถามสิ่งที่ไม่มีใน description (เช่น ถามเรื่องรสชาติหรือราคา) ให้ใช้ความรู้ของนายตอบตามความจริงได้เลย
    4. **โทนการพูด**: สุภาพ ชัดเจน และสนับสนุนผู้เล่น (ตัดความกวนประสาททิ้ง 100%%)
    5. หากทายถูก (หรือความหมายตรงกันเป๊ะ): ตอบคำเดียวว่า "ถูกต้อง"
    6. ห้ามสปอยล์ชื่อ "%s" ออกมาเด็ดขาด

    คำถามจากผู้เล่น: "%s"`, secretWord, description, secretWord, question)

	return AskGroqCustom(prompt, 300)
}

// AskGroqHint คงเดิม
func AskGroqHint(description string) string {
	prompt := fmt.Sprintf(`
    [โจทย์] สร้างคำใบ้ 1 ประโยคจากข้อมูลนี้: "%s"
    [กฎ] ใบ้อ้อมๆ แต่ต้องสื่อถึงความจริง ห้ามเฉลยชื่อ ห้ามกวนจนเดาทางไม่ได้`, description)
	return AskGroqCustom(prompt, 200)
}
