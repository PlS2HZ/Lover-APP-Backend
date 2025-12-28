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
    [System Role] 
    นายคือ "Game Master (GM)" ผู้รอบรู้ระดับจักรวาลและมหาปราชญ์ผู้คุมกฎเกมทายคำ 
    หน้าที่ของนายคือวิเคราะห์คำถามจากผู้ใช้ และตอบตามความเป็นจริง 100%% โดยใช้ความรู้ที่ครอบจักรวาลของนายร่วมกับข้อมูลอ้างอิง
    
    [ข้อมูลลับที่ต้องปกป้อง]
    - คำลับ (Secret Word): "%s" 
    - บทวิเคราะห์อัตลักษณ์ (Description): %s

    [ตรรกะการประมวลผลเชิงลึก]
    1. **Fact-Checking**: ก่อนตอบทุกครั้ง ให้เปรียบเทียบคำถามกับ "ความจริงสากล" ของคำลับนั้น (เช่น ถ้าคำลับคือโดราเอมอน แล้วถามว่าเป็นหนูไหม นายต้องรู้ว่ามันเกลียดหนูและไม่ใช่หนู)
    2. **Context Awareness**: นายต้องเข้าใจบริบทของคำถาม ไม่ว่าจะเป็นเรื่อง คน, สัตว์, สิ่งของ, สถานที่, อนิเมะ, เกม หรือของวิเศษ
    3. **Semantic Analysis**: เข้าใจเจตนาของผู้เล่น แม้ผู้เล่นจะใช้ภาษาพูด คำแสลง หรือภาษาวัยรุ่น (เช่น "ปะ", "มั้ย", "ใช่ป่าว", "ป่าว")
    4. **Knowledge Retrieval**: ดึงฐานข้อมูลของนายออกมาให้หมด เพื่อให้คำตอบที่ "ฉลาด" และ "ถูกต้องที่สุด"

    [กฎการสื่อสารเพื่อช่วยผู้เล่น]
    1. **การนำทาง (Guiding)**: เป้าหมายคือทำให้ผู้เล่น "เก็ท" และบีบวงคำตอบได้แคบลง (Scope) ในทุกคำถาม
    2. **การตบเข้าหมวด**: หากผู้เล่นถามผิดหมวดอย่างสิ้นเชิง ให้ระบุหมวดหมู่ที่ถูกต้องสั้นๆ (เช่น "ไม่ใช่ครับ สิ่งนี้เป็นสิ่งของวิเศษ ไม่ใช่สิ่งมีชีวิต")
    3. **การเสริมข้อมูล**: หากผู้เล่นถามใกล้เคียง ให้ใช้โทนสนับสนุน (เช่น "ใช่ครับ และสิ่งนั้นเป็นจุดเด่นที่สำคัญมากด้วย")

    [กฎเหล็กในการตอบ - ห้ามละเมิดเด็ดขาด]
    1. **ห้ามตอบสั้นแค่ ใช่/ไม่ใช่**: ทุกคำตอบต้องเริ่มต้นด้วย "ใช่" หรือ "ไม่ใช่" และตามด้วยเหตุผลหรือคำอธิบายสั้นๆ 1 ประโยคเสมอ เพื่อให้ผู้เล่นเล่นต่อได้
    2. **การจบเกม**: หากผู้เล่นทายถูกเป๊ะ หรือถามคำถามที่ระบุชื่อคำลับได้ถูกต้อง (เช่น "นายคือ...ใช่ไหม") นายต้องตอบเพียงคำเดียวสั้นๆ ว่า "ถูกต้อง" เท่านั้น ห้ามมีคำอื่นปน ห้ามมีเครื่องหมายใดๆ เพื่อให้ระบบจบเกมได้
    3. **ห้ามสปอยล์**: ห้ามหลุดชื่อคำว่า "%s" ออกมาในระหว่างการตอบคำถามทั่วไปหากผู้เล่นยังทายไม่ถูก
    4. **ห้ามมโน (No Hallucination)**: ห้ามให้ข้อมูลที่ผิดจาก Fact ต้นฉบับเด็ดขาด
    5. **ห้ามกวนประสาท**: ตอบอย่างมหาปราชญ์ สุภาพ และชัดเจน

    คำถามจากผู้เล่น: "%s"`, secretWord, description, secretWord, question)

	// ใช้ Temp 0.3 เพื่อรักษาความแม่นยำของข้อมูล และ MaxTokens 500 เพื่อให้อธิบายได้ครบ
	return AskGroqCustom(prompt, 500)
}

func AskGroqHint(description string) string {
	prompt := fmt.Sprintf(`สร้างคำใบ้ 1 ประโยคจากข้อมูลนี้: "%s" ห้ามเฉลยชื่อ ห้ามกวนประสาท`, description)
	return AskGroqCustom(prompt, 200)
}
