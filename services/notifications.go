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

// ✅ เพิ่มฟังก์ชันช่วยแปลงตัวเลข Priority เป็นดาว (เพื่อใช้ใน Wishlist)
func getStars(priority int) string {
	stars := ""
	for i := 0; i < priority; i++ {
		stars += "⭐"
	}
	if stars == "" {
		return "⭐" // Default อย่างน้อย 1 ดวง
	}
	return stars
}

// SendDiscordEmbed ส่งแจ้งเตือน Discord (ฉบับอัปเกรดให้โชว์รูปและรายละเอียดครบ)
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

	// ✅ แสดงรูปภาพถ้ามีการแนบมา (ใช้สำหรับ Wishlist และ Moments)
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
		fmt.Printf("⚠️ [RATE LIMIT] ต้องรออีก %s วินาที\n", retryAfter)
	} else if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		fmt.Println("⭐️ [SUCCESS] Sent to Discord")
	}
}

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
			fmt.Printf("✅ [PUSH SUCCESS] Sent to user: %s\n", userID)
			resp.Body.Close()
		} else {
			fmt.Printf("❌ [PUSH ERROR] %v\n", err)
		}
	}
}

// CheckAndNotify: เช็คเวลาและแจ้งเตือนวันสำคัญ (คงเดิม)
func CheckAndNotify() {
	client, _ := supabase.NewClient(os.Getenv("SUPABASE_URL"), os.Getenv("SUPABASE_KEY"), nil)
	nowTime := time.Now().In(loc).Truncate(time.Minute)
	nowStr := nowTime.Format("2006-01-02T15:04")

	var results []map[string]interface{}
	client.From("events").Select("*", "exact", false).Eq("is_notified", "false").ExecuteTo(&results)

	if len(results) > 0 {
		for _, ev := range results {
			eventDateStr := ev["event_date"].(string)
			t, err := time.Parse("2006-01-02 15:04:05-07", eventDateStr)
			if err != nil {
				t, _ = time.Parse(time.RFC3339, eventDateStr)
			}
			eventInThai := t.In(loc).Format("2006-01-02T15:04")

			if eventInThai == nowStr {
				id := ev["id"].(string)
				msg := fmt.Sprintf("💖 แจ้งเตือนวันสำคัญ!\n📌 **หัวข้อ:** %s\n📝 **รายละเอียด:** %s", ev["title"], ev["description"])
				SendDiscordEmbed("แจ้งเตือน!", msg, 16761035, nil, "")
				client.From("events").Update(map[string]interface{}{"is_notified": true}, "", "").Eq("id", id).Execute()

				if visibleTo, ok := ev["visible_to"].([]interface{}); ok {
					for _, uid := range visibleTo {
						go TriggerPushNotification(uid.(string), "🔔 ถึงเวลาแล้วนะ!", ev["title"].(string))
					}
				}
			}
		}
	}
}

// SendMindGameNotification แจ้งเตือน Mind Game (คงเดิม)
func SendMindGameNotification(creatorName string) {
	title := "🎮 ด่านใหม่มาแล้ว!"
	appURL := os.Getenv("APP_URL")
	if appURL == "" {
		appURL = "https://lover-frontend-ashen.vercel.app/"
	}
	msg := fmt.Sprintf("✨ **มีด่านใหม่มาท้าทาย!**\n👤 สร้างโดย: **%s**\n🔗 เล่นที่นี่: %s", creatorName, appURL)
	SendDiscordEmbed(title, msg, 3066993, nil, "")
}
