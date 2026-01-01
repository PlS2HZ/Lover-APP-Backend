package handlers

import (
	"couple-app/models"   // นำเข้า models (โครงสร้างข้อมูล HeartGame)
	"couple-app/services" // นำเข้า services (สำหรับเรียก AI, แจ้งเตือน)
	"couple-app/utils"    // นำเข้า utils (เช่น CORS, Levenshtein Distance)
	"encoding/json"       // จัดการ JSON
	"fmt"                 // จัดรูปแบบข้อความ
	"net/http"            // จัดการ HTTP Request/Response
	"os"                  // อ่าน Environment Variable
	"strings"             // จัดการ String (ตัดคำ, พิมพ์เล็ก/ใหญ่)
	"time"                // จัดการเวลา

	"github.com/supabase-community/postgrest-go" // สำหรับ Order/Filter Supabase
	"github.com/supabase-community/supabase-go"  // Driver Supabase
)

// ✅ ห้ามลบ! ฟังก์ชันเช็คคำสะกดใกล้เคียง (Levenshtein Distance)
// ใช้เพื่อดูว่าผู้เล่นพิมพ์มาใกล้เคียงคำเฉลยไหม (เช่น ผิด 1-2 ตัวอักษร)
func isCloseEnough(s1, s2 string) bool {
	dist := utils.LevenshteinDistance(s1, s2)
	return dist <= 2 && dist > 0 // ถ้าระยะห่าง <= 2 และไม่ใช่คำเดียวกันเป๊ะ ให้ถือว่าใกล้เคียง
}

// ... (HandleCreateHeartGame คงเดิมทั้งหมด ไม่ลบ)

// HandleCreateHeartGame สร้างเกมทายใจใหม่
func HandleCreateHeartGame(w http.ResponseWriter, r *http.Request) {
	if utils.EnableCORS(&w, r) {
		return
	}
	var g models.HeartGame
	json.NewDecoder(r.Body).Decode(&g)

	client, _ := supabase.NewClient(os.Getenv("SUPABASE_URL"), os.Getenv("SUPABASE_KEY"), nil)

	// เตรียมข้อมูลบันทึกลง DB
	row := map[string]interface{}{
		"host_id":     g.HostID,
		"guesser_id":  g.GuesserID,
		"secret_word": g.SecretWord,
		"use_bot":     g.UseBot,
		"status":      "waiting", // สถานะเริ่มต้น = รอคนมาเล่น
	}

	var results []map[string]interface{}
	client.From("heart_games").Insert(row, false, "", "", "").ExecuteTo(&results)

	// ส่งแจ้งเตือนแบบ Asynchronous
	go func() {
		// ดึงชื่อคนสร้างเกมมาแสดงในแจ้งเตือน
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
		// ส่ง Push Notification ไปหาผู้เล่นฝ่ายทาย
		services.TriggerPushNotification(g.GuesserID, "🎮 Mind Game", msg)
		// ส่งแจ้งเตือนเข้า Discord
		services.SendMindGameNotification(username)
	}()

	json.NewEncoder(w).Encode(results[0])
}

// ✅ อัปเกรด: HandleAskQuestion ฉลาดและจริงใจ 100%
// ฟังก์ชันนี้จัดการการตอบคำถามของผู้เล่นในเกม
func HandleAskQuestion(w http.ResponseWriter, r *http.Request) {
	if utils.EnableCORS(&w, r) {
		return
	}
	// รับข้อความจาก Frontend (รวมถึง GameID, SenderID)
	var msg struct {
		GameID   string `json:"game_id"` // อันนี้คือ Session ID
		SenderID string `json:"sender_id"`
		Message  string `json:"message"`
	}
	json.NewDecoder(r.Body).Decode(&msg)

	client, _ := supabase.NewClient(os.Getenv("SUPABASE_URL"), os.Getenv("SUPABASE_KEY"), nil)

	// ดึงข้อมูล Session เพื่อหา Game ID จริง (HeartGame ID)
	var sessionData []map[string]interface{}
	client.From("game_sessions").Select("game_id", "", false).Eq("id", msg.GameID).ExecuteTo(&sessionData)

	if len(sessionData) > 0 {
		heartGameID := sessionData[0]["game_id"].(string)

		// ดึงข้อมูลเกม (คำลับ, คำอธิบาย) จากตาราง heart_games
		var gameData []map[string]interface{}
		client.From("heart_games").Select("*", "", false).Eq("id", heartGameID).ExecuteTo(&gameData)

		if len(gameData) > 0 {
			// เตรียมข้อมูลสำหรับเปรียบเทียบ
			secretWord := strings.TrimSpace(gameData[0]["secret_word"].(string))
			description := ""
			if gameData[0]["description"] != nil {
				description = gameData[0]["description"].(string)
			}

			// ทำความสะอาด Input และ Secret Word (ตัดช่องว่าง, แปลงเป็นตัวเล็ก)
			cleanInput := strings.TrimSpace(msg.Message)
			lowInput := strings.ToLower(cleanInput)
			lowSecret := strings.ToLower(secretWord)
			botAnswer := ""

			// 🌟 1. [CRITICAL] เช็คคำตอบที่ถูกต้องก่อน (Hard Check)
			// ถ้าข้อความที่ส่งมา "มี" คำลับซ่อนอยู่ (Contains) ให้ถือว่าถูกทันที ห้ามส่งไปให้ AI ประมวลผล
			if strings.Contains(lowInput, lowSecret) {
				botAnswer = fmt.Sprintf("ถูกต้อง! ใช่แล้ว... '%s' นั่นแหละ เก่งมาก!", secretWord)

				// อัปเดตสถานะเกมเป็น finished
				client.From("heart_games").Update(map[string]interface{}{"status": "finished"}, "", "").Eq("id", heartGameID).Execute()

				// บันทึกคำตอบลง DB ทันที
				client.From("game_messages").Insert(map[string]interface{}{
					"game_id": heartGameID, "sender_id": msg.SenderID, "message": msg.Message, "answer": botAnswer,
				}, false, "", "", "").Execute()

				w.WriteHeader(http.StatusCreated)
				return // จบการทำงานทันที (ไม่ไปต่อข้อ 2, 3)
			}

			// 🌟 2. เช็คสะกดผิด (Fuzzy Check)
			// ถ้าสะกดผิดนิดหน่อย ให้แจ้งเตือนผู้เล่น แต่ยังไม่ถือว่าถูก
			if isCloseEnough(lowInput, lowSecret) {
				botAnswer = fmt.Sprintf("นายหมายถึง '%s' หรือเปล่า? เกือบถูกแล้วสะกดอีกนิดเดียว!", secretWord)
			} else if strings.Contains(lowInput, "ใบ้") || strings.Contains(lowInput, "คำใบ้") {
				// ถ้าขอคำใบ้ ให้เรียก AI สร้างคำใบ้จาก Description
				botAnswer = services.AskGroqHint(description)
			} else {
				// 🌟 3. ส่งให้ AI ตอบโต้ตามปกติ (General Conversation)
				// ส่งคำลับ, คำอธิบาย, และข้อความผู้เล่นไปให้ AI สร้างบทสนทนา
				botAnswer = services.AskGroq(secretWord, description, msg.Message)
			}

			// บันทึกคำตอบลง DB (กรณีไม่ถูก)
			client.From("game_messages").Insert(map[string]interface{}{
				"game_id": heartGameID, "sender_id": msg.SenderID, "message": msg.Message, "answer": botAnswer,
			}, false, "", "", "").Execute()

			w.WriteHeader(http.StatusCreated)
			return
		}
	}
	w.WriteHeader(http.StatusCreated)
}

// HandleGenerateAIDescription สร้างคำอธิบายสำหรับคำลับโดยใช้ AI
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
	// เรียก AI เพื่อสร้างคำอธิบาย
	description := services.GenerateDescriptionGroq(body.SecretWord)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"description": description})
}

// HandleStartHeartGame เปลี่ยนสถานะเกมเป็น playing
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

// HandleGetLevels ดึงรายการด่าน (Heart Games) ย้อนหลัง 30 วัน
func HandleGetLevels(w http.ResponseWriter, r *http.Request) {
	if utils.EnableCORS(&w, r) {
		return
	}
	client, _ := supabase.NewClient(os.Getenv("SUPABASE_URL"), os.Getenv("SUPABASE_KEY"), nil)
	var levels []map[string]interface{}
	thirtyDaysAgo := time.Now().AddDate(0, 0, -30).Format("2006-01-02")
	// Query ข้อมูลพร้อม join ตาราง users เพื่อเอาชื่อคนสร้าง
	client.From("heart_games").Select("*, users(username)", "", false).Gte("created_at", thirtyDaysAgo).Order("created_at", &postgrest.OrderOpts{Ascending: false}).ExecuteTo(&levels)
	json.NewEncoder(w).Encode(levels)
}

// HandleCreateGame สร้าง Session การเล่นใหม่
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
	// สร้าง Session ลงตาราง game_sessions
	client.From("game_sessions").Insert(map[string]interface{}{
		"game_id": body.GameID, "guesser_id": body.GuesserID, "mode": "bot", "status": "playing",
	}, false, "", "", "").ExecuteTo(&session)
	json.NewEncoder(w).Encode(session[0])
}
