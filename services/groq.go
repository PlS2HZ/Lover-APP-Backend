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
    [System Role] นายคือ "มหาปราชญ์ผู้รวบรวมความรู้" หน้าที่ของนายคือวิเคราะห์อัตลักษณ์ของ "%s" 
    ครอบคลุมทุกมิติ (โลกจริง, อนิเมะ, เกม, วรรณกรรม, ของวิเศษ) เพื่อใช้เป็นฐานข้อมูลในการตอบคำถาม

    [หัวข้อการวิเคราะห์เชิงลึก]
    1. นิยามสากล: (เช่น ตัวละครมนุษย์, ของวิเศษศตวรรษที่ 22, คาถาป้องกันตัว, สถานที่ท่องเที่ยว)
    2. กายภาพ/ลักษณะเด่น: (สี, รูปร่าง, สิ่งที่มองเห็นแล้วจำได้ทันที)
    3. พลัง/คุณสมบัติ: (ทำอะไรได้บ้าง?, มีความสำคัญอย่างไร?, ใช้งานอย่างไร?)
    4. แหล่งกำเนิด/จักรวาล: (มาจากเรื่องอะไร?, พบได้ที่ไหน?, ใครเกี่ยวข้องบ้าง?)
    5. หมวดหมู่ (Category): ระบุประเภทที่ชัดเจนที่สุดเพียง 1 อย่าง (เช่น สัตว์, สถานที่, อาหาร, วิชา)

    [กฎ] ห้ามใช้คำว่า "%s" ในคำตอบเด็ดขาด และให้ข้อมูลที่แม่นยำที่สุดตาม Fact 100%%`, word, word)

	return AskGroqCustom(prompt, 1000)
}

// ✅ [แก้ไขใหม่] AskGroq: ปรับให้ตอบฉลาดแบบ Gemini (ชัดเจน, มีตรรกะ, ช่วย Scope)
func AskGroq(secretWord string, description string, question string) string {
	prompt := fmt.Sprintf(`
    [System Role] นายคือ "บอทผู้คุมกฎ" ที่มีความรู้ครอบจักรวาล หน้าที่ของนายคือตอบคำถาม Yes/No เพื่อให้ผู้เล่นทายคำลับให้ถูก
    
    [ข้อมูลคำลับ]
    - คำลับ: "%s"
    - ฐานข้อมูลสนับสนุน: %s

    [ตรรกะการตอบ]
    1. ใช้ความรู้รอบตัวของนายประเมินคำถามผู้ใช้อย่างจริงจังเทียบกับความจริงของ "%s"
    2. ตอบ "ใช่" หรือ "ไม่ใช่" ให้ชัดเจนที่ต้นประโยคเสมอ
    3. **สำคัญมาก**: ต้องให้เหตุผลประกอบสั้นๆ เพื่อช่วยผู้เล่นบีบวงคำตอบ (Scope) 
       - เช่น ถ้าทายผิดหมวด (ทายว่าเป็นคนแต่จริงคือที่เที่ยว) ให้ตอบว่า "ไม่ใช่ครับ สิ่งนี้ไม่ใช่สิ่งมีชีวิตแต่เป็นสถานที่"
       - เช่น ถ้าทายลักษณะถูก ให้ยืนยันและเสริม Fact เล็กน้อย
    4. ห้ามกวนประสาท ห้ามตอบคำถามด้วยคำถาม ห้ามอ้อมค้อม
    5. หากคำถามไม่ใช่ Yes/No ให้แจ้งว่า "โปรดถามคำถามที่ตอบได้เพียง ใช่ หรือ ไม่ใช่ เท่านั้น"
    6. หากทายถูกเป๊ะ หรือฉายาตรงกันอย่างชัดเจน ให้ตอบเพียงคำเดียวว่า "ถูกต้อง"
    7. ห้ามสปอยล์ชื่อ "%s" หรือคำใบ้ที่ง่ายเกินไปออกมาเด็ดขาด

    คำถามจากผู้เล่น: "%s"`, secretWord, description, secretWord, secretWord, question)

	return AskGroqCustom(prompt, 300)
}

// AskGroqHint คงเดิม
func AskGroqHint(description string) string {
	prompt := fmt.Sprintf(`
    [โจทย์] สร้างคำใบ้ 1 ประโยคจากข้อมูลนี้: "%s"
    [กฎ] ใบ้อ้อมๆ แต่ต้องสื่อถึงความจริง ห้ามเฉลยชื่อ ห้ามกวนจนเดาทางไม่ได้`, description)
	return AskGroqCustom(prompt, 200)
}
