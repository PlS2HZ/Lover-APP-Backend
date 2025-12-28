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

// SendDiscordEmbed ส่งแจ้งเตือน Discord (คงเดิม)
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
		retryAfter := resp.Header.Get("Retry-After")
		fmt.Printf("⚠️ [RATE LIMIT] ต้องรออีก %s วินาที ถึงจะส่งได้ใหม่\n", retryAfter)
	} else if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		fmt.Println("⭐️ [SUCCESS] Sent to Discord")
	}
}

// ✅ CheckAndNotify: เช็คเวลาและแจ้งเตือน (คงเดิมตาม Logic ป้องกันการยิงซ้ำ)
// ✅ แก้ไข: ปรับการเช็คเวลาให้รองรับ Timezone ใน Database
func CheckAndNotify() {
	client, _ := supabase.NewClient(os.Getenv("SUPABASE_URL"), os.Getenv("SUPABASE_KEY"), nil)

	// 1. ดึงเวลาปัจจุบันในไทย (Format: 2025-12-27T19:11)
	nowTime := time.Now().In(loc).Truncate(time.Minute)
	nowStr := nowTime.Format("2006-01-02T15:04")

	var results []map[string]interface{}
	// 2. ดึงรายการที่ยังไม่ได้แจ้งเตือนทั้งหมดมาตรวจสอบ (เพื่อความแม่นยำเรื่อง Timezone)
	client.From("events").
		Select("*", "exact", false).
		Eq("is_notified", "false").
		ExecuteTo(&results)

	if len(results) > 0 {
		for _, ev := range results {
			eventDateStr := ev["event_date"].(string)

			// 3. แปลงเวลาจาก DB ให้เป็นเวลาไทยเพื่อเปรียบเทียบ
			// รองรับทั้งฟอร์แมต 2025-12-27 12:11:00+00 และ RFC3339
			t, err := time.Parse("2006-01-02 15:04:05-07", eventDateStr)
			if err != nil {
				t, _ = time.Parse(time.RFC3339, eventDateStr)
			}

			eventInThai := t.In(loc).Format("2006-01-02T15:04")

			// 4. ถ้าเวลาไทยตรงกับปัจจุบัน ให้ยิงแจ้งเตือน!
			if eventInThai == nowStr {
				id := ev["id"].(string)
				title := ev["title"].(string)
				desc := ev["description"].(string)
				repeat := ev["repeat_type"].(string)

				msg := fmt.Sprintf("💖 แจ้งเตือนวันสำคัญ!\n📌 **หัวข้อ:** %s\n📝 **รายละเอียด:** %s\n🔁 **วนซ้ำ:** %s", title, desc, repeat)

				// ส่ง Discord
				SendDiscordEmbed("แจ้งเตือน!", msg, 16761035, nil, "")

				// อัปเดตสถานะเป็นแจ้งเตือนแล้ว
				client.From("events").Update(map[string]interface{}{"is_notified": true}, "", "").Eq("id", id).Execute()

				// ส่ง Push PWA
				if visibleTo, ok := ev["visible_to"].([]interface{}); ok {
					for _, uid := range visibleTo {
						go TriggerPushNotification(uid.(string), "🔔 ถึงเวลาแล้วนะ!", title)
					}
				}
			}
		}
	}
}

// ✅ เพิ่มใหม่: ส่งแจ้งเตือนเมื่อมีด่าน Mind Game ใหม่
func SendMindGameNotification(creatorName string) {
	title := "🎮 ด่านใหม่มาแล้ว!"
	// ใช้ APP_URL จาก Env เพื่อความยืดหยุ่น
	appURL := os.Getenv("APP_URL")
	if appURL == "" {
		appURL = "https://lover-frontend-ashen.vercel.app/"
	}

	msg := fmt.Sprintf("✨ **มีด่านใหม่มาท้าทาย!**\n👤 สร้างโดย: **%s**\n\nลับสมองรอไว้เลย พร้อมเล่นรึยัง?\n🔗 เข้าไปเล่นที่นี่: %s",
		creatorName, appURL)

	SendDiscordEmbed(title, msg, 3066993, nil, "")
}
