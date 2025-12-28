package handlers

import (
	"couple-app/models"
	"couple-app/services"
	"couple-app/utils"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/supabase-community/postgrest-go"
	"github.com/supabase-community/supabase-go"
)

// HandleSubmitQuizResponse - ต้องใช้ utils.EnableCORS เป็นบรรทัดแรกเสมอ
func HandleSubmitQuizResponse(w http.ResponseWriter, r *http.Request) {
	// ✅ จัดการ CORS สำหรับทั้ง OPTIONS (Preflight) และ POST
	if utils.EnableCORS(&w, r) {
		return
	}

	fmt.Println("🚀 Submit API Called: Method =", r.Method) // Log เช็คใน Terminal Go

	var req struct {
		PartnerID  string `json:"partner_id"`
		Question   string `json:"question"`
		WrongCount int    `json:"wrong_count"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		fmt.Println("❌ Decode Error:", err)
		http.Error(w, "Bad Request", 400)
		return
	}

	// เรียกใช้ไฟล์แจ้งเตือนแยก
	services.NotifyQuizSuccess(req.PartnerID, req.Question, req.WrongCount)

	w.WriteHeader(http.StatusOK)
	fmt.Println("✅ Notification Sent Successfully")
}

// --- ฟังก์ชันอื่นๆ (คงเดิม) ---

func HandleSaveMemory(w http.ResponseWriter, r *http.Request) {
	if utils.EnableCORS(&w, r) {
		return
	}
	var m struct {
		UserID     string   `json:"user_id"`
		Category   string   `json:"category"`
		Content    string   `json:"content"`
		HappenedAt string   `json:"happened_at"`
		VisibleTo  []string `json:"visible_to"`
	}
	json.NewDecoder(r.Body).Decode(&m)
	client, _ := supabase.NewClient(os.Getenv("SUPABASE_URL"), os.Getenv("SUPABASE_KEY"), nil)
	insertData := map[string]interface{}{"user_id": m.UserID, "category": m.Category, "content": m.Content, "visible_to": m.VisibleTo}
	if m.HappenedAt != "" {
		insertData["happened_at"] = m.HappenedAt
	}
	client.From("memories").Insert(insertData, false, "", "", "").Execute()
	w.WriteHeader(201)
}

func HandleGetAllMemories(w http.ResponseWriter, r *http.Request) {
	if utils.EnableCORS(&w, r) {
		return
	}
	userId := r.URL.Query().Get("user_id")
	client, _ := supabase.NewClient(os.Getenv("SUPABASE_URL"), os.Getenv("SUPABASE_KEY"), nil)
	var memories []models.Memory
	client.From("memories").Select("*", "exact", false).Filter("visible_to", "cs", "{"+userId+"}").Order("created_at", &postgrest.OrderOpts{Ascending: false}).ExecuteTo(&memories)
	json.NewEncoder(w).Encode(memories)
}

func HandleDeleteMemory(w http.ResponseWriter, r *http.Request) {
	if utils.EnableCORS(&w, r) {
		return
	}
	id := r.URL.Query().Get("id")
	client, _ := supabase.NewClient(os.Getenv("SUPABASE_URL"), os.Getenv("SUPABASE_KEY"), nil)
	client.From("memories").Delete("", "").Eq("id", id).Execute()
	w.WriteHeader(http.StatusOK)
}

func HandleGetRandomQuiz(w http.ResponseWriter, r *http.Request) {
	if utils.EnableCORS(&w, r) {
		return
	}
	client, _ := supabase.NewClient(os.Getenv("SUPABASE_URL"), os.Getenv("SUPABASE_KEY"), nil)
	var memories []map[string]interface{}
	client.From("memories").Select("content", "exact", false).Limit(100, "").ExecuteTo(&memories)
	if len(memories) == 0 {
		http.Error(w, "No memories", 404)
		return
	}
	rand.Seed(time.Now().UnixNano())
	content := memories[rand.Intn(len(memories))]["content"].(string)
	quiz, _ := services.GenerateQuizFromMemory(content)
	json.NewEncoder(w).Encode(quiz)
}
