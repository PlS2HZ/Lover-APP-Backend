package handlers

import (
	"couple-app/services"
	"couple-app/utils"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/supabase-community/postgrest-go"
	"github.com/supabase-community/supabase-go"
)

func HandleSaveMood(w http.ResponseWriter, r *http.Request) {
	if utils.EnableCORS(&w, r) {
		return
	}

	var m struct {
		UserID    string   `json:"user_id"`
		MoodEmoji string   `json:"mood_emoji"`
		MoodName  string   `json:"mood_name"` // ✅ สำหรับใช้ใน Discord เท่านั้น
		MoodText  string   `json:"mood_text"`
		VisibleTo []string `json:"visible_to"` // ✅ สำหรับใช้ส่ง Push/Discord
	}

	if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
		http.Error(w, "Bad Request", 400)
		return
	}

	client, _ := supabase.NewClient(os.Getenv("SUPABASE_URL"), os.Getenv("SUPABASE_KEY"), nil)

	// ✅ แก้ไข: เลือกบันทึกเฉพาะฟิลด์ที่มีอยู่ใน Database จริงๆ (user_id, mood_emoji, mood_text)
	// [อ้างอิงโครงสร้างจาก Screenshot 10.26.57 AM]
	dbRow := map[string]interface{}{
		"user_id":    m.UserID,
		"mood_emoji": m.MoodEmoji,
		"mood_text":  m.MoodText,
	}

	// บันทึกลง DB พร้อมเช็ค Error
	_, _, err := client.From("daily_moods").Insert(dbRow, false, "", "", "").Execute()
	if err != nil {
		fmt.Println("❌ Supabase Insert Error:", err)
		http.Error(w, "Internal Server Error", 500)
		return
	}

	// ✅ แจ้งเตือน Discord & Push
	go func() {
		var user []map[string]interface{}
		client.From("users").Select("username", "exact", false).Eq("id", m.UserID).ExecuteTo(&user)
		username := "แฟนของคุณ"
		if len(user) > 0 {
			username = user[0]["username"].(string)
		}

		// ปรับข้อความตามเงื่อนไข: ใส่ Mood Name (คลั่งรัก) และ รายละเอียดความรู้สึก
		msg := fmt.Sprintf("**%s** ความรู้สึกตอนนี้:\n✨ **Mood:** %s %s\n💭 **รายละเอียดความรู้สึก:** %s\n\n🔗 ดูความรู้สึกแฟน: %s",
			username, m.MoodEmoji, m.MoodName, m.MoodText, "https://lover-frontend-ashen.vercel.app/")

		services.SendDiscordEmbed("อัปเดตอารมณ์ความรู้สึก 💖", msg, 16738740, nil, "")

		for _, targetID := range m.VisibleTo {
			if targetID != m.UserID {
				services.TriggerPushNotification(targetID, "💖 แฟนส่งความรู้สึกมานะ", m.MoodEmoji+" "+m.MoodName)
			}
		}
	}()

	w.WriteHeader(http.StatusCreated)
}

func HandleGetMoods(w http.ResponseWriter, r *http.Request) {
	if utils.EnableCORS(&w, r) {
		return
	}

	client, _ := supabase.NewClient(os.Getenv("SUPABASE_URL"), os.Getenv("SUPABASE_KEY"), nil)
	var results []map[string]interface{}

	// ดึงรายการล่าสุด 20 อันดับ
	client.From("daily_moods").Select("*", "exact", false).Order("created_at", &postgrest.OrderOpts{Ascending: false}).Limit(20, "").ExecuteTo(&results)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

func HandleDeleteMood(w http.ResponseWriter, r *http.Request) {
	if utils.EnableCORS(&w, r) {
		return
	}

	id := r.URL.Query().Get("id")
	client, _ := supabase.NewClient(os.Getenv("SUPABASE_URL"), os.Getenv("SUPABASE_KEY"), nil)

	// ✅ ดึงข้อมูลก่อนลบเพื่อเอามาแจ้งเตือน
	var oldData []map[string]interface{}
	client.From("daily_moods").Select("*", "", false).Eq("id", id).ExecuteTo(&oldData)

	// ทำการลบ
	client.From("daily_moods").Delete("", "").Eq("id", id).Execute()

	if len(oldData) > 0 {
		emoji := oldData[0]["mood_emoji"].(string)
		text := oldData[0]["mood_text"].(string)
		// แจ้งเตือนเมื่อลบ พร้อมรายละเอียดครบถ้วน
		go services.SendDiscordEmbed("Mood Deleted 🗑️", fmt.Sprintf("ลบความทรงจำความรู้สึกออกไปแล้ว:\n✨ **Mood:** %s\n💭 **รายละเอียด:** %s", emoji, text), 16729149, nil, "")
	}

	w.WriteHeader(http.StatusOK)
}
