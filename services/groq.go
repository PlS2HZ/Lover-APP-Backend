package services

import (
	"bytes"         // ใช้สำหรับจัดการ Buffer ข้อมูล (Payload)
	"encoding/json" // ใช้สำหรับแปลงข้อมูลเป็น JSON
	"fmt"           // ใช้สำหรับจัดรูปแบบข้อความ (Sprintf)
	"io"            // ใช้สำหรับอ่าน Body ของ Response
	"net/http"      // ใช้สำหรับยิง HTTP Request ไปหา Groq
	"os"            // ใช้สำหรับอ่าน API Key จาก Environment Variable
	"strings"       // ใช้สำหรับจัดการ String (ตัดช่องว่าง)
	"time"          // ใช้สำหรับกำหนด Timeout ของการยิง API
)

// AskGroqRaw: ฟังก์ชันเรียก AI แบบสุ่มสูง (Temperature 0.7)
// เหมาะสำหรับงานที่ต้องการความคิดสร้างสรรค์ เช่น การสุ่มคำศัพท์ตั้งต้นในโหมดบอท
func AskGroqRaw(prompt string) string {
	return AskGroqCustomWithTemp(prompt, 200, 0.7)
}

// AskGroqCustomWithTemp: ฟังก์ชันหลัก (Core) ในการยิง Request ไปหา Groq API
// รับค่า Prompt, MaxTokens (ความยาวสูงสุด), และ Temperature (ความสุ่ม)
func AskGroqCustomWithTemp(prompt string, maxTokens int, temp float64) string {
	apiKey := os.Getenv("GROQ_API_KEY")                      // อ่าน Key จาก Env
	url := "https://api.groq.com/openai/v1/chat/completions" // Endpoint ของ Groq

	// เตรียม Payload (ข้อมูลที่จะส่งไป)
	payload := map[string]interface{}{
		"model":       "llama-3.3-70b-versatile", // ใช้โมเดล Llama 3.3 ตัวใหม่ล่าสุด (เก่งและเร็ว)
		"messages":    []map[string]interface{}{{"role": "user", "content": prompt}},
		"temperature": temp,      // ค่าความสุ่ม (0.0 = เป๊ะมาก, 1.0 = มั่วได้ใจ)
		"max_tokens":  maxTokens, // จำกัดความยาวคำตอบ
		"top_p":       0.8,       // เทคนิคการสุ่มคำแบบ Top P
	}

	// แปลง Payload เป็น JSON
	jsonData, _ := json.Marshal(payload)

	// สร้าง HTTP Request
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey) // ใส่ Token ยืนยันตัวตน

	// สร้าง Client พร้อมกำหนด Timeout 15 วินาที (ถ้า AI ตอบช้าเกินให้ตัดจบ)
	client := &http.Client{Timeout: 15 * time.Second}
	resp, _ := client.Do(req)
	defer resp.Body.Close() // ปิด Connection เมื่อเสร็จสิ้น

	// อ่านข้อมูลที่ AI ตอบกลับมา
	body, _ := io.ReadAll(resp.Body)

	// สร้างโครงสร้างมารับ JSON Response
	var groqResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	// แกะซอง JSON
	json.Unmarshal(body, &groqResp)

	// ถ้าไม่มีคำตอบกลับมา ให้ส่งค่าว่าง
	if len(groqResp.Choices) == 0 {
		return ""
	}

	// ส่งคำตอบกลับไป (ตัดช่องว่างหน้าหลังออกด้วย TrimSpace)
	return strings.TrimSpace(groqResp.Choices[0].Message.Content)
}

// AskGroqCustom: ฟังก์ชันเรียก AI แบบแม่นยำสูง (Temperature 0.3)
// เหมาะสำหรับการตอบคำถาม Fact, ความรู้ทั่วไป, หรือการวิเคราะห์ข้อมูลที่ห้ามมั่ว
func AskGroqCustom(prompt string, maxTokens int) string {
	return AskGroqCustomWithTemp(prompt, maxTokens, 0.3)
}

// ✅ [Masterpiece] GenerateDescriptionGroq: ฟังก์ชันสร้างคำอธิบาย (Description) สำหรับคำลับ
// ใช้เทคนิค Prompt Engineering เพื่อบีบให้ AI ตอบเฉพาะเนื้อหาสำคัญ ไม่เอาน้ำ
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

	return AskGroqCustom(prompt, 1000) // ใช้ Token เยอะหน่อยเพื่อให้ AI อธิบายได้ครบถ้วน แต่ Temp ต่ำเพื่อความแม่น
}

// ✅ [Masterpiece] AskGroq: ฟังก์ชันหลักของ "Game Master (GM)"
// ทำหน้าที่ตอบคำถามของผู้เล่น โดยใช้ข้อมูลคำลับ (SecretWord) และคำอธิบาย (Description) เป็นฐานข้อมูล
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
    2. **การจบเกม (เข้มงวดสูงสุด)**: หากผู้เล่นทายถูกเป๊ะ หรือถามคำถามที่ระบุชื่อคำลับได้ถูกต้อง นายต้องตอบคำว่า "ถูกต้อง" เพียงคำเดียวเท่านั้น ห้ามมีคำอื่นปน ห้ามมีเว้นวรรค ห้ามมีจุด (.) ห้ามมีเครื่องหมายอัศเจรีย์ (!) หรือสัญลักษณ์ใดๆ ทั้งสิ้น เพื่อให้ระบบจบเกมได้ทันที
    3. **ห้ามสปอยล์**: ห้ามหลุดชื่อคำว่า "%s" ออกมาในระหว่างการตอบคำถามทั่วไปหากผู้เล่นยังทายไม่ถูก
    4. **ห้ามมโน (No Hallucination)**: ห้ามให้ข้อมูลที่ผิดจาก Fact ต้นฉบับเด็ดขาด
    5. **ห้ามกวนประสาท**: ตอบอย่างมหาปราชญ์ สุภาพ และชัดเจน

    คำถามจากผู้เล่น: "%s"`, secretWord, description, secretWord, question)

	// ใช้ Temp 0.3 เพื่อรักษาความแม่นยำของข้อมูล และ MaxTokens 500 เพื่อให้อธิบายได้พอดี ไม่ยาวเกินไป
	return AskGroqCustom(prompt, 500)
}

// AskGroqHint: ฟังก์ชันสร้างคำใบ้จาก Description
func AskGroqHint(description string) string {
	prompt := fmt.Sprintf(`สร้างคำใบ้ 1 ประโยคจากข้อมูลนี้: "%s" ห้ามเฉลยชื่อ ห้ามกวนประสาท`, description)
	return AskGroqCustom(prompt, 200)
}
