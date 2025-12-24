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

	"github.com/supabase-community/postgrest-go"
	"github.com/supabase-community/supabase-go"
)

const APP_URL = "https://lover-frontend-ashen.vercel.app/"

// HandleCreateRequest สร้างคำขอใหม่และส่งแจ้งเตือน
func HandleCreateRequest(w http.ResponseWriter, r *http.Request) {
	if utils.EnableCORS(&w, r) {
		return
	}

	// โครงสร้างรับข้อมูลจาก Frontend
	var req struct {
		SenderID         string `json:"sender_id"`
		ReceiverUsername string `json:"receiver_username"`
		Header           string `json:"header"`
		Title            string `json:"title"`
		Description      string `json:"description"`
		StartTime        string `json:"time_start"`
		EndTime          string `json:"time_end"`
		Duration         string `json:"duration"`
		ImageURL         string `json:"image_url"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Bad Request", 400)
		return
	}

	client, _ := supabase.NewClient(os.Getenv("SUPABASE_URL"), os.Getenv("SUPABASE_KEY"), nil)

	// 1. หา ID คนรับ
	var targetUser []map[string]interface{}
	client.From("users").Select("id", "exact", false).Eq("username", req.ReceiverUsername).ExecuteTo(&targetUser)
	if len(targetUser) == 0 {
		http.Error(w, "Receiver Not Found", 404)
		return
	}
	rID := targetUser[0]["id"].(string)

	// 2. หาชื่อคนส่ง
	var senderUser []map[string]interface{}
	client.From("users").Select("username", "exact", false).Eq("id", req.SenderID).ExecuteTo(&senderUser)
	sName := "Unknown"
	if len(senderUser) > 0 {
		sName = senderUser[0]["username"].(string)
	}

	// 3. บันทึกลงตาราง requests (แมปตัวแปรให้ตรงกับ Schema)
	// title: เก็บรายละเอียดคำขอ, description: เก็บระยะเวลารวม
	dbRow := map[string]interface{}{
		"category":      req.Header,
		"title":         req.Title,
		"description":   req.Duration,
		"sender_id":     req.SenderID,
		"sender_name":   sName,
		"receiver_id":   rID,
		"receiver_name": req.ReceiverUsername,
		"status":        "pending",
		"image_url":     req.ImageURL,
		"remark":        fmt.Sprintf("เริ่ม: %s สิ้นสุด: %s", req.StartTime, req.EndTime),
	}

	_, _, err := client.From("requests").Insert(dbRow, false, "", "", "").Execute()
	if err != nil {
		fmt.Println("❌ DB Insert Error:", err)
		http.Error(w, "Internal Server Error", 500)
		return
	}

	// 4. แจ้งเตือน Discord & Push
	go func() {
		// ✅ เปลี่ยน T เป็นคำว่า " เวลา " เพื่อให้อ่านง่ายขึ้น
		formattedStart := strings.Replace(req.StartTime, "T", " เวลา ", 1)
		formattedEnd := strings.Replace(req.EndTime, "T", " เวลา ", 1)

		msg := fmt.Sprintf("👤 **จาก:** %s\n🏷️ **ประเภท:** %s\n📖 **รายละเอียดคำขอ:** %s\n⏰ **เริ่ม:** %s\n🏁 **สิ้นสุด:** %s\n⏳ **ระยะเวลารวม:** %s\n\n🔗 เข้าแอปที่นี่: %s",
			sName, req.Header, req.Title, formattedStart, formattedEnd, req.Duration, APP_URL)

		services.SendDiscordEmbed("💌 มีคำขอใหม่รอการอนุมัติ!", msg, 16738740, nil, req.ImageURL)
		services.TriggerPushNotification(rID, "💌 มีคำขอใหม่จาก "+sName, req.Title)
	}()

	w.WriteHeader(http.StatusCreated)
}

// HandleUpdateStatus อัปเดตสถานะ อนุมัติ/ปฏิเสธ
func HandleUpdateStatus(w http.ResponseWriter, r *http.Request) {
	if utils.EnableCORS(&w, r) {
		return
	}
	var body struct {
		ID      string `json:"id"`
		Status  string `json:"status"`
		Comment string `json:"comment"`
	}
	json.NewDecoder(r.Body).Decode(&body)
	client, _ := supabase.NewClient(os.Getenv("SUPABASE_URL"), os.Getenv("SUPABASE_KEY"), nil)

	var reqData []map[string]interface{}
	client.From("requests").Select("sender_id, title, receiver_name", "", false).Eq("id", body.ID).ExecuteTo(&reqData)

	client.From("requests").Update(map[string]interface{}{
		"status": body.Status, "comment": body.Comment, "processed_at": time.Now(),
	}, "", "").Eq("id", body.ID).Execute()

	if len(reqData) > 0 {
		senderID := reqData[0]["sender_id"].(string)
		title := reqData[0]["title"].(string)
		rName := reqData[0]["receiver_name"].(string)
		statusTxt := "✅ ได้รับอนุมัติแล้ว ✨"
		color := 5763719
		if body.Status == "rejected" {
			statusTxt = "❌ ถูกปฏิเสธ"
			color = 16729149
		}

		go func() {
			msg := fmt.Sprintf("📢 **คำขอ:** %s\n🎭 **สถานะ:** %s\n👤 **โดย:** %s\n💬 **เหตุผล:** %s\n\n🔗 ตรวจสอบ: %s",
				title, statusTxt, rName, body.Comment, APP_URL)
			services.SendDiscordEmbed("🔔 อัปเดตสถานะคำขอ", msg, color, nil, "")
			services.TriggerPushNotification(senderID, "📢 สถานะคำขอ: "+title, statusTxt)
		}()
	}
	w.WriteHeader(http.StatusOK)
}

// HandleGetMyRequests ดึงรายการทั้งหมดที่เกี่ยวข้องกับผู้ใช้
func HandleGetMyRequests(w http.ResponseWriter, r *http.Request) {
	if utils.EnableCORS(&w, r) {
		return
	}
	uID := r.URL.Query().Get("user_id")
	client, _ := supabase.NewClient(os.Getenv("SUPABASE_URL"), os.Getenv("SUPABASE_KEY"), nil)
	var data []map[string]interface{}

	query := fmt.Sprintf("sender_id.eq.%s,receiver_id.eq.%s", uID, uID)
	client.From("requests").Select("*", "exact", false).Or(query, "").Order("created_at", &postgrest.OrderOpts{Ascending: false}).ExecuteTo(&data)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}
