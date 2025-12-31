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

func HandleGetMoodInsight(w http.ResponseWriter, r *http.Request) {
	if utils.EnableCORS(&w, r) {
		return
	}

	var req struct {
		TargetID   string `json:"target_id"`
		TargetName string `json:"target_name"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	client, _ := supabase.NewClient(os.Getenv("SUPABASE_URL"), os.Getenv("SUPABASE_KEY"), nil)
	var history []map[string]interface{}

	// ดึงข้อมูล 10 อันดับล่าสุดของคนที่เลือก เพื่อหาข้อมูลย้อนหลัง 3 วัน
	client.From("daily_moods").Select("*", "exact", false).
		Eq("user_id", req.TargetID).
		Order("created_at", &postgrest.OrderOpts{Ascending: false}).
		Limit(10, "").ExecuteTo(&history)

	if len(history) == 0 {
		json.NewEncoder(w).Encode(map[string]string{"insight": "ยังไม่มีข้อมูลอารมณ์ของเขานะ ลองชวนเขามาบันทึกดูสิ ❤️"})
		return
	}

	// รวบรวมข้อความอารมณ์ย้อนหลัง
	historyText := ""
	for _, h := range history {
		historyText += fmt.Sprintf("- [%s] %s\n", h["mood_emoji"], h["mood_text"])
	}

	insight, _ := services.GetMoodInsight(req.TargetName, historyText)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"insight": insight})
}

// HandleSaveMood (คงเดิมของนาย)
func HandleSaveMood(w http.ResponseWriter, r *http.Request) {
	if utils.EnableCORS(&w, r) {
		return
	}

	var m struct {
		UserID    string   `json:"user_id"`
		MoodEmoji string   `json:"mood_emoji"`
		MoodName  string   `json:"mood_name"`
		MoodText  string   `json:"mood_text"`
		VisibleTo []string `json:"visible_to"`
	}

	if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
		http.Error(w, "Bad Request", 400)
		return
	}

	client, _ := supabase.NewClient(os.Getenv("SUPABASE_URL"), os.Getenv("SUPABASE_KEY"), nil)

	dbRow := map[string]interface{}{
		"user_id":    m.UserID,
		"mood_emoji": m.MoodEmoji,
		"mood_text":  m.MoodText,
	}

	_, _, err := client.From("daily_moods").Insert(dbRow, false, "", "", "").Execute()
	if err != nil {
		fmt.Println("❌ Supabase Insert Error:", err)
		http.Error(w, "Internal Server Error", 500)
		return
	}

	go func() {
		var user []map[string]interface{}
		client.From("users").Select("username", "exact", false).Eq("id", m.UserID).ExecuteTo(&user)
		username := "แฟนของคุณ"
		if len(user) > 0 {
			username = user[0]["username"].(string)
		}

		msg := fmt.Sprintf("**%s** ความรู้สึกตอนนี้:\n✨ **Mood:** %s %s\n💭 **รายละเอียดความรู้สึก:** %s",
			username, m.MoodEmoji, m.MoodName, m.MoodText)

		services.SendDiscordEmbed("อัปเดตอารมณ์ความรู้สึก 💖", msg, 16738740, nil, "")

		for _, targetID := range m.VisibleTo {
			if targetID != m.UserID {
				services.TriggerPushNotification(targetID, "💖 แฟนส่งความรู้สึกมานะ", m.MoodEmoji+" "+m.MoodName)
			}
		}
	}()

	w.WriteHeader(http.StatusCreated)
}

// HandleGetMoods (คงเดิมของนาย)
func HandleGetMoods(w http.ResponseWriter, r *http.Request) {
	if utils.EnableCORS(&w, r) {
		return
	}

	client, _ := supabase.NewClient(os.Getenv("SUPABASE_URL"), os.Getenv("SUPABASE_KEY"), nil)
	var results []map[string]interface{}
	client.From("daily_moods").Select("*", "exact", false).Order("created_at", &postgrest.OrderOpts{Ascending: false}).Limit(20, "").ExecuteTo(&results)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

// HandleDeleteMood (คงเดิมของนาย)
func HandleDeleteMood(w http.ResponseWriter, r *http.Request) {
	if utils.EnableCORS(&w, r) {
		return
	}

	id := r.URL.Query().Get("id")
	client, _ := supabase.NewClient(os.Getenv("SUPABASE_URL"), os.Getenv("SUPABASE_KEY"), nil)

	var oldData []map[string]interface{}
	client.From("daily_moods").Select("*", "", false).Eq("id", id).ExecuteTo(&oldData)

	client.From("daily_moods").Delete("", "").Eq("id", id).Execute()

	if len(oldData) > 0 {
		emoji := oldData[0]["mood_emoji"].(string)
		text := oldData[0]["mood_text"].(string)
		go services.SendDiscordEmbed("Mood Deleted 🗑️", fmt.Sprintf("ลบความทรงจำความรู้สึกออกไปแล้ว:\n✨ **Mood:** %s\n💭 **รายละเอียด:** %s", emoji, text), 16729149, nil, "")
	}

	w.WriteHeader(http.StatusOK)
}
