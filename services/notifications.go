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
		if err != nil {
			fmt.Printf("❌ [PUSH ERROR] %v\n", err)
		} else {
			resp.Body.Close()
		}
	}
}

func SendDiscordEmbed(title, description string, color int, fields []map[string]interface{}, imageURL string) {
	webhookURL := os.Getenv("DISCORD_WEBHOOK_URL")
	if webhookURL == "" {
		fmt.Println("❌ [ERROR] NO WEBHOOK URL")
		return
	}

	embed := map[string]interface{}{
		"title":       "💖 " + title,
		"description": description,
		"color":       color,
		"footer": map[string]string{
			"text": "Lover App • " + time.Now().In(loc).Format("02 Jan 15:04"),
		},
	}
	if imageURL != "" && imageURL != "null" {
		embed["image"] = map[string]string{"url": imageURL}
	}

	payload := map[string]interface{}{
		"content": "@everyone",
		"embeds":  []interface{}{embed},
	}
	jsonData, _ := json.Marshal(payload)

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Post(webhookURL, "application/json", bytes.NewBuffer(jsonData))

	if err != nil {
		fmt.Printf("❌ [CRITICAL] API CONNECTION ERROR: %v\n", err)
		return
	}
	defer resp.Body.Close()

	// ✅ ตรวจสอบ Status จริงจาก Discord
	if resp.StatusCode == 429 {
		fmt.Println("⚠️ [DISCORD WARNING] RATE LIMITED! PLEASE WAIT 5 MIN.")
	} else if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		fmt.Printf("❌ [DISCORD ERROR] STATUS: %d\n", resp.StatusCode)
	} else {
		fmt.Println("⭐️ [SUCCESS] DISCORD MESSAGE SENT!")
	}
}

func CheckAndNotify() {
	client, _ := supabase.NewClient(os.Getenv("SUPABASE_URL"), os.Getenv("SUPABASE_KEY"), nil)
	now := time.Now().In(loc).Truncate(time.Minute).Format("2006-01-02T15:04:00.000Z")
	var results []map[string]interface{}
	client.From("events").Select("*", "exact", false).Eq("event_date", now).ExecuteTo(&results)
	if len(results) > 0 {
		for _, ev := range results {
			title := ev["title"].(string)
			SendDiscordEmbed("🔔 แจ้งเตือนวันสำคัญ!", title, 16761035, nil, "")
		}
	}
}
