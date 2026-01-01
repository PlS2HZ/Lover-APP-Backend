package handlers

import (
	"couple-app/models"   // นำเข้า Models ของโปรเจกต์
	"couple-app/services" // นำเข้า Services (เช่น AI, Notification)
	"couple-app/utils"    // นำเข้า Utils (เช่น CORS)
	"encoding/json"       // จัดการ JSON
	"fmt"                 // จัดรูปแบบข้อความและ Log
	"math/rand"           // สุ่มตัวเลข
	"net/http"            // จัดการ HTTP Request/Response
	"os"                  // อ่าน Environment Variable
	"time"                // จัดการเวลา

	"github.com/supabase-community/postgrest-go" // ตัวช่วยสร้าง Query Supabase
	"github.com/supabase-community/supabase-go"  // Driver Supabase
)

// HandleSubmitQuizResponse - รับข้อมูลผลการเล่น Quiz และส่งแจ้งเตือนหาคู่รัก
func HandleSubmitQuizResponse(w http.ResponseWriter, r *http.Request) {
	// ✅ จัดการ CORS สำหรับทั้ง OPTIONS (Preflight) และ POST
	if utils.EnableCORS(&w, r) {
		return
	}

	fmt.Println("🚀 Submit API Called: Method =", r.Method) // Log เช็คใน Terminal Go ว่า API ถูกเรียก

	// โครงสร้างรับข้อมูลจาก Frontend
	var req struct {
		PartnerID  string `json:"partner_id"`  // ID ของแฟนที่จะให้แจ้งเตือน
		Question   string `json:"question"`    // คำถามที่ตอบถูก
		WrongCount int    `json:"wrong_count"` // จำนวนข้อที่ตอบผิด
	}

	// แปลง JSON จาก Body
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		fmt.Println("❌ Decode Error:", err)
		http.Error(w, "Bad Request", 400)
		return
	}

	// เรียกใช้ Service แจ้งเตือนความสำเร็จ (ส่ง Discord / Push Notification)
	services.NotifyQuizSuccess(req.PartnerID, req.Question, req.WrongCount)

	w.WriteHeader(http.StatusOK) // ตอบกลับ 200 OK
	fmt.Println("✅ Notification Sent Successfully")
}

// --- ฟังก์ชันอื่นๆ (คงเดิม) ---

// HandleSaveMemory - บันทึกความทรงจำใหม่ลงฐานข้อมูล
func HandleSaveMemory(w http.ResponseWriter, r *http.Request) {
	// จัดการ CORS
	if utils.EnableCORS(&w, r) {
		return
	}
	// โครงสร้างรับข้อมูลความทรงจำ
	var m struct {
		UserID     string   `json:"user_id"`
		Category   string   `json:"category"`
		Content    string   `json:"content"`
		HappenedAt string   `json:"happened_at"` // วันที่เกิดเหตุการณ์ (Optional)
		VisibleTo  []string `json:"visible_to"`  // ใครเห็นได้บ้าง
	}
	json.NewDecoder(r.Body).Decode(&m)

	// เชื่อมต่อ Supabase
	client, _ := supabase.NewClient(os.Getenv("SUPABASE_URL"), os.Getenv("SUPABASE_KEY"), nil)

	// เตรียมข้อมูล Insert
	insertData := map[string]interface{}{"user_id": m.UserID, "category": m.Category, "content": m.Content, "visible_to": m.VisibleTo}
	if m.HappenedAt != "" {
		insertData["happened_at"] = m.HappenedAt
	}

	// บันทึกลงตาราง memories
	client.From("memories").Insert(insertData, false, "", "", "").Execute()
	w.WriteHeader(201) // Created
}

// HandleGetAllMemories - ดึงความทรงจำทั้งหมดที่ User มีสิทธิ์เห็น
func HandleGetAllMemories(w http.ResponseWriter, r *http.Request) {
	// จัดการ CORS
	if utils.EnableCORS(&w, r) {
		return
	}
	userId := r.URL.Query().Get("user_id") // รับ User ID จาก Query Param

	client, _ := supabase.NewClient(os.Getenv("SUPABASE_URL"), os.Getenv("SUPABASE_KEY"), nil)
	var memories []models.Memory

	// Query: เลือกทุก field ที่ visible_to มี userId นี้อยู่ และเรียงตามวันที่สร้างล่าสุด
	client.From("memories").Select("*", "exact", false).Filter("visible_to", "cs", "{"+userId+"}").Order("created_at", &postgrest.OrderOpts{Ascending: false}).ExecuteTo(&memories)

	json.NewEncoder(w).Encode(memories)
}

// HandleDeleteMemory - ลบความทรงจำ
func HandleDeleteMemory(w http.ResponseWriter, r *http.Request) {
	// จัดการ CORS
	if utils.EnableCORS(&w, r) {
		return
	}
	id := r.URL.Query().Get("id") // รับ Memory ID

	client, _ := supabase.NewClient(os.Getenv("SUPABASE_URL"), os.Getenv("SUPABASE_KEY"), nil)
	// ลบข้อมูลจากตาราง memories
	client.From("memories").Delete("", "").Eq("id", id).Execute()

	w.WriteHeader(http.StatusOK)
}

// HandleGetRandomQuiz - สุ่มความทรงจำมา 1 เรื่อง แล้วให้ AI สร้างคำถาม
func HandleGetRandomQuiz(w http.ResponseWriter, r *http.Request) {
	// จัดการ CORS
	if utils.EnableCORS(&w, r) {
		return
	}

	userId := r.URL.Query().Get("user_id")
	client, err := supabase.NewClient(os.Getenv("SUPABASE_URL"), os.Getenv("SUPABASE_KEY"), nil)
	if err != nil {
		http.Error(w, "Database Connection Error", 500)
		return
	}

	var memories []map[string]interface{}
	// ดึงเฉพาะ Content ของความทรงจำที่ User นี้เห็นได้ (Limit 500 เพื่อประสิทธิภาพ)
	query := client.From("memories").Select("content", "exact", false)
	if userId != "" {
		query = query.Filter("visible_to", "cs", "{"+userId+"}")
	}

	_, err = query.Limit(500, "").ExecuteTo(&memories)
	if err != nil || len(memories) == 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound) // 404 ถ้าไม่เจอความทรงจำ
		json.NewEncoder(w).Encode(map[string]string{"error": "ไม่พบความทรงจำ"})
		return
	}

	// สุ่มแบบกระจายตัวสมบูรณ์ (Seed ด้วยเวลาปัจจุบัน)
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	content := memories[rng.Intn(len(memories))]["content"].(string)

	// ส่ง Content ไปให้ AI สร้างคำถาม (ผ่าน Service)
	quiz, err := services.GenerateQuizFromMemory(content)
	if err != nil {
		fmt.Printf("❌ AI Error: %v\n", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK) // ส่ง 200 เพื่อให้หน้าบ้านแสดง Error นุ่มนวล ไม่พัง
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	// ส่ง Quiz JSON กลับไปให้ Frontend
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(quiz)
}
