package handlers

import (
	"couple-app/services" // นำเข้า Service (Discord)
	"couple-app/utils"    // นำเข้า Utils (CORS)
	"encoding/json"       // จัดการ JSON
	"fmt"                 // จัดรูปแบบข้อความ
	"net/http"            // จัดการ HTTP Request/Response
	"os"                  // อ่าน Environment Variable

	"github.com/supabase-community/postgrest-go" // ตัวช่วยสร้าง Query
	"github.com/supabase-community/supabase-go"  // Driver Supabase
)

// HandleSaveWishlist: บันทึกรายการของที่อยากได้ใหม่
func HandleSaveWishlist(w http.ResponseWriter, r *http.Request) {
	// จัดการ CORS
	if utils.EnableCORS(&w, r) {
		return
	}

	// โครงสร้างรับข้อมูลจาก Frontend
	var item struct {
		UserID      string   `json:"user_id"`
		ItemName    string   `json:"item_name"`
		Description string   `json:"item_description"`
		ItemURL     string   `json:"item_url"`
		ImageURL    string   `json:"image_url"`
		Priority    int      `json:"priority"`    // ระดับความอยากได้ (1-5)
		PriceRange  string   `json:"price_range"` // ช่วงราคา
		VisibleTo   []string `json:"visible_to"`  // ใครเห็นได้บ้าง
	}
	json.NewDecoder(r.Body).Decode(&item)

	// เชื่อมต่อ Supabase
	client, _ := supabase.NewClient(os.Getenv("SUPABASE_URL"), os.Getenv("SUPABASE_KEY"), nil)
	// Insert ข้อมูลลงตาราง wishlists
	client.From("wishlists").Insert(item, false, "", "", "").Execute()

	// ทำงานเบื้องหลัง: ส่งแจ้งเตือน Discord
	go func() {
		// ดึงชื่อคนเพิ่มรายการ
		var user []map[string]interface{}
		client.From("users").Select("username", "exact", false).Eq("id", item.UserID).ExecuteTo(&user)
		username := "แฟนของคุณ"
		if len(user) > 0 {
			username = user[0]["username"].(string)
		}

		// สร้าง String ดาวตาม Priority
		stars := ""
		for i := 0; i < item.Priority; i++ {
			stars += "⭐"
		}

		// สร้างข้อความแจ้งเตือน Discord
		msg := fmt.Sprintf("**%s** ได้เพิ่มของที่อยากได้ใหม่!\n🎁 **รายการ:** %s\n🔥 **ความอยากได้:** %s\n💰 **งบประมาณ:** %s\n📝 **รายละเอียด:** %s",
			username, item.ItemName, stars, item.PriceRange, item.Description)

		// ถ้ามีลิ้งค์สินค้า ให้แนบไปด้วย
		if item.ItemURL != "" {
			msg += "\n🔗 **ลิงก์สินค้า:** " + item.ItemURL
		}

		// ส่ง Discord Embed (สีส้ม) พร้อมรูปภาพ
		services.SendDiscordEmbed("Wishlist Added! ✨", msg, 16753920, nil, item.ImageURL)
	}()

	w.WriteHeader(http.StatusCreated)
}

// HandleGetWishlist: ดึงรายการ Wishlist ทั้งหมด
func HandleGetWishlist(w http.ResponseWriter, r *http.Request) {
	// จัดการ CORS
	if utils.EnableCORS(&w, r) {
		return
	}
	client, _ := supabase.NewClient(os.Getenv("SUPABASE_URL"), os.Getenv("SUPABASE_KEY"), nil)
	var results []map[string]interface{}
	// ดึงข้อมูลทั้งหมด เรียงจากใหม่ไปเก่า
	client.From("wishlists").Select("*", "exact", false).Order("created_at", &postgrest.OrderOpts{Ascending: false}).ExecuteTo(&results)
	json.NewEncoder(w).Encode(results)
}

// ✅ HandleCompleteWish: ทำเครื่องหมายว่าได้รับของแล้ว + แจ้งเตือนละเอียด
func HandleCompleteWish(w http.ResponseWriter, r *http.Request) {
	if utils.EnableCORS(&w, r) {
		return
	}
	id := r.URL.Query().Get("id") // รับ id รายการ
	client, _ := supabase.NewClient(os.Getenv("SUPABASE_URL"), os.Getenv("SUPABASE_KEY"), nil)

	// 1. ดึงข้อมูลครบทุกฟิลด์ของรายการนี้มาก่อน (เพื่อเอาไปใส่ใน Discord Embed)
	var data []map[string]interface{}
	client.From("wishlists").Select("*", "", false).Eq("id", id).ExecuteTo(&data)

	// 2. อัปเดตสถานะ is_received เป็น true ใน Database
	client.From("wishlists").Update(map[string]interface{}{"is_received": true}, "", "").Eq("id", id).Execute()

	// 3. ส่ง Discord Notification แบบละเอียด
	if len(data) > 0 {
		d := data[0]
		stars := ""
		p, _ := d["priority"].(float64) // Supabase คืนค่าตัวเลขมาเป็น float64 เสมอใน Go interface{}
		for i := 0; i < int(p); i++ {
			stars += "⭐"
		}

		// สร้างข้อความแจ้งเตือน (คล้ายตอนเพิ่มใหม่)
		msg := fmt.Sprintf("เย้! รายการ Wishlist สำเร็จแล้วหนึ่งอย่าง:\n🎁 **รายการ:** %s\n🔥 **ความอยากได้:** %s\n💰 **งบประมาณ:** %s\n📝 **รายละเอียด:** %s",
			d["item_name"], stars, d["price_range"], d["item_description"])

		if url, ok := d["item_url"].(string); ok && url != "" {
			msg += "\n🔗 **ลิงก์สินค้า:** " + url
		}

		img := ""
		if val, ok := d["image_url"].(string); ok {
			img = val
		}

		// ส่ง Discord Embed (สีเขียว: 5763719)
		go services.SendDiscordEmbed("Wish Completed! 🎉", msg, 5763719, nil, img)
	}
	w.WriteHeader(http.StatusOK)
}

// ✅ HandleDeleteWishlist: ลบรายการ + แจ้งเตือนละเอียด
func HandleDeleteWishlist(w http.ResponseWriter, r *http.Request) {
	if utils.EnableCORS(&w, r) {
		return
	}
	id := r.URL.Query().Get("id")
	client, _ := supabase.NewClient(os.Getenv("SUPABASE_URL"), os.Getenv("SUPABASE_KEY"), nil)

	// 1. ดึงข้อมูลครบทุกฟิลด์ก่อนลบ (เพื่อเอาไปแจ้งเตือนว่าลบอะไรไป)
	var data []map[string]interface{}
	client.From("wishlists").Select("*", "", false).Eq("id", id).ExecuteTo(&data)

	// 2. สั่งลบข้อมูลออกจาก Database
	client.From("wishlists").Delete("", "").Eq("id", id).Execute()

	// 3. ส่ง Discord Notification แบบละเอียด
	if len(data) > 0 {
		d := data[0]
		stars := ""
		p, _ := d["priority"].(float64)
		for i := 0; i < int(p); i++ {
			stars += "⭐"
		}

		msg := fmt.Sprintf("ลบรายการ Wishlist ออกแล้ว:\n🎁 **รายการ:** %s\n🔥 **ความอยากได้:** %s\n💰 **งบประมาณ:** %s\n📝 **รายละเอียด:** %s",
			d["item_name"], stars, d["price_range"], d["item_description"])

		if url, ok := d["item_url"].(string); ok && url != "" {
			msg += "\n🔗 **ลิงก์สินค้า:** " + url
		}

		img := ""
		if val, ok := d["image_url"].(string); ok {
			img = val
		}

		// ส่ง Discord Embed (สีแดง: 16729149)
		go services.SendDiscordEmbed("Wishlist Deleted 🗑️", msg, 16729149, nil, img)
	}
	w.WriteHeader(http.StatusOK)
}
