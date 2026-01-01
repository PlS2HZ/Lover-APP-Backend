package handlers // ประกาศชื่อ package handlers

import (
	"couple-app/models"   // นำเข้า models (โครงสร้างข้อมูล Event)
	"couple-app/services" // นำเข้า services (เช่น Discord, Push Notification)
	"couple-app/utils"    // นำเข้า utils (เช่น CORS)
	"encoding/json"       // จัดการ JSON
	"fmt"                 // จัดรูปแบบข้อความ
	"net/http"            // จัดการ HTTP Server
	"os"                  // อ่าน Environment Variable
	"time"                // จัดการเวลา

	"github.com/supabase-community/postgrest-go" // Library ช่วยสร้าง Query สำหรับ Supabase
	"github.com/supabase-community/supabase-go"  // Driver เชื่อมต่อ Supabase
)

// ✅ ลบ const APP_URL ออกจากที่นี่ เพราะมีอยู่ใน social_handlers.go แล้ว (Golang มองเห็นตัวแปรใน package เดียวกันได้)

// กำหนด Timezone เป็น Asia/Bangkok (GMT+7) เพื่อให้การแสดงผลเวลาถูกต้องตามเวลาไทย
var loc = time.FixedZone("Asia/Bangkok", 7*60*60)

// HandleCreateEvent ฟังก์ชันสำหรับสร้างกิจกรรม/นัดหมายใหม่ลงในปฏิทิน
func HandleCreateEvent(w http.ResponseWriter, r *http.Request) {
	// ตรวจสอบและอนุญาต CORS (เพื่อให้ Frontend เรียก API ได้)
	if utils.EnableCORS(&w, r) {
		return
	}

	var ev models.Event                 // สร้างตัวแปรรับข้อมูลตามโครงสร้าง Event
	json.NewDecoder(r.Body).Decode(&ev) // อ่าน JSON จาก Body แล้วแปลงใส่ตัวแปร ev

	// เชื่อมต่อ Supabase
	client, _ := supabase.NewClient(os.Getenv("SUPABASE_URL"), os.Getenv("SUPABASE_KEY"), nil)

	// เตรียมข้อมูลที่จะบันทึกลงฐานข้อมูล (Map ชื่อ Field ให้ตรงกับ Database)
	row := map[string]interface{}{
		"event_date":    ev.EventDate,
		"title":         ev.Title,
		"description":   ev.Description,
		"created_by":    ev.CreatedBy,
		"visible_to":    ev.VisibleTo,                 // Array ของ UserID ที่มองเห็นได้
		"repeat_type":   ev.RepeatType,                // การวนซ้ำ (daily, monthly, yearly)
		"category_type": ev.CategoryType,              // ประเภท (normal, special)
		"is_special":    ev.CategoryType == "special", // ถ้าเป็น special ให้ตั้ง flag เป็น true
		"is_notified":   false,                        // ✅ กำหนดเป็น false เสมอเมื่อเริ่มสร้าง (รอให้ Cron Job มาเช็คและแจ้งเตือน)
	}

	// สั่ง Insert ข้อมูลลงตาราง "events"
	client.From("events").Insert(row, false, "", "", "").Execute()

	// ทำงานแบบ Asynchronous (Go Routine) เพื่อส่งแจ้งเตือนโดยไม่บล็อกการตอบกลับ API
	go func() {
		// แปลง String วันเวลาเป็น Time Object
		t, err := time.Parse(time.RFC3339, ev.EventDate)
		if err != nil {
			// ถ้า format ผิด ลอง parse แบบไม่มี Timezone
			t, _ = time.Parse("2006-01-02T15:04", ev.EventDate)
		}
		// จัดรูปแบบวันที่ให้อ่านง่ายแบบไทย (DD/MM/YYYY HH:MM)
		dateStr := t.In(loc).Format("02/01/2006 15:04")

		// สร้างข้อความแจ้งเตือนสำหรับ Discord
		msg := fmt.Sprintf("📅 **หัวข้อ:** %s\n🗓️ **วันที่/เวลา:** %s\n📝 **รายละเอียด:** %s\n🔁 **การวนซ้ำ:** %s\n\n🔗 ดูปฏิทิน: %s",
			ev.Title, dateStr, ev.Description, ev.RepeatType, APP_URL)

		// ส่ง Discord Embed (สีฟ้า: 3447003)
		services.SendDiscordEmbed("Calendar Added! 📌", msg, 3447003, nil, "")

		// ส่ง Push Notification ไปหาผู้ใช้ทุกคนที่มีสิทธิ์เห็น (VisibleTo)
		for _, uid := range ev.VisibleTo {
			services.TriggerPushNotification(uid, "📅 นัดหมายใหม่!", ev.Title+" ("+dateStr+")")
		}
	}()

	// ตอบกลับ Status 201 Created
	w.WriteHeader(http.StatusCreated)
}

// HandleDeleteEvent ฟังก์ชันลบนัดหมาย
func HandleDeleteEvent(w http.ResponseWriter, r *http.Request) {
	// จัดการ CORS
	if utils.EnableCORS(&w, r) {
		return
	}
	// รับค่า id และชื่อกิจกรรม (title) จาก Query Params
	id := r.URL.Query().Get("id")
	title := r.URL.Query().Get("title")

	// เชื่อมต่อ Supabase
	client, _ := supabase.NewClient(os.Getenv("SUPABASE_URL"), os.Getenv("SUPABASE_KEY"), nil)

	// สั่งลบข้อมูลจากตาราง events ที่มี id ตรงกัน
	client.From("events").Delete("", "").Eq("id", id).Execute()

	// สร้างข้อความแจ้งเตือนการลบ
	msg := fmt.Sprintf("🗑️ ลบนัดหมาย **'%s'** ออกจากปฏิทินแล้ว\n\n🔗 จัดการปฏิทิน: %s", title, APP_URL)
	// ส่ง Discord Embed (สีแดง: 16729149)
	go services.SendDiscordEmbed("Calendar Deleted 🗑️", msg, 16729149, nil, "")

	// ตอบกลับ Status 200 OK
	w.WriteHeader(http.StatusOK)
}

// HandleGetMyEvents ฟังก์ชันดึงรายการนัดหมายทั้งหมดของผู้ใช้
func HandleGetMyEvents(w http.ResponseWriter, r *http.Request) {
	// จัดการ CORS
	if utils.EnableCORS(&w, r) {
		return
	}
	// รับ user_id จาก Query Params
	uID := r.URL.Query().Get("user_id")

	client, _ := supabase.NewClient(os.Getenv("SUPABASE_URL"), os.Getenv("SUPABASE_KEY"), nil)

	var data []map[string]interface{} // ตัวแปรเก็บผลลัพธ์

	// สร้างเงื่อนไข Query: (เป็นคนสร้างเอง) OR (มีชื่ออยู่ใน visible_to)
	// Syntax PostgREST: field.operator.value
	// visible_to.cs.{ID} หมายถึง Array visible_to "Contains" ID นี้
	query := fmt.Sprintf("created_by.eq.%s,visible_to.cs.{%s}", uID, uID)

	// สั่ง Query โดยใช้ .Or() เพื่อรวมเงื่อนไข และเรียงลำดับตามวันที่
	client.From("events").Select("*", "exact", false).Or(query, "").Order("event_date", &postgrest.OrderOpts{Ascending: true}).ExecuteTo(&data)

	// ส่งข้อมูลกลับเป็น JSON
	json.NewEncoder(w).Encode(data)
}

// HandleGetHighlights ดึงเฉพาะรายการที่เป็น Highlight (วันสำคัญ/พิเศษ)
func HandleGetHighlights(w http.ResponseWriter, r *http.Request) {
	// จัดการ CORS
	if utils.EnableCORS(&w, r) {
		return
	}
	uID := r.URL.Query().Get("user_id")
	client, _ := supabase.NewClient(os.Getenv("SUPABASE_URL"), os.Getenv("SUPABASE_KEY"), nil)
	var data []map[string]interface{}

	// Query: เลือกเฉพาะ record ที่ is_special = true และ user นี้มีสิทธิ์เห็น
	client.From("events").Select("*", "exact", false).Eq("is_special", "true").Filter("visible_to", "cs", "{"+uID+"}").Order("event_date", &postgrest.OrderOpts{Ascending: true}).ExecuteTo(&data)

	json.NewEncoder(w).Encode(data)
}

// SaveSubscriptionHandler บันทึกข้อมูลการสมัครรับแจ้งเตือน (Web Push) ลงฐานข้อมูล
func SaveSubscriptionHandler(w http.ResponseWriter, r *http.Request) {
	// จัดการ CORS
	if utils.EnableCORS(&w, r) {
		return
	}
	// รับข้อมูล JSON (UserID และ Subscription JSON string จาก Frontend)
	var sub struct {
		UserID       string `json:"user_id"`
		Subscription string `json:"subscription"`
	}
	json.NewDecoder(r.Body).Decode(&sub)

	client, _ := supabase.NewClient(os.Getenv("SUPABASE_URL"), os.Getenv("SUPABASE_KEY"), nil)
	// Insert ข้อมูลลงตาราง push_subscriptions
	client.From("push_subscriptions").Insert(map[string]interface{}{"user_id": sub.UserID, "subscription_json": sub.Subscription}, false, "", "", "").Execute()

	w.WriteHeader(http.StatusOK)
}

// HandleUnsubscribe ยกเลิกการรับแจ้งเตือน (ลบ Subscription)
func HandleUnsubscribe(w http.ResponseWriter, r *http.Request) {
	// จัดการ CORS
	if utils.EnableCORS(&w, r) {
		return
	}
	var body struct {
		UserID string `json:"user_id"`
	}
	json.NewDecoder(r.Body).Decode(&body)

	client, _ := supabase.NewClient(os.Getenv("SUPABASE_URL"), os.Getenv("SUPABASE_KEY"), nil)
	// ลบข้อมูลจากตาราง push_subscriptions ตาม user_id
	client.From("push_subscriptions").Delete("", "").Eq("user_id", body.UserID).Execute()

	w.WriteHeader(http.StatusOK)
}

// HandleCheckSubscription ตรวจสอบว่า User นี้เปิดแจ้งเตือนไว้หรือยัง
func HandleCheckSubscription(w http.ResponseWriter, r *http.Request) {
	// จัดการ CORS
	if utils.EnableCORS(&w, r) {
		return
	}
	uID := r.URL.Query().Get("user_id")
	client, _ := supabase.NewClient(os.Getenv("SUPABASE_URL"), os.Getenv("SUPABASE_KEY"), nil)

	var results []map[string]interface{}
	// Query ดูว่ามี record ใน push_subscriptions ของ user นี้ไหม
	client.From("push_subscriptions").Select("id", "exact", false).Eq("user_id", uID).ExecuteTo(&results)

	// ส่งกลับ boolean (true ถ้าเจอข้อมูล, false ถ้าไม่เจอ)
	json.NewEncoder(w).Encode(map[string]bool{"subscribed": len(results) > 0})
}
