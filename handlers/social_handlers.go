package handlers

import (
	"couple-app/services"
	"couple-app/utils"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	_ "strings"
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

	var targetUser []map[string]interface{}
	client.From("users").Select("id", "exact", false).Eq("username", req.ReceiverUsername).ExecuteTo(&targetUser)
	if len(targetUser) == 0 {
		http.Error(w, "Receiver Not Found", 404)
		return
	}
	rID := targetUser[0]["id"].(string)

	var senderUser []map[string]interface{}
	client.From("users").Select("username", "exact", false).Eq("id", req.SenderID).ExecuteTo(&senderUser)
	sName := "Unknown"
	if len(senderUser) > 0 {
		sName = senderUser[0]["username"].(string)
	}

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

	go func() {
		// ✅ แก้ไข: ปรับการ Parse เวลาให้รองรับ Format จาก HTML datetime-local (2006-01-02T15:04)
		parseTime := func(iso string) string {
			// ลอง parse แบบ ISO8601 ก่อน (RFC3339)
			t, err := time.Parse(time.RFC3339, iso)
			if err != nil {
				// ถ้าพลาด ให้ลอง parse แบบ HTML Input datetime-local
				t, err = time.Parse("2006-01-02T15:04", iso)
			}
			if err != nil {
				return iso // ถ้าไม่ได้จริงๆ ให้ส่งค่าดิบกลับไป
			}
			return t.Format("02/01/2006 เวลา 15:04")
		}

		formattedStart := parseTime(req.StartTime)
		formattedEnd := parseTime(req.EndTime)

		// ✅ แก้ไข: เพิ่มหัวข้อ "ถึงคุณ:" และจัดระเบียบข้อความใหม่
		msg := fmt.Sprintf("👤 **จาก:** %s\n🎯 **ถึงคุณ:** %s\n🏷️ **ประเภท:** %s\n📖 **รายละเอียด:** %s\n⏰ **เริ่ม:** %s\n🏁 **สิ้นสุด:** %s\n⏳ **ระยะเวลา:** %s\n\n🔗 เข้าแอปที่นี่: %s",
			sName, req.ReceiverUsername, req.Header, req.Title, formattedStart, formattedEnd, req.Duration, APP_URL)

		services.SendDiscordEmbed("💌 มีคำขอใหม่รอการอนุมัติ!", msg, 16738740, nil, req.ImageURL)
		services.TriggerPushNotification(rID, "💌 มีคำขอใหม่จาก "+sName, req.Title)
	}()

	w.WriteHeader(http.StatusCreated)
}

// HandleUpdateStatus อัปเดตสถานะ และส่งแจ้งเตือนพร้อมคอมเมนต์ (เหมือนเดิม)
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
			commentSection := body.Comment
			if commentSection == "" {
				commentSection = "-"
			}

			msg := fmt.Sprintf("📢 **คำขอ:** %s\n🎭 **สถานะ:** %s\n👤 **โดย:** %s\n💬 **ข้อความ:** %s\n\n🔗 ตรวจสอบ: %s",
				title, statusTxt, rName, commentSection, APP_URL)

			services.SendDiscordEmbed("🔔 อัปเดตสถานะคำขอ", msg, color, nil, "")

			pushMsg := statusTxt
			if body.Comment != "" {
				pushMsg = fmt.Sprintf("%s (%s)", statusTxt, body.Comment)
			}
			services.TriggerPushNotification(senderID, "📢 สถานะคำขอ: "+title, pushMsg)
		}()
	}
	w.WriteHeader(http.StatusOK)
}

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
