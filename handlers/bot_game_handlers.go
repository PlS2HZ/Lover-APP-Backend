package handlers

import (
	"couple-app/services" // นำเข้า Service สำหรับเรียก AI (Groq)
	"couple-app/utils"    // นำเข้า Utility เช่น CORS
	"encoding/json"       // จัดการ JSON
	"fmt"                 // จัดรูปแบบข้อความ (Sprintf)
	"net/http"            // จัดการ HTTP Request/Response
	"os"                  // อ่าน Environment Variables
	"strings"             // จัดการ String (ตัดคำ, แทนที่คำ)
	"time"                // จัดการเวลา (ใช้สุ่ม Random)

	"github.com/supabase-community/supabase-go" // Driver สำหรับ Supabase
)

// BOT_ID ของระบบ (ไอดีสมมติที่กำหนดไว้ตายตัวว่าเป็นของบอท)
const BOT_ID = "00000000-0000-0000-0000-000000000000"

// HandleBotAutoCreateGame ฟังก์ชันหลักสำหรับสร้างเกมใหม่โดยให้บอทเป็นคนตั้งโจทย์
func HandleBotAutoCreateGame(w http.ResponseWriter, r *http.Request) {
	// 1. จัดการ CORS เพื่อให้ Frontend (คนละโดเมน) สามารถเรียก API นี้ได้
	if utils.EnableCORS(&w, r) {
		return
	}

	// 2. รับข้อมูลจาก Body ว่าใครจะเป็นคนทาย (Guesser)
	var body struct {
		GuesserID string `json:"guesser_id"`
	}
	json.NewDecoder(r.Body).Decode(&body)

	// 3. สร้าง Client เชื่อมต่อ Supabase
	client, _ := supabase.NewClient(os.Getenv("SUPABASE_URL"), os.Getenv("SUPABASE_KEY"), nil)

	// ✅ 4. ดึงคำลับที่มีอยู่แล้วทั้งหมดมาป้องกันการซ้ำ (Logic เดิมของคุณ)
	var existingGames []map[string]interface{}
	// Query เลือกเฉพาะ column 'secret_word' จากตาราง heart_games
	client.From("heart_games").Select("secret_word", "", false).ExecuteTo(&existingGames)

	// วนลูปเก็บคำที่ใช้ไปแล้วเข้า Array
	var usedWords []string
	for _, g := range existingGames {
		if word, ok := g["secret_word"].(string); ok {
			usedWords = append(usedWords, word)
		}
	}
	// แปลง Array เป็น String ยาวๆ คั่นด้วย comma เพื่อส่งไปบอก AI ว่า "ห้ามเอาคำพวกนี้นะ"
	avoidList := strings.Join(usedWords, ", ")

	// ✅ 5. ปรับ Prompt: สุ่มหมวดหมู่ในระดับ Go เพื่อบีบ AI ให้ไม่ตอบ Pattern เดิม
	// รายการหมวดหมู่ทั้งหมด (Hardcode ไว้เพื่อให้ครอบคลุมหลายแนว)
	categories := []string{
		// --- หมวดคนและอาชีพ ---
		"อาชีพแปลกๆ", "อาชีพในฝัน", "ตัวละครในตำนาน", "บุคคลสำคัญ", "ศิลปิน/นักร้อง",
		"ซูเปอร์ฮีโร่", "ตัวการ์ตูน", "อาชีพในอดีต", "นักกีฬาชื่อดัง", "นักวิทยาศาสตร์",

		// --- หมวดสัตว์ ---
		"สัตว์ป่า", "สัตว์เลี้ยง", "สัตว์ทะเล", "แมลง", "สัตว์ดึกดำบรรพ์",
		"สัตว์ปีก", "สัตว์เลื้อยคลาน", "สัตว์ที่สูญพันธุ์ไปแล้ว", "สัตว์นำโชค", "สัตว์ในเทพนิยาย",

		// --- หมวดสิ่งของ/เครื่องใช้ ---
		"ของใช้ในบ้าน", "ของใช้ส่วนตัว", "อุปกรณ์ไอที", "เครื่องใช้ไฟฟ้า", "เฟอร์นิเจอร์",
		"ของเล่น", "เครื่องดนตรี", "อุปกรณ์กีฬา", "เครื่องครัว", "เครื่องเขียน",
		"เครื่องมือช่าง", "ของสะสม", "เครื่องประดับ", "อุปกรณ์เดินป่า", "ของใช้ในอดีต",

		// --- หมวดสถานที่ ---
		"สถานที่ท่องเที่ยวในไทย", "สถานที่สำคัญในโลก", "แลนด์มาร์ค", "สถานที่ในโรงเรียน", "ห้างสรรพสินค้า",
		"ร้านอาหาร", "สวนสาธารณะ", "โรงพยาบาล", "สถานีรถไฟ", "สนามบิน",
		"วัด/โบราณสถาน", "ยอดเขาชื่อดัง", "เกาะต่างๆ", "ถ้ำ", "น้ำตก",

		// --- หมวดพาหนะและเทคโนโลยี ---
		"ยานพาหนะทางบก", "ยานพาหนะทางน้ำ", "ยานพาหนะทางอากาศ", "จักรยานยอดฮิต", "รถแข่ง",

		// --- หมวดอาหารและธรรมชาติ ---
		"อาหารไทย", "ขนมหวาน", "ผลไม้ไทย", "เครื่องดื่ม", "สมุนไพร",
		"ดอกไม้", "ต้นไม้", "อวัยวะในร่างกาย", "เสื้อผ้า/เครื่องแต่งกาย", "งานอดิเรก",
		"ปรากฏการณ์ทางธรรมชาติ", "ดาวเคราะห์", "สภาพอากาศ", "เครื่องปรุงอาหาร", "ผักสวนครัว",
	}

	// ใช้เวลาปัจจุบัน (UnixNano) เพื่อสุ่ม Index ของหมวดหมู่
	randomCat := categories[time.Now().UnixNano()%int64(len(categories))]

	// สร้าง Prompt สำหรับขอ "คำลับ" จาก AI โดยระบุหมวดหมู่ที่สุ่มได้ และคำต้องห้าม
	prompt := fmt.Sprintf(`จงสุ่มคำนามภาษาไทยในหมวด "%s" มา 1 คำ 
    กฎ: ต้องไม่ใช่คำที่ง่ายเกินไป (เช่น รถยนต์, แมว, โทรศัพท์) และห้ามซ้ำกับ: %s. 
    ตอบแค่คำลับนั้นคำเดียวเท่านั้น!`, randomCat, avoidList)

	// 6. เรียกใช้ Service AI เพื่อขอคำลับ
	secretWord := services.AskGroqRaw(prompt) // ฟังก์ชันนี้ใช้ Temp 1.2 เพื่อความหลากหลาย
	// ทำความสะอาดผลลัพธ์เบื้องต้น (ตัดช่องว่าง, ตัดจุด)
	secretWord = strings.TrimSpace(strings.ReplaceAll(secretWord, ".", ""))

	// สร้าง Prompt สำหรับขอ "คำอธิบาย" 5 ข้อ
	descPrompt := fmt.Sprintf(`
    อธิบายลักษณะของ "%s" เป็นภาษาไทย 5 หัวข้อ:
    1.ประเภท... 2.ลักษณะ... 3.การใช้... 4.สถานที่... 5.จุดเด่น...
    [กฎ] ห้ามระบุจุดเด่นที่ทำให้รู้คำเฉลยทันที ห้ามใช้คำที่เฉพาะเจาะจงเกินไป และตอบสั้นๆ หัวข้อละ 1 ประโยค ห้ามยาว ห้ามเฉลยชื่อคำนี้เด็ดขาด`, secretWord)

	// 7. Sanitization: ทำความสะอาดคำลับอีกรอบเผื่อ AI ดื้อใส่เครื่องหมายมา
	secretWord = strings.TrimSpace(secretWord)
	secretWord = strings.ReplaceAll(secretWord, ".", "")
	secretWord = strings.ReplaceAll(secretWord, "\"", "") // ตัดฟันหนูคู่
	secretWord = strings.ReplaceAll(secretWord, "'", "")  // ตัดฟันหนูเดี่ยว

	// ถ้า AI เอ๋อ ไม่ตอบอะไรกลับมา ให้ใช้คำว่า "เครื่องบิน" กันโปรแกรมพัง
	if secretWord == "" {
		secretWord = "เครื่องบิน" // Fallback word
	}

	// ✅ 8. สั่งให้ AI สร้างคำอธิบาย (Description) ตาม Prompt ที่เตรียมไว้
	description := services.AskGroqCustom(descPrompt, 800)

	// 9. เตรียมข้อมูลสำหรับบันทึกลงตาราง heart_games
	newGame := map[string]interface{}{
		"host_id":     BOT_ID,      // คนสร้างคือบอท
		"secret_word": secretWord,  // คำลับที่ได้
		"description": description, // คำใบ้
		"use_bot":     true,        // เปิดโหมดบอท
		"is_template": true,        // เป็นเทมเพลต (อาจจะเอากลับมาเล่นซ้ำได้ในอนาคต)
		"status":      "waiting",   // สถานะเริ่มต้น
	}

	// 10. Insert ลง Database (Table: heart_games)
	resp, _, err := client.From("heart_games").Insert(newGame, false, "", "", "").Execute()
	if err != nil {
		fmt.Printf("❌ DB Insert Error: %v\n", err)
		http.Error(w, "DB Error", 500)
		return
	}

	// แปลงผลลัพธ์จาก DB เพื่อเอา Game ID
	var gameResult []map[string]interface{}
	json.Unmarshal(resp, &gameResult)
	if len(gameResult) == 0 {
		http.Error(w, "No data returned", 500)
		return
	}
	gameID := gameResult[0]["id"].(string)

	// 11. สร้าง Session การเล่น (Table: game_sessions)
	newSession := map[string]interface{}{
		"game_id":    gameID,         // ผูกกับเกมที่เพิ่งสร้าง
		"guesser_id": body.GuesserID, // ผูกกับคนทาย
		"mode":       "bot",          // โหมดบอท
		"status":     "playing",      // สถานะกำลังเล่น
	}
	// Insert Session
	respSess, _, _ := client.From("game_sessions").Insert(newSession, false, "", "", "").Execute()

	// แปลงผลลัพธ์ Session เพื่อส่งกลับให้ Frontend
	var sessionResult []map[string]interface{}
	json.Unmarshal(respSess, &sessionResult)

	// 12. ส่ง Response กลับไปหา Frontend (JSON)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated) // 201 Created
	json.NewEncoder(w).Encode(sessionResult[0])
}
