package handlers

import (
	"couple-app/services"
	"couple-app/utils"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

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

	// ✅ 2. สั่ง AI สุ่มคำใหม่ (เน้นย้ำให้ตอบแค่คำเดียว ห้ามมีอย่างอื่นเจือปน)
	prompt := fmt.Sprintf(`จงสุ่มคำนามไทย 1 คำ ห้ามซ้ำกับ: %s. ตอบแค่คำนั้นคำเดียว!`, avoidList)
	secretWord := services.AskGroqRaw(prompt)
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
