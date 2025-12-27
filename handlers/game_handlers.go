package handlers

import (
	"couple-app/models"
	"couple-app/services"
	"couple-app/utils"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/supabase-community/postgrest-go"
	"github.com/supabase-community/supabase-go"
)

// ✅ ห้ามลบ! คงเดิม
func isCloseEnough(s1, s2 string) bool {
	dist := utils.LevenshteinDistance(s1, s2)
	return dist <= 2 && dist > 0
}

// ... (HandleCreateHeartGame คงเดิมทั้งหมด ไม่ลบ)

func HandleCreateHeartGame(w http.ResponseWriter, r *http.Request) {
	if utils.EnableCORS(&w, r) {
		return
	}
	var g models.HeartGame
	json.NewDecoder(r.Body).Decode(&g)
	client, _ := supabase.NewClient(os.Getenv("SUPABASE_URL"), os.Getenv("SUPABASE_KEY"), nil)
	row := map[string]interface{}{
		"host_id":     g.HostID,
		"guesser_id":  g.GuesserID,
		"secret_word": g.SecretWord,
		"use_bot":     g.UseBot,
		"status":      "waiting",
	}
	var results []map[string]interface{}
	client.From("heart_games").Insert(row, false, "", "", "").ExecuteTo(&results)
	go func() {
		var userData []map[string]interface{}
		client.From("users").Select("username", "", false).Eq("id", g.HostID).ExecuteTo(&userData)
		username := "ใครบางคน"
		if len(userData) > 0 {
			username = userData[0]["username"].(string)
		}
		msg := "มีคำทายรออยู่ในใจเค้า... ❤️"
		if g.UseBot {
			msg = "เค้าส่งบอท Gemini มาท้าทายเธอ! 🤖"
		}
		services.TriggerPushNotification(g.GuesserID, "🎮 Mind Game", msg)
		services.SendMindGameNotification(username)
	}()
	json.NewEncoder(w).Encode(results[0])
}

// ✅ อัปเกรด: HandleAskQuestion ฉลาดและจริงใจ 100%
func HandleAskQuestion(w http.ResponseWriter, r *http.Request) {
	if utils.EnableCORS(&w, r) {
		return
	}
	var msg struct {
		GameID   string `json:"game_id"`
		SenderID string `json:"sender_id"`
		Message  string `json:"message"`
	}
	json.NewDecoder(r.Body).Decode(&msg)

	client, _ := supabase.NewClient(os.Getenv("SUPABASE_URL"), os.Getenv("SUPABASE_KEY"), nil)
	var sessionData []map[string]interface{}
	client.From("game_sessions").Select("game_id", "", false).Eq("id", msg.GameID).ExecuteTo(&sessionData)

	if len(sessionData) > 0 {
		heartGameID := sessionData[0]["game_id"].(string)
		var gameData []map[string]interface{}
		client.From("heart_games").Select("*", "", false).Eq("id", heartGameID).ExecuteTo(&gameData)

		if len(gameData) > 0 {
			secretWord := strings.TrimSpace(gameData[0]["secret_word"].(string))
			description := ""
			if gameData[0]["description"] != nil {
				description = gameData[0]["description"].(string)
			}

			cleanInput := strings.TrimSpace(msg.Message)
			lowInput := strings.ToLower(cleanInput)
			lowSecret := strings.ToLower(secretWord)
			botAnswer := ""

			// 🌟 1. [CRITICAL] เช็คคำตอบที่ถูกต้องก่อน (ห้ามผ่าน AI เด็ดขาด)
			if strings.Contains(lowInput, lowSecret) {
				botAnswer = fmt.Sprintf("ถูกต้อง! ใช่แล้ว... '%s' นั่นแหละ เก่งมาก!", secretWord)
				client.From("heart_games").Update(map[string]interface{}{"status": "finished"}, "", "").Eq("id", heartGameID).Execute()

				// บันทึกและ Return ทันทีเพื่อไม่ให้หลุดไปหา AI
				client.From("game_messages").Insert(map[string]interface{}{
					"game_id": heartGameID, "sender_id": msg.SenderID, "message": msg.Message, "answer": botAnswer,
				}, false, "", "", "").Execute()
				w.WriteHeader(http.StatusCreated)
				return
			}

			// 🌟 2. เช็คสะกดผิด (ห้ามผ่าน AI เช่นกัน)
			if isCloseEnough(lowInput, lowSecret) {
				botAnswer = fmt.Sprintf("นายหมายถึง '%s' หรือเปล่า? เกือบถูกแล้วสะกดอีกนิดเดียว!", secretWord)
			} else if strings.Contains(lowInput, "ใบ้") || strings.Contains(lowInput, "คำใบ้") {
				botAnswer = services.AskGroqHint(description)
			} else {
				// 🌟 3. ส่งให้ AI ตอบด้วย Prompt ที่นายต้องการ
				botAnswer = services.AskGroq(secretWord, description, msg.Message)
			}

			client.From("game_messages").Insert(map[string]interface{}{
				"game_id": heartGameID, "sender_id": msg.SenderID, "message": msg.Message, "answer": botAnswer,
			}, false, "", "", "").Execute()
			w.WriteHeader(http.StatusCreated)
			return
		}
	}
	w.WriteHeader(http.StatusCreated)
}

// ... (HandleGenerateAIDescription, HandleStartHeartGame, HandleGetLevels, HandleCreateGame คงเดิม)
func HandleGenerateAIDescription(w http.ResponseWriter, r *http.Request) {
	if utils.EnableCORS(&w, r) {
		return
	}
	var body struct {
		SecretWord string `json:"secret_word"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		return
	}
	description := services.GenerateDescriptionGroq(body.SecretWord)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"description": description})
}

func HandleStartHeartGame(w http.ResponseWriter, r *http.Request) {
	if utils.EnableCORS(&w, r) {
		return
	}
	gameID := r.URL.Query().Get("id")
	client, _ := supabase.NewClient(os.Getenv("SUPABASE_URL"), os.Getenv("SUPABASE_KEY"), nil)
	client.From("heart_games").Update(map[string]interface{}{
		"status":     "playing",
		"start_time": time.Now(),
	}, "", "").Eq("id", gameID).Execute()
	w.WriteHeader(http.StatusOK)
}

func HandleGetLevels(w http.ResponseWriter, r *http.Request) {
	if utils.EnableCORS(&w, r) {
		return
	}
	client, _ := supabase.NewClient(os.Getenv("SUPABASE_URL"), os.Getenv("SUPABASE_KEY"), nil)
	var levels []map[string]interface{}
	thirtyDaysAgo := time.Now().AddDate(0, 0, -30).Format("2006-01-02")
	client.From("heart_games").Select("*, users(username)", "", false).Gte("created_at", thirtyDaysAgo).Order("created_at", &postgrest.OrderOpts{Ascending: false}).ExecuteTo(&levels)
	json.NewEncoder(w).Encode(levels)
}

func HandleCreateGame(w http.ResponseWriter, r *http.Request) {
	if utils.EnableCORS(&w, r) {
		return
	}
	var body struct {
		GameID    string `json:"game_id"`
		GuesserID string `json:"guesser_id"`
	}
	json.NewDecoder(r.Body).Decode(&body)
	client, _ := supabase.NewClient(os.Getenv("SUPABASE_URL"), os.Getenv("SUPABASE_KEY"), nil)
	var session []map[string]interface{}
	client.From("game_sessions").Insert(map[string]interface{}{
		"game_id": body.GameID, "guesser_id": body.GuesserID, "mode": "bot", "status": "playing",
	}, false, "", "", "").ExecuteTo(&session)
	json.NewEncoder(w).Encode(session[0])
}
