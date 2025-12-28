package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/SherClockHolmes/webpush-go"
	"github.com/supabase-community/supabase-go"
)

var loc = time.FixedZone("Asia/Bangkok", 7*60*60)

// ✅ ฟังก์ชันหัวใจ: เลือก Webhook ตามสภาพแวดล้อม
func getTargetWebhook() string {
	// 🌟 บังคับ: ถ้ามี TEST_WEBHOOK_URL ใน .env (ซึ่งมีเฉพาะในเครื่องนาย) ให้ใช้อันนั้นเสมอ
	testURL := os.Getenv("TEST_WEBHOOK_URL")
	appEnv := os.Getenv("APP_ENV")

	if appEnv == "local" && testURL != "" {
		return testURL
	}
	return os.Getenv("DISCORD_WEBHOOK_URL")
}

// TriggerPushNotification ส่งแจ้งเตือน PWA (คงเดิมตามคำสั่ง)
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

// ✅ อัปเกรด: SendDiscordEmbed (คงฟังก์ชันเดิมแต่ปรับปรุง Logic การเลือก Webhook ให้แม่นยำ 100%)
func SendDiscordEmbed(title, description string, color int, fields []map[string]interface{}, imageURL string) {
	appEnv := os.Getenv("APP_ENV")
	if appEnv != "local" && (strings.Contains(title, "ทดสอบ") || strings.Contains(description, "ทดสอบ")) {
		fmt.Println("🚫 [RENDER] Ignored test notification to prevent spamming live channel")
		return
	}

	webhookURL := getTargetWebhook() // 🌟 ใช้ตัวคัดกรองที่นี่

	if webhookURL == "" {
		return
	}

	// ถ้าเป็นเครื่อง Local ให้หน่วงเวลาเล็กน้อยกันโดน Discord เพ่งเล็ง
	if os.Getenv("APP_ENV") == "local" {
		time.Sleep(1 * time.Second) // กัน Rate Limit เวลาเทส
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
		fmt.Printf("❌ [DISCORD ERROR] %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == 429 {
		retryAfter := resp.Header.Get("Retry-After")
		fmt.Printf("⚠️ [RATE LIMIT] ต้องรออีก %s วินาที\n", retryAfter)
	} else if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		fmt.Printf("⭐️ [SUCCESS] Sent to Discord (%s)\n", os.Getenv("APP_ENV"))
	}
}

// CheckAndNotify: เช็คเวลาและแจ้งเตือน (คงเดิม)
func CheckAndNotify() {
	client, _ := supabase.NewClient(os.Getenv("SUPABASE_URL"), os.Getenv("SUPABASE_KEY"), nil)

	nowTime := time.Now().In(loc).Truncate(time.Minute)
	nowStr := nowTime.Format("2006-01-02T15:04")

	var results []map[string]interface{}
	client.From("events").
		Select("*", "exact", false).
		Eq("is_notified", "false").
		ExecuteTo(&results)

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
				title := ev["title"].(string)
				desc := ev["description"].(string)
				repeat := ev["repeat_type"].(string)

				msg := fmt.Sprintf("💖 แจ้งเตือนวันสำคัญ!\n📌 **หัวข้อ:** %s\n📝 **รายละเอียด:** %s\n🔁 **วนซ้ำ:** %s", title, desc, repeat)

				// เรียกใช้ SendDiscordEmbed ที่อัปเกรดแล้ว
				SendDiscordEmbed("แจ้งเตือน!", msg, 16761035, nil, "")

				client.From("events").Update(map[string]interface{}{"is_notified": true}, "", "").Eq("id", id).Execute()

				if visibleTo, ok := ev["visible_to"].([]interface{}); ok {
					for _, uid := range visibleTo {
						go TriggerPushNotification(uid.(string), "🔔 ถึงเวลาแล้วนะ!", title)
					}
				}
			}
		}
	}
}

// SendMindGameNotification: ส่งแจ้งเตือนเมื่อมีด่านใหม่ (คงเดิม)
func SendMindGameNotification(creatorName string) {
	title := "🎮 ด่านใหม่มาแล้ว!"
	appURL := os.Getenv("APP_URL")
	if appURL == "" {
		appURL = "https://lover-frontend-ashen.vercel.app/"
	}

	msg := fmt.Sprintf("✨ **มีด่านใหม่มาท้าทาย!**\n👤 สร้างโดย: **%s**\n\nลับสมองรอไว้เลย พร้อมเล่นรึยัง?\n🔗 เข้าไปเล่นที่นี่: %s",
		creatorName, appURL)

	// เรียกใช้ SendDiscordEmbed ที่อัปเกรดแล้ว
	SendDiscordEmbed(title, msg, 3066993, nil, "")
}
