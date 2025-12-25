package handlers

import (
	"couple-app/services"
	"couple-app/utils"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/supabase-community/supabase-go"
)

// BOT_ID ของระบบ
const BOT_ID = "00000000-0000-0000-0000-000000000000"

func HandleBotAutoCreateGame(w http.ResponseWriter, r *http.Request) {
	if utils.EnableCORS(&w, r) {
		return
	}
	var body struct {
		GuesserID string `json:"guesser_id"`
	}
	json.NewDecoder(r.Body).Decode(&body)

	client, _ := supabase.NewClient(os.Getenv("SUPABASE_URL"), os.Getenv("SUPABASE_KEY"), nil)

	// ✅ 1. ดึงคำลับที่มีอยู่แล้วทั้งหมดมาป้องกันการซ้ำ
	var existingGames []map[string]interface{}
	client.From("heart_games").Select("secret_word", "", false).ExecuteTo(&existingGames)

	var usedWords []string
	for _, g := range existingGames {
		if word, ok := g["secret_word"].(string); ok {
			usedWords = append(usedWords, word)
		}
	}
	avoidList := strings.Join(usedWords, ", ")

	// ✅ 2. ปรับ Prompt: สุ่มหมวดหมู่ในระดับ Go เพื่อบีบ AI ให้ไม่ตอบ Pattern เดิม
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
	// ใช้เวลาปัจจุบันสุ่ม Index
	randomCat := categories[time.Now().UnixNano()%int64(len(categories))]

	prompt := fmt.Sprintf(`จงสุ่มคำนามภาษาไทยในหมวด "%s" มา 1 คำ 
	กฎ: ต้องไม่ใช่คำที่ง่ายเกินไป (เช่น รถยนต์, แมว, โทรศัพท์) และห้ามซ้ำกับ: %s. 
	ตอบแค่คำลับนั้นคำเดียวเท่านั้น!`, randomCat, avoidList)

	secretWord := services.AskGroqRaw(prompt) // ตัวนี้จะใช้ Temp 1.2 ที่เราแก้ไว้
	secretWord = strings.TrimSpace(strings.ReplaceAll(secretWord, ".", ""))
	descPrompt := fmt.Sprintf(`
    อธิบายลักษณะของ "%s" เป็นภาษาไทย 5 หัวข้อ:
    1.ประเภท... 2.ลักษณะ... 3.การใช้... 4.สถานที่... 5.จุดเด่น...
    [กฎ] ห้ามระบุจุดเด่นที่ทำให้รู้คำเฉลยทันที ห้ามใช้คำที่เฉพาะเจาะจงเกินไป และตอบสั้นๆ หัวข้อละ 1 ประโยค ห้ามยาว ห้ามเฉลยชื่อคำนี้เด็ดขาด`, secretWord)

	// ทำความสะอาดคำเผื่อ AI ดื้อใส่เครื่องหมายมา
	secretWord = strings.TrimSpace(secretWord)
	secretWord = strings.ReplaceAll(secretWord, ".", "")
	secretWord = strings.ReplaceAll(secretWord, "\"", "")
	secretWord = strings.ReplaceAll(secretWord, "'", "")

	if secretWord == "" {
		secretWord = "เครื่องบิน" // Fallback word
	}

	// ✅ 3. สั่งให้ AI สร้างฐานความรู้ (ใช้คำสั่ง GenerateDescription ที่นายพอใจ)
	description := services.AskGroqCustom(descPrompt, 800)

	// 4. บันทึกลงตาราง heart_games
	newGame := map[string]interface{}{
		"host_id":     BOT_ID,
		"secret_word": secretWord,
		"description": description,
		"use_bot":     true,
		"is_template": true,
		"status":      "waiting",
	}

	resp, _, err := client.From("heart_games").Insert(newGame, false, "", "", "").Execute()
	if err != nil {
		fmt.Printf("❌ DB Insert Error: %v\n", err)
		http.Error(w, "DB Error", 500)
		return
	}

	var gameResult []map[string]interface{}
	json.Unmarshal(resp, &gameResult)
	if len(gameResult) == 0 {
		http.Error(w, "No data returned", 500)
		return
	}
	gameID := gameResult[0]["id"].(string)

	// 5. สร้าง Session
	newSession := map[string]interface{}{
		"game_id": gameID, "guesser_id": body.GuesserID, "mode": "bot", "status": "playing",
	}
	respSess, _, _ := client.From("game_sessions").Insert(newSession, false, "", "", "").Execute()
	var sessionResult []map[string]interface{}
	json.Unmarshal(respSess, &sessionResult)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(sessionResult[0])
}
