package handlers

import (
	"couple-app/services" // นำเข้า Service (Discord, Push Notification)
	"couple-app/utils"    // นำเข้า Utils (CORS)
	"encoding/json"       // จัดการ JSON
	"fmt"                 // จัดรูปแบบข้อความ
	"net/http"            // จัดการ HTTP Request/Response
	"os"                  // อ่าน Env Vars
	_ "strings"           // (ไม่ได้ใช้ แต่ import เผื่อไว้)
	"time"                // จัดการเวลา

	"github.com/supabase-community/postgrest-go" // ตัวช่วยสร้าง Query
	"github.com/supabase-community/supabase-go"  // Driver Supabase
)

// APP_URL: URL ของหน้าเว็บ Frontend (ใช้สำหรับสร้างลิ้งค์ใน Discord)
const APP_URL = "https://lover-frontend-ashen.vercel.app/"

// HandleCreateRequest: สร้างคำขอใหม่และส่งแจ้งเตือน (Create)
func HandleCreateRequest(w http.ResponseWriter, r *http.Request) {
	// 1. จัดการ CORS
	if utils.EnableCORS(&w, r) {
		return
	}

	// 2. รับข้อมูลจาก Frontend
	var req struct {
		SenderID         string `json:"sender_id"`         // ไอดีคนส่ง
		ReceiverUsername string `json:"receiver_username"` // ชื่อคนรับ (เช่น แฟน)
		Header           string `json:"header"`            // หมวดหมู่ (เช่น เที่ยว, กินข้าว)
		Title            string `json:"title"`             // รายละเอียด
		Description      string `json:"description"`       // คำอธิบายเพิ่มเติม
		StartTime        string `json:"time_start"`        // เวลาเริ่ม
		EndTime          string `json:"time_end"`          // เวลาจบ
		Duration         string `json:"duration"`          // ระยะเวลารวม
		ImageURL         string `json:"image_url"`         // รูปภาพประกอบ (ถ้ามี)
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Bad Request", 400)
		return
	}

	// 3. เชื่อมต่อ Supabase
	client, _ := supabase.NewClient(os.Getenv("SUPABASE_URL"), os.Getenv("SUPABASE_KEY"), nil)

	// 4. ค้นหา ID ของผู้รับจาก Username
	var targetUser []map[string]interface{}
	client.From("users").Select("id", "exact", false).Eq("username", req.ReceiverUsername).ExecuteTo(&targetUser)
	if len(targetUser) == 0 {
		http.Error(w, "Receiver Not Found", 404) // ถ้าไม่เจอชื่อนี้ในระบบ
		return
	}
	rID := targetUser[0]["id"].(string)

	// 5. ค้นหาชื่อของผู้ส่งจาก ID (เพื่อเอาไปแสดงผล)
	var senderUser []map[string]interface{}
	client.From("users").Select("username", "exact", false).Eq("id", req.SenderID).ExecuteTo(&senderUser)
	sName := "Unknown"
	if len(senderUser) > 0 {
		sName = senderUser[0]["username"].(string)
	}

	// 6. เตรียมข้อมูลลง Database
	dbRow := map[string]interface{}{
		"category":      req.Header,
		"title":         req.Title,
		"description":   req.Duration, // บันทึก Duration ลงในช่อง description ของ DB
		"sender_id":     req.SenderID,
		"sender_name":   sName,
		"receiver_id":   rID,
		"receiver_name": req.ReceiverUsername,
		"status":        "pending", // สถานะเริ่มต้น = รออนุมัติ
		"image_url":     req.ImageURL,
		"remark":        fmt.Sprintf("เริ่ม: %s สิ้นสุด: %s", req.StartTime, req.EndTime), // หมายเหตุเรื่องเวลา
	}

	// Insert ลงตาราง requests
	_, _, err := client.From("requests").Insert(dbRow, false, "", "", "").Execute()
	if err != nil {
		fmt.Println("❌ DB Insert Error:", err)
		http.Error(w, "Internal Server Error", 500)
		return
	}

	// 7. ทำงานเบื้องหลัง (Go Routine) เพื่อส่งแจ้งเตือนโดยไม่บล็อกผู้ใช้
	go func() {
		fmt.Println("🚀 Starting Discord Notification GoRoutine...")

		// ฟังก์ชันช่วยแปลงเวลาให้สวยงาม
		parseTime := func(iso string) string {
			t, err := time.Parse(time.RFC3339, iso)
			if err != nil {
				t, _ = time.Parse("2006-01-02T15:04", iso)
			}
			return t.Format("02/01/2006 เวลา 15:04")
		}

		formattedStart := parseTime(req.StartTime)
		formattedEnd := parseTime(req.EndTime)

		// สร้างข้อความ Discord
		msg := fmt.Sprintf("👤 **จาก:** %s\n🎯 **ถึงคุณ:** %s\n🏷️ **ประเภท:** %s\n📖 **รายละเอียด:** %s\n⏰ **เริ่ม:** %s\n🏁 **สิ้นสุด:** %s\n⏳ **ระยะเวลา:** %s\n\n🔗 เข้าแอปที่นี่: %s",
			sName, req.ReceiverUsername, req.Header, req.Title, formattedStart, formattedEnd, req.Duration, APP_URL)

		// ส่ง Discord Embed (สีส้ม)
		services.SendDiscordEmbed("💌 มีคำขอใหม่รอการอนุมัติ!", msg, 16738740, nil, req.ImageURL)
		fmt.Println("✅ Discord Embed sent command triggered")

		// ส่ง Push Notification ไปหาผู้รับ (rID)
		services.TriggerPushNotification(rID, "💌 มีคำขอใหม่จาก "+sName, req.Title)
		fmt.Println("✅ Push Notification triggered")
	}()

	w.WriteHeader(http.StatusCreated)
}

// HandleUpdateStatus: อัปเดตสถานะ (อนุมัติ/ปฏิเสธ) และส่งแจ้งเตือนกลับหาคนขอ
func HandleUpdateStatus(w http.ResponseWriter, r *http.Request) {
	// 1. จัดการ CORS
	if utils.EnableCORS(&w, r) {
		return
	}
	var body struct {
		ID      string `json:"id"`      // ID ของ Request
		Status  string `json:"status"`  // สถานะใหม่ (approved, rejected)
		Comment string `json:"comment"` // เหตุผลประกอบ
	}
	json.NewDecoder(r.Body).Decode(&body)

	client, _ := supabase.NewClient(os.Getenv("SUPABASE_URL"), os.Getenv("SUPABASE_KEY"), nil)

	// 2. ดึงข้อมูล Request เดิมมาก่อน (เพื่อเอา ID คนส่ง จะได้แจ้งเตือนกลับถูก)
	var reqData []map[string]interface{}
	client.From("requests").Select("sender_id, title, receiver_name", "", false).Eq("id", body.ID).ExecuteTo(&reqData)

	// 3. อัปเดตสถานะลง Database
	client.From("requests").Update(map[string]interface{}{
		"status": body.Status, "comment": body.Comment, "processed_at": time.Now(),
	}, "", "").Eq("id", body.ID).Execute()

	// 4. ส่งแจ้งเตือนกลับ
	if len(reqData) > 0 {
		senderID := reqData[0]["sender_id"].(string)
		title := reqData[0]["title"].(string)
		rName := reqData[0]["receiver_name"].(string) // ชื่อคนกดอนุมัติ (คนรับเรื่อง)

		// กำหนดข้อความและสีตามสถานะ
		statusTxt := "✅ ได้รับอนุมัติแล้ว ✨"
		color := 5763719 // สีเขียว
		if body.Status == "rejected" {
			statusTxt = "❌ ถูกปฏิเสธ"
			color = 16729149 // สีแดง
		}

		go func() {
			fmt.Println("🚀 Updating status on Discord...")
			commentSection := body.Comment
			if commentSection == "" {
				commentSection = "-"
			}

			// สร้างข้อความ Discord
			msg := fmt.Sprintf("📢 **คำขอ:** %s\n🎭 **สถานะ:** %s\n👤 **โดย:** %s\n💬 **ข้อความ:** %s\n\n🔗 ตรวจสอบ: %s",
				title, statusTxt, rName, commentSection, APP_URL)

			services.SendDiscordEmbed("🔔 อัปเดตสถานะคำขอ", msg, color, nil, "")

			// สร้างข้อความ Push Notification
			pushMsg := statusTxt
			if body.Comment != "" {
				pushMsg = fmt.Sprintf("%s (%s)", statusTxt, body.Comment)
			}
			// ส่ง Push กลับหาคนขอ (senderID)
			services.TriggerPushNotification(senderID, "📢 สถานะคำขอ: "+title, pushMsg)
		}()
	}
	w.WriteHeader(http.StatusOK)
}

// HandleGetMyRequests: ดึงรายการคำขอทั้งหมดที่เกี่ยวข้องกับเรา (ทั้งส่งและรับ)
func HandleGetMyRequests(w http.ResponseWriter, r *http.Request) {
	if utils.EnableCORS(&w, r) {
		return
	}
	uID := r.URL.Query().Get("user_id")
	client, _ := supabase.NewClient(os.Getenv("SUPABASE_URL"), os.Getenv("SUPABASE_KEY"), nil)
	var data []map[string]interface{}

	// Query: (ฉันเป็นคนส่ง) OR (ฉันเป็นคนรับ)
	query := fmt.Sprintf("sender_id.eq.%s,receiver_id.eq.%s", uID, uID)

	// ดึงข้อมูลและเรียงจากใหม่ไปเก่า
	client.From("requests").Select("*", "exact", false).Or(query, "").Order("created_at", &postgrest.OrderOpts{Ascending: false}).ExecuteTo(&data)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}
