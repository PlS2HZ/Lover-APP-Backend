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

// ✅ ปรับความสุ่มให้เหมาะสม (0.7 สำหรับการสุ่มคำ)
func AskGroqRaw(prompt string) string {
	return AskGroqCustomWithTemp(prompt, 200, 0.7)
}

func AskGroqCustomWithTemp(prompt string, maxTokens int, temp float64) string {
	apiKey := os.Getenv("GROQ_API_KEY")
	url := "https://api.groq.com/openai/v1/chat/completions"
	payload := map[string]interface{}{
		"model":       "llama-3.3-70b-versatile",
		"messages":    []map[string]interface{}{{"role": "user", "content": prompt}},
		"temperature": temp,
		"max_tokens":  maxTokens,
		"top_p":       0.8,
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

// ✅ ปรับความแม่นยำสูง (0.3 สำหรับการตอบคำถาม)
func AskGroqCustom(prompt string, maxTokens int) string {
	return AskGroqCustomWithTemp(prompt, maxTokens, 0.3)
}

// ✅ [Masterpiece] GenerateDescriptionGroq: เน้นเนื้อๆ ป้องกัน Context Overflow
func GenerateDescriptionGroq(word string) string {
	prompt := fmt.Sprintf(`
    [SYSTEM ROLE] นายคือ "มหาปราชญ์ผู้สรุปสาระสำคัญ" 
    [GOAL] วิเคราะห์ "%s" ให้ได้ข้อมูล "เนื้อๆ" ที่ถูกต้อง 100%% เพื่อใช้ในเกมทายคำ

    [ข้อมูลที่ต้องระบุ - เน้นความสำคัญตามลำดับ]
    1. **ประเภทที่แท้จริง**: (ห้ามผิดเด็ดขาด เช่น หุ่นยนต์, ตัวละคร, พืช, สถานที่)
    2. **อัตลักษณ์ที่เห็นแล้วจำได้ทันที**: (รูปร่าง สี ผิวพรรณ อุปกรณ์เสริม)
    3. **ความสามารถ/จุดเด่นที่สุด**: (สิ่งที่ทำให้สิ่งนี้ต่างจากอย่างอื่นในประเภทเดียวกัน)
    4. **จักรวาล/ถิ่นที่อยู่**: (มาจากเรื่องไหน หรือพบได้ที่ไหน)
    5. **จุดอ่อน/สิ่งที่เกลียด**: (ข้อมูลสำคัญที่ช่วยแยกแยะ Fact)

    [RULES] 
    - ห้ามใช้คำว่า "%s" เด็ดขาด
    - **ห้ามเขียนน้ำเยอะ**: ให้เน้นข้อมูลที่เป็นเอกลักษณ์เฉพาะตัว (Unique Identifiers)
    - ข้อมูลต้องตรงตาม Fact 100%% หากมโนข้อมูลผิดจะถูกลงโทษ`, word, word)

	return AskGroqCustom(prompt, 1000) // ปรับลดลงตามข้อแนะนำ เพื่อป้องกัน AI หลุดโฟกัส
}

// ✅ [Masterpiece] AskGroq: ฉลาด ยืดหยุ่น และมีระบบ Anti-Cheese
func AskGroq(secretWord string, description string, question string) string {
	prompt := fmt.Sprintf(`
    [บทบาท] นายคือ "Game Master (GM)" ที่มีความรอบรู้และยืดหยุ่นสูงเหมือน Gemini
    
    [ข้อมูลลับ]
    คำลับ: "%s" | ข้อมูลอ้างอิง: %s

    [ตรรกะการประมวลผล]
    1. **Flexibility**: ทำความเข้าใจภาษาพูด คำแสลง (เช่น "ปะ", "มั้ย", "ตัวมันเขียวปะ") อย่างยืดหยุ่น
    2. **Anti-Cheese**: หากผู้เล่นถามไล่ประเภทแบบไม่มีชั้นเชิง (เช่น "เป็นสัตว์ไหม" "เป็นคนไหม" ติดๆ กัน) ให้ตอบปฏิเสธพร้อมคำใบ้ที่ "ท้าทาย" ขึ้น ไม่คายคำตอบง่ายเกินไป
    3. **Fact-First**: ตรวจสอบความถูกต้องกับฐานความรู้จริงของนายเสมอ

    [กฎการสื่อสาร]
    1. เริ่มด้วย "ใช่" หรือ "ไม่ใช่" ให้ชัดเจน
    2. **การ Scope**: ช่วยผู้เล่นเมื่อเขาถามด้วยตรรกะที่ดี แต่เริ่ม "กั๊ก" ข้อมูลเมื่อเขาถามแบบ Cheese
    3. **โทน**: เป็นกันเองแต่ทรงความรู้ ไม่กวนประสาทแต่มีความน่าเกรงขามแบบ GM
    4. หากทายถูกเป๊ะหรือใกล้เคียงมาก: ตอบคำเดียวว่า "ถูกต้อง"
    5. ห้ามสปอยล์ชื่อ "%s" เด็ดขาด

    คำถามจากผู้เล่น: "%s"`, secretWord, description, secretWord, question)

	return AskGroqCustom(prompt, 500)
}

func AskGroqHint(description string) string {
	prompt := fmt.Sprintf(`สร้างคำใบ้ 1 ประโยคจากข้อมูลนี้: "%s" ห้ามเฉลยชื่อ ห้ามกวนประสาท`, description)
	return AskGroqCustom(prompt, 200)
}
