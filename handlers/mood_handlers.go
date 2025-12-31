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
	client.From("daily_moods").Select("*", "exact", false).
		Eq("user_id", req.TargetID).
		Order("created_at", &postgrest.OrderOpts{Ascending: false}).
		Limit(10, "").ExecuteTo(&history)

	if len(history) == 0 {
		json.NewEncoder(w).Encode(map[string]string{"insight": "ยังไม่มีข้อมูลนะ ลองชวนเขาดูสิ ❤️"})
		return
	}

	historyText := ""
	for _, h := range history {
		historyText += fmt.Sprintf("- [%s] %s\n", h["mood_emoji"], h["mood_text"])
	}
	insight, _ := services.GetMoodInsight(req.TargetName, historyText)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"insight": insight})
}

func HandleSaveMood(w http.ResponseWriter, r *http.Request) {
	if utils.EnableCORS(&w, r) {
		return
	}
	var m struct {
		UserID    string   `json:"user_id"`
		MoodEmoji string   `json:"mood_emoji"`
		MoodName  string   `json:"mood_name"`
		MoodText  string   `json:"mood_text"`
		ImageURL  string   `json:"image_url"`
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
		"image_url":  m.ImageURL,
		"visible_to": m.VisibleTo,
	}

	client.From("daily_moods").Insert(dbRow, false, "", "", "").Execute()

	go func() {
		var user []map[string]interface{}
		client.From("users").Select("username", "exact", false).Eq("id", m.UserID).ExecuteTo(&user)
		username := "ใครบางคน"
		if len(user) > 0 {
			username = user[0]["username"].(string)
		}

		// ✅ แจ้งเตือนครบทุกรายละเอียด
		msg := fmt.Sprintf("**%s** บันทึกความรู้สึกใหม่:\n✨ **Mood:** %s (%s)\n💭 **รายละเอียด:** %s",
			username, m.MoodEmoji, m.MoodName, m.MoodText)

		services.SendDiscordEmbed("New Mood & Moment 💖", msg, 16738740, nil, m.ImageURL)

		for _, targetID := range m.VisibleTo {
			if targetID != m.UserID {
				services.TriggerPushNotification(targetID, "💖 "+username+" ส่งความรู้สึกมานะ", m.MoodEmoji+" "+m.MoodName)
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
	// ดึงทั้งหมด 50 รายการล่าสุด
	client.From("daily_moods").Select("*", "exact", false).Order("created_at", &postgrest.OrderOpts{Ascending: false}).Limit(50, "").ExecuteTo(&results)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

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
		d := oldData[0]
		img := ""
		if val, ok := d["image_url"].(string); ok {
			img = val
		}
		// ✅ แจ้งเตือนตอนลบให้ละเอียดขึ้น
		msg := fmt.Sprintf("ข้อมูลถูกลบออกแล้ว:\n✨ **Mood:** %s\n💭 **รายละเอียดเดิม:** %s", d["mood_emoji"], d["mood_text"])
		go services.SendDiscordEmbed("Mood & Moment Deleted 🗑️", msg, 16729149, nil, img)
	}
	w.WriteHeader(http.StatusOK)
}
