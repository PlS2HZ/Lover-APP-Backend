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

// ✅ GenerateDescriptionGroq คงเดิม
func GenerateDescriptionGroq(word string) string {
	prompt := fmt.Sprintf(`
    [บทบาท] นายคือ "สารานุกรมอัจฉริยะ" ที่รู้ทุกเรื่อง (ความจริง, อนิเมะ, เกม, ประวัติศาสตร์, สิ่งของลึกลับ)
    [โจทย์] วิเคราะห์อัตลักษณ์ของ "%s" ให้แม่นยำที่สุด 100%% 

    [มิติการวิเคราะห์]
    1. ประเภทและแหล่งที่มา: (บอกหมวดหมู่ให้ชัดเจน เช่น จังหวัดในไทย, ตัวละครอนิเมะ, ของวิเศษ, สัตว์ในเทพนิยาย)
    2. รูปลักษณ์: (ระบุสีตามจริง เช่น ลำไยต้องเปลือกน้ำตาลเนื้อขาว, โกโจผมขาวผ้าปิดตา, ภูเก็ตมีเกาะและหาด)
    3. บทบาท/พลัง/หน้าที่: (ระบุการใช้งานจริงหรือพลังในเรื่อง เช่น คอปเตอร์ไม้ไผ่ใช้บิน, ลำไยเป็นผลไม้/เครื่องดื่ม)
    4. พิกัดที่พบเจอ: (ระบุสถานที่จริงหรือจักรวาลต้นสังกัด เช่น อันดามัน, จูจุสึไคเซ็น, คาเฟ่, ห้างสรรพสินค้า)
    5. จุดเด่น/ความลับ: (ข้อมูลเชิงลึกที่ถูกต้องที่สุด เช่น ภูเก็ตมี 3 อำเภอ, ลำไยมีสรรพคุณทางยา)

    [กฎ] ตอบสั้นๆ เป็นข้อๆ ห้ามเอ่ยชื่อ "%s" ออกมาเด็ดขาด`, word, word)

	return AskGroqCustom(prompt, 800)
}

// ✅ อัปเกรด: AskGroq ตามกฎเหล็ก 14 ข้อของนาย
func AskGroq(secretWord string, description string, question string) string {
	prompt := fmt.Sprintf(`
    [บทบาท] นายคือ "ผู้ช่วยผู้รอบรู้" ในเกมทายใจ หน้าที่ของนายคือช่วยให้ผู้เล่นมาถูกทางด้วยข้อมูลที่ถูกต้อง
    คำลับคือ: "%s" | ข้อมูลอ้างอิง: %s

    [ตรรกะการตอบ - สำคัญมาก]
    1. วิเคราะห์คำถามและเปรียบเทียบกับความจริงเสมอ:
       - เช่น คำลับคือ "ลำไย" ถามว่า "เป็นสัตว์ใช่ไหม" -> ตอบ "ไม่ใช่เลย คนละสปีชีส์! สิ่งนี้เป็นพืชผลไม้นะ"
       - เช่น คำลับคือ "ภูเก็ต" ถามว่า "เป็นสถานที่ใช่ไหม" -> ตอบ "ใช่แล้ว และเป็นจังหวัดที่มีลักษณะเป็นเกาะด้วยนะ"
       - เช่น คำลับคือ "โกโจ" ถามว่า "เก่งไหม" -> ตอบ "แข็งแกร่งที่สุดในจักรวาลของเขาเลยละ ไม่มีใครเทียบได้"
    
    2. ห้ามตอบแค่ "ใช่/ไม่ใช่" ทื่อๆ: ต้องตอบแบบมีเหตุผลรองรับสั้นๆ และบอกใบ้อ้อมๆ พ่วงไปด้วยเสมอ
    3. กฎการตอบ 1-2 ประโยค: กระชับ ได้ใจความ และสุภาพ (ลดความกวนประสาทลง)
    4. การเพิ่มอารมณ์ร่วม: 
       - หากผู้เล่นเริ่มทายสโคปแคบเข้าหาความจริง ให้ตอบแบบ "ตื่นเต้น" หรือ "กดดัน" เพื่อเพิ่มความสนุก (เช่น "เริ่มใกล้เข้ามาแล้วนะ!", "โอ้โห มาถูกทางแบบน่าเหลือเชื่อเลย!")
    5. ห้ามสปอยล์: ห้ามใช้คำอธิบายที่ให้ไปมาตอบตรงๆ หรือหลุดชื่อคำลับออกมาเด็ดขาด
    6. วิเคราะห์คำถามอย่างจริงใจ: ถ้าถามว่า "เป็น...ใช่ไหม" ให้ตอบตามความจริง 100%% โดยอิงจากความรู้รอบตัวและฐานข้อมูล
    7. ห้ามกวนประสาท: ตัดนิสัยตอบปฏิเสธทุกอย่างทิ้งไป ถ้าผู้เล่นถามสิ่งที่เกี่ยวข้อง ให้ยืนยันและให้ข้อมูลเพิ่มสั้นๆ
    8. การันตีความถูกต้อง: ไม่ว่าจะเป็น จังหวัด, อนิเมะ, หรือผลไม้ นายต้องตอบข้อมูลที่ถูกต้องตามความจริงที่สุด
    9. หากผู้เล่นใกล้ความจริง: ให้ใช้โทนเสียงที่ตื่นเต้นและสนับสนุน
    10. หากผู้เล่นถามคำถามปลายเปิดที่ไม่สามารถตอบด้วย ใช่/ไม่ใช่ ได้: ให้ตอบว่า "โปรดถามคำถามที่ตอบได้เพียง ใช่ หรือ ไม่ใช่ เท่านั้น"
    11. หากผู้เล่นทายคำลับถูก (หรือมีความหมายตรงกันอย่างชัดเจนตามวิจารณญาณของนาย): ให้ตอบเพียงคำเดียวว่า "ถูกต้อง"
    12. ห้ามหลุดคำว่า "%s" หรือส่วนประกอบของคำนี้ออกมาในคำตอบเด็ดขาด
    13. ห้ามกวนด้วยการตอบคำถามด้วยคำถามเด็ดขาด
    14. ห้ามตอบคำถามที่ไม่เกี่ยวข้องกับคำลับหรือข้อมูลอ้างอิง

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
