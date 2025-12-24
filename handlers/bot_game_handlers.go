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

	// ✅ 1. สุ่มคำลับผ่าน Groq (ไวและฉลาด)
	secretWord := services.AskGroqRaw("สุ่มคำนามไทย 1 คำ (สัตว์/สิ่งของ/ตัวละคร/สถานที่) ตอบแค่คำนั้นคำเดียว")
	secretWord = strings.TrimSpace(strings.ReplaceAll(secretWord, ".", ""))
	if secretWord == "" {
		secretWord = "กาแฟ"
	}

	// ✅ 2. สั่งให้ AI สร้างฐานความรู้ (ลบ descPrompt ที่ไม่ได้ใช้ออกเพื่อแก้ Error)
	description := services.GenerateDescriptionGroq(secretWord)

	// 3. บันทึกลงตาราง heart_games
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

	// 4. สร้าง Session
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
