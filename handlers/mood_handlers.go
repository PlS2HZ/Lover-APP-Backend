package handlers

import (
	"couple-app/services" // นำเข้า Services (AI, Discord, Push Notification)
	"couple-app/utils"    // นำเข้า Utils (CORS)
	"encoding/json"       // จัดการ JSON
	"fmt"                 // จัดรูปแบบข้อความ
	"net/http"            // จัดการ HTTP Server
	"os"                  // อ่าน Environment Variable

	"github.com/supabase-community/postgrest-go" // ตัวช่วยสร้าง Query
	"github.com/supabase-community/supabase-go"  // Driver Supabase
)

// HandleGetMoodInsight: วิเคราะห์อารมณ์โดยใช้ AI จากประวัติ 10 รายการล่าสุด
func HandleGetMoodInsight(w http.ResponseWriter, r *http.Request) {
	// จัดการ CORS
	if utils.EnableCORS(&w, r) {
		return
	}

	// รับข้อมูล TargetID (ไอดีของคนที่จะให้วิเคราะห์) และชื่อ
	var req struct {
		TargetID   string `json:"target_id"`
		TargetName string `json:"target_name"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	// เชื่อมต่อ Supabase
	client, _ := supabase.NewClient(os.Getenv("SUPABASE_URL"), os.Getenv("SUPABASE_KEY"), nil)

	var history []map[string]interface{}
	// ดึงประวัติอารมณ์ 10 รายการล่าสุด ของ user_id นั้นๆ
	client.From("daily_moods").Select("*", "exact", false).
		Eq("user_id", req.TargetID).
		Order("created_at", &postgrest.OrderOpts{Ascending: false}). // เรียงจากใหม่ไปเก่า
		Limit(10, "").ExecuteTo(&history)

	// ถ้าไม่มีข้อมูลเลย ให้ส่งข้อความแนะนำกลับไป
	if len(history) == 0 {
		json.NewEncoder(w).Encode(map[string]string{"insight": "ยังไม่มีข้อมูลนะ ลองชวนเขาดูสิ ❤️"})
		return
	}

	// แปลงข้อมูลประวัติเป็น Text เพื่อส่งให้ AI อ่าน
	historyText := ""
	for _, h := range history {
		historyText += fmt.Sprintf("- [%s] %s\n", h["mood_emoji"], h["mood_text"])
	}

	// เรียก AI Service เพื่อวิเคราะห์
	insight, _ := services.GetMoodInsight(req.TargetName, historyText)

	// ส่งผลวิเคราะห์กลับไป
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"insight": insight})
}

// HandleSaveMood: บันทึกอารมณ์ใหม่ พร้อมส่งแจ้งเตือน
func HandleSaveMood(w http.ResponseWriter, r *http.Request) {
	// จัดการ CORS
	if utils.EnableCORS(&w, r) {
		return
	}

	// รับข้อมูลอารมณ์จาก Frontend
	var m struct {
		UserID    string   `json:"user_id"`
		MoodEmoji string   `json:"mood_emoji"`
		MoodName  string   `json:"mood_name"`
		MoodText  string   `json:"mood_text"`
		ImageURL  string   `json:"image_url"`
		VisibleTo []string `json:"visible_to"` // ใครเห็นได้บ้าง
	}
	if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
		http.Error(w, "Bad Request", 400)
		return
	}

	client, _ := supabase.NewClient(os.Getenv("SUPABASE_URL"), os.Getenv("SUPABASE_KEY"), nil)

	// เตรียมข้อมูลลง Database
	dbRow := map[string]interface{}{
		"user_id":    m.UserID,
		"mood_emoji": m.MoodEmoji,
		"mood_text":  m.MoodText,
		"image_url":  m.ImageURL,
		"visible_to": m.VisibleTo,
	}

	// Insert ลงตาราง daily_moods
	client.From("daily_moods").Insert(dbRow, false, "", "", "").Execute()

	// ทำงานเบื้องหลัง (Go Routine) เพื่อส่งแจ้งเตือนโดยไม่บล็อก Response
	go func() {
		// ดึงชื่อผู้ใช้เพื่อเอามาใส่ในแจ้งเตือน
		var user []map[string]interface{}
		client.From("users").Select("username", "exact", false).Eq("id", m.UserID).ExecuteTo(&user)
		username := "ใครบางคน"
		if len(user) > 0 {
			username = user[0]["username"].(string)
		}

		// ✅ สร้างข้อความแจ้งเตือน Discord แบบละเอียด
		msg := fmt.Sprintf("**%s** บันทึกความรู้สึกใหม่:\n✨ **Mood:** %s (%s)\n💭 **รายละเอียด:** %s",
			username, m.MoodEmoji, m.MoodName, m.MoodText)

		// ส่ง Discord Embed (สีชมพูเข้ม) พร้อมรูปภาพ (ถ้ามี)
		services.SendDiscordEmbed("New Mood & Moment 💖", msg, 16738740, nil, m.ImageURL)

		// ส่ง Push Notification หาคนที่มีสิทธิ์เห็น (VisibleTo)
		for _, targetID := range m.VisibleTo {
			if targetID != m.UserID { // ไม่ต้องแจ้งเตือนตัวเอง
				services.TriggerPushNotification(targetID, "💖 "+username+" ส่งความรู้สึกมานะ", m.MoodEmoji+" "+m.MoodName)
			}
		}
	}()

	w.WriteHeader(http.StatusCreated)
}

// HandleGetMoods: ดึงรายการอารมณ์ทั้งหมด (Feed)
func HandleGetMoods(w http.ResponseWriter, r *http.Request) {
	// จัดการ CORS
	if utils.EnableCORS(&w, r) {
		return
	}
	client, _ := supabase.NewClient(os.Getenv("SUPABASE_URL"), os.Getenv("SUPABASE_KEY"), nil)

	var results []map[string]interface{}
	// ดึงทั้งหมด 50 รายการล่าสุด เรียงจากใหม่ไปเก่า
	client.From("daily_moods").Select("*", "exact", false).Order("created_at", &postgrest.OrderOpts{Ascending: false}).Limit(50, "").ExecuteTo(&results)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

// HandleDeleteMood: ลบรายการอารมณ์
func HandleDeleteMood(w http.ResponseWriter, r *http.Request) {
	// จัดการ CORS
	if utils.EnableCORS(&w, r) {
		return
	}
	id := r.URL.Query().Get("id") // รับ id จาก Query Params
	client, _ := supabase.NewClient(os.Getenv("SUPABASE_URL"), os.Getenv("SUPABASE_KEY"), nil)

	// ดึงข้อมูลเก่าก่อนลบ (เพื่อเอาไว้แจ้งเตือนว่าลบอะไรไป)
	var oldData []map[string]interface{}
	client.From("daily_moods").Select("*", "", false).Eq("id", id).ExecuteTo(&oldData)

	// สั่งลบข้อมูล
	client.From("daily_moods").Delete("", "").Eq("id", id).Execute()

	// ถ้าเจอข้อมูลเก่า ให้ส่งแจ้งเตือนเข้า Discord
	if len(oldData) > 0 {
		d := oldData[0]
		img := ""
		if val, ok := d["image_url"].(string); ok {
			img = val
		}
		// ✅ แจ้งเตือนตอนลบให้ละเอียดขึ้น (สีแดง)
		msg := fmt.Sprintf("ข้อมูลถูกลบออกแล้ว:\n✨ **Mood:** %s\n💭 **รายละเอียดเดิม:** %s", d["mood_emoji"], d["mood_text"])
		go services.SendDiscordEmbed("Mood & Moment Deleted 🗑️", msg, 16729149, nil, img)
	}

	w.WriteHeader(http.StatusOK)
}
