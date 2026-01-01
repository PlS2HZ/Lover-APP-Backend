package services

import (
	"bytes"         // ใช้สำหรับจัดการ Buffer ข้อมูลที่จะส่งผ่าน HTTP
	"encoding/json" // ใช้สำหรับแปลงข้อมูลเป็น JSON
	"fmt"           // ใช้สำหรับจัดรูปแบบข้อความและพิมพ์ Log
	"net/http"      // ใช้สำหรับยิง Request ไปหา Discord และ Google FCM
	"os"            // ใช้สำหรับอ่าน Environment Variables
	"time"          // ใช้สำหรับจัดการเวลา

	"github.com/SherClockHolmes/webpush-go"     // Library สำหรับส่ง Web Push Notification
	"github.com/supabase-community/supabase-go" // Driver สำหรับเชื่อมต่อ Supabase
)

// กำหนด Timezone เป็น Asia/Bangkok (GMT+7) เพื่อให้เวลาตรงกับประเทศไทย
var loc = time.FixedZone("Asia/Bangkok", 7*60*60)

// ✅ getStars: ฟังก์ชันช่วยแปลงตัวเลข Priority เป็นดาว (เช่น 3 -> ⭐⭐⭐)
// ใช้สำหรับตกแต่งข้อความแจ้งเตือน Wishlist หรือ Mood
func getStars(priority int) string {
	stars := ""
	for i := 0; i < priority; i++ {
		stars += "⭐"
	}
	if stars == "" {
		return "⭐" // Default อย่างน้อย 1 ดวง ถ้าส่งมา 0
	}
	return stars
}

// SendDiscordEmbed: ส่งแจ้งเตือนเข้า Discord แบบ Embed (สวยงามกว่า Text ธรรมดา)
// รองรับการใส่สี (color), รายละเอียด (fields - ในโค้ดนี้ไม่ได้ใช้แต่เตรียมไว้), และรูปภาพ (imageURL)
func SendDiscordEmbed(title, description string, color int, fields []map[string]interface{}, imageURL string) {
	webhookURL := os.Getenv("DISCORD_WEBHOOK_URL") // อ่าน URL Webhook จาก Env
	if webhookURL == "" {
		return // ถ้าไม่ได้ตั้งค่าไว้ ก็ไม่ต้องทำอะไร
	}

	// สร้าง Payload ตามรูปแบบของ Discord API
	payload := map[string]interface{}{
		"content": "@everyone", // แท็กทุกคนในห้อง (Optional: ลบออกได้ถ้ารำคาญ)
		"embeds": []interface{}{
			map[string]interface{}{
				"title":       "💖 " + title,                                                                   // หัวข้อ
				"description": description,                                                                    // เนื้อหา
				"color":       color,                                                                          // สีแถบข้าง (Decimal Color)
				"footer":      map[string]string{"text": "Lover App • " + time.Now().In(loc).Format("15:04")}, // เวลาที่ส่ง
			},
		},
	}

	// ✅ เพิ่มรูปภาพถ้ามีการแนบมา (เช่น รูป Wishlist, รูป Mood)
	// เช็คว่ามี URL และไม่ได้เป็นคำว่า "null" (บางที Frontend ส่ง string "null" มา)
	if imageURL != "" && imageURL != "null" {
		payload["embeds"].([]interface{})[0].(map[string]interface{})["image"] = map[string]string{"url": imageURL}
	}

	// แปลง Payload เป็น JSON
	jsonData, _ := json.Marshal(payload)

	// สร้าง HTTP Client พร้อม Timeout 15 วินาที (กันค้าง)
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Post(webhookURL, "application/json", bytes.NewBuffer(jsonData))

	if err != nil {
		return // ส่งไม่ผ่าน (อาจจะเน็ตหลุด) ช่างมัน
	}
	defer resp.Body.Close()

	// เช็ค Response จาก Discord
	if resp.StatusCode == 429 {
		// กรณีส่งรัวเกินไป (Rate Limit)
		retryAfter := resp.Header.Get("Retry-After")
		fmt.Printf("⚠️ [RATE LIMIT] ต้องรออีก %s วินาที\n", retryAfter)
	} else if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		fmt.Println("⭐️ [SUCCESS] Sent to Discord") // ส่งสำเร็จ
	}
}

// TriggerPushNotification: ส่งแจ้งเตือนแบบ Push ไปที่ Browser/PWA ของผู้ใช้
func TriggerPushNotification(userID string, title string, message string) {
	// เชื่อมต่อ Supabase
	client, _ := supabase.NewClient(os.Getenv("SUPABASE_URL"), os.Getenv("SUPABASE_KEY"), nil)

	var results []map[string]interface{}
	// ดึงข้อมูล Subscription ของ User คนนี้จากตาราง push_subscriptions
	client.From("push_subscriptions").Select("subscription_json", "exact", false).Eq("user_id", userID).ExecuteTo(&results)

	// วนลูปส่งทุก Device ที่ User คนนี้ Login ไว้
	for _, res := range results {
		subStr, ok := res["subscription_json"].(string)
		if !ok {
			// กรณีข้อมูลใน DB ไม่ใช่ String (บางที Supabase เก็บเป็น JSON Object)
			b, _ := json.Marshal(res["subscription_json"])
			subStr = string(b)
		}

		// แปลง JSON String กลับเป็น Struct ของ webpush
		s := &webpush.Subscription{}
		json.Unmarshal([]byte(subStr), s)

		// ส่ง Notification ผ่าน VAPID Keys
		resp, err := webpush.SendNotification([]byte(fmt.Sprintf(`{"title":"%s", "body":"%s", "url":"/"}`, title, message)), s, &webpush.Options{
			Subscriber:      os.Getenv("VAPID_EMAIL"),
			VAPIDPublicKey:  os.Getenv("VAPID_PUBLIC_KEY"),
			VAPIDPrivateKey: os.Getenv("VAPID_PRIVATE_KEY"),
			TTL:             30, // อายุของ Noti (วินาที) ถ้าส่งไม่ผ่านในเวลานี้ให้ทิ้งไป
		})

		if err == nil {
			fmt.Printf("✅ [PUSH SUCCESS] Sent to user: %s\n", userID)
			resp.Body.Close()
		} else {
			fmt.Printf("❌ [PUSH ERROR] %v\n", err)
		}
	}
}

// CheckAndNotify: ฟังก์ชันสำหรับ Cron Job เช็คว่าถึงเวลานัดหมายหรือยัง
func CheckAndNotify() {
	client, _ := supabase.NewClient(os.Getenv("SUPABASE_URL"), os.Getenv("SUPABASE_KEY"), nil)

	// เวลาปัจจุบัน (ตัดวินาทีออก เพื่อเทียบแค่นาที)
	nowTime := time.Now().In(loc).Truncate(time.Minute)
	nowStr := nowTime.Format("2006-01-02T15:04") // รูปแบบ YYYY-MM-DDTHH:MM

	var results []map[string]interface{}
	// ดึง Event ที่ยังไม่ได้แจ้งเตือน (is_notified = false)
	client.From("events").Select("*", "exact", false).Eq("is_notified", "false").ExecuteTo(&results)

	if len(results) > 0 {
		for _, ev := range results {
			eventDateStr := ev["event_date"].(string)

			// พยายาม Parse เวลาจาก DB (Supabase บางทีส่งมาหลาย Format)
			t, err := time.Parse("2006-01-02 15:04:05-07", eventDateStr)
			if err != nil {
				t, _ = time.Parse(time.RFC3339, eventDateStr)
			}

			// แปลงเวลา Event เป็นรูปแบบเดียวกับเวลาปัจจุบันเพื่อเปรียบเทียบ
			eventInThai := t.In(loc).Format("2006-01-02T15:04")

			// ถ้าเวลาตรงกันเป๊ะ (ระดับนาที)
			if eventInThai == nowStr {
				id := ev["id"].(string)

				// 1. ส่งเข้า Discord
				msg := fmt.Sprintf("💖 แจ้งเตือนวันสำคัญ!\n📌 **หัวข้อ:** %s\n📝 **รายละเอียด:** %s", ev["title"], ev["description"])
				SendDiscordEmbed("แจ้งเตือน!", msg, 16761035, nil, "")

				// 2. อัปเดต DB ว่าแจ้งเตือนแล้ว (กันแจ้งซ้ำ)
				client.From("events").Update(map[string]interface{}{"is_notified": true}, "", "").Eq("id", id).Execute()

				// 3. ส่ง Push Noti หาคนที่เกี่ยวข้อง (Visible To)
				if visibleTo, ok := ev["visible_to"].([]interface{}); ok {
					for _, uid := range visibleTo {
						go TriggerPushNotification(uid.(string), "🔔 ถึงเวลาแล้วนะ!", ev["title"].(string))
					}
				}
			}
		}
	}
}

// SendMindGameNotification: ฟังก์ชันเฉพาะสำหรับแจ้งเตือนเกมใหม่
func SendMindGameNotification(creatorName string) {
	title := "🎮 ด่านใหม่มาแล้ว!"
	appURL := os.Getenv("APP_URL")
	if appURL == "" {
		appURL = "https://lover-frontend-ashen.vercel.app/"
	}
	// สร้างข้อความและส่งเข้า Discord
	msg := fmt.Sprintf("✨ **มีด่านใหม่มาท้าทาย!**\n👤 สร้างโดย: **%s**\n🔗 เล่นที่นี่: %s", creatorName, appURL)
	SendDiscordEmbed(title, msg, 3066993, nil, "")
}
