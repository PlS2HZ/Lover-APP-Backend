package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/SherClockHolmes/webpush-go"
	"github.com/supabase-community/supabase-go"
)

var loc = time.FixedZone("Asia/Bangkok", 7*60*60)

// TriggerPushNotification ส่งแจ้งเตือน PWA (คงเดิม)
func TriggerPushNotification(userID string, title string, message string) {
	client, _ := supabase.NewClient(os.Getenv("SUPABASE_URL"), os.Getenv("SUPABASE_KEY"), nil)
	var results []map[string]interface{}
	client.From("push_subscriptions").Select("subscription_json", "exact", false).Eq("user_id", userID).ExecuteTo(&results)

	for _, res := range results {
		subStr, ok := res["subscription_json"].(string)
		if !ok {
			b, _ := json.Marshal(res["subscription_json"])
			subStr = string(b)
		}
		s := &webpush.Subscription{}
		json.Unmarshal([]byte(subStr), s)
		resp, err := webpush.SendNotification([]byte(fmt.Sprintf(`{"title":"%s", "body":"%s", "url":"/"}`, title, message)), s, &webpush.Options{
			Subscriber:      os.Getenv("VAPID_EMAIL"),
			VAPIDPublicKey:  os.Getenv("VAPID_PUBLIC_KEY"),
			VAPIDPrivateKey: os.Getenv("VAPID_PRIVATE_KEY"),
			TTL:             30,
		})
		if err == nil {
			resp.Body.Close()
		} else {
			fmt.Printf("❌ [PUSH ERROR] %v\n", err)
		}
	}
}

// SendDiscordEmbed ส่งแจ้งเตือน Discord (เวอร์ชันที่นายส่งมาล่าสุด)
func SendDiscordEmbed(title, description string, color int, fields []map[string]interface{}, imageURL string) {
	webhookURL := os.Getenv("DISCORD_WEBHOOK_URL")
	if webhookURL == "" {
		return
	}

	payload := map[string]interface{}{
		"content": "@everyone",
		"embeds": []interface{}{
			map[string]interface{}{
				"title":       "💖 " + title,
				"description": description,
				"color":       color,
				"footer":      map[string]string{"text": "Lover App • " + time.Now().In(loc).Format("15:04")},
			},
		},
	}
	if imageURL != "" && imageURL != "null" {
		payload["embeds"].([]interface{})[0].(map[string]interface{})["image"] = map[string]string{"url": imageURL}
	}

	jsonData, _ := json.Marshal(payload)
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Post(webhookURL, "application/json", bytes.NewBuffer(jsonData))

	if err != nil {
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == 429 {
		fmt.Println("⚠️ [RATE LIMIT] Discord blocks us. Slow down!")
	} else if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		fmt.Println("⭐️ [SUCCESS] Sent to Discord")
	}
}

// ✅ แก้ไข: เพิ่มการเช็ค is_notified เพื่อป้องกันการยิงซ้ำ
func CheckAndNotify() {
	client, _ := supabase.NewClient(os.Getenv("SUPABASE_URL"), os.Getenv("SUPABASE_KEY"), nil)

	// ดึงเวลาปัจจุบันในไทย
	now := time.Now().In(loc).Truncate(time.Minute).Format("2006-01-02T15:04")

	var results []map[string]interface{}
	// ✅ เพิ่มเงื่อนไข: ดึงเฉพาะรายการที่เวลาตรงกัน และ ยังไม่ได้แจ้งเตือน (is_notified = false)
	client.From("events").
		Select("*", "exact", false).
		Like("event_date", now+"%").
		Eq("is_notified", "false").
		ExecuteTo(&results)

	if len(results) > 0 {
		for _, ev := range results {
			id := ev["id"].(string)
			title := ev["title"].(string)
			desc := ev["description"].(string)
			repeat := ev["repeat_type"].(string)

			msg := fmt.Sprintf("💖 แจ้งเตือนวันสำคัญ!\n📌 **หัวข้อ:** %s\n📝 **รายละเอียด:** %s\n🔁 **วนซ้ำ:** %s", title, desc, repeat)

			// 1. ส่ง Discord
			SendDiscordEmbed("แจ้งเตือน!", msg, 16761035, nil, "")

			// 2. อัปเดตสถานะเป็น "แจ้งเตือนแล้ว" ทันที เพื่อไม่ให้ส่งซ้ำ
			client.From("events").Update(map[string]interface{}{"is_notified": true}, "", "").Eq("id", id).Execute()

			// 3. ส่ง Push Notification ให้ทุกคนที่เกี่ยวข้อง
			if visibleTo, ok := ev["visible_to"].([]interface{}); ok {
				for _, uid := range visibleTo {
					go TriggerPushNotification(uid.(string), "🔔 ถึงเวลาแล้วนะ!", title)
				}
			}
		}
	}
}
