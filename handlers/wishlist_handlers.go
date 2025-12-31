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

// HandleSaveWishlist (คงเดิมตามที่นายโอเคแล้ว)
func HandleSaveWishlist(w http.ResponseWriter, r *http.Request) {
	if utils.EnableCORS(&w, r) {
		return
	}
	var item struct {
		UserID      string   `json:"user_id"`
		ItemName    string   `json:"item_name"`
		Description string   `json:"item_description"`
		ItemURL     string   `json:"item_url"`
		ImageURL    string   `json:"image_url"`
		Priority    int      `json:"priority"`
		PriceRange  string   `json:"price_range"`
		VisibleTo   []string `json:"visible_to"`
	}
	json.NewDecoder(r.Body).Decode(&item)
	client, _ := supabase.NewClient(os.Getenv("SUPABASE_URL"), os.Getenv("SUPABASE_KEY"), nil)
	client.From("wishlists").Insert(item, false, "", "", "").Execute()

	go func() {
		var user []map[string]interface{}
		client.From("users").Select("username", "exact", false).Eq("id", item.UserID).ExecuteTo(&user)
		username := "แฟนของคุณ"
		if len(user) > 0 {
			username = user[0]["username"].(string)
		}

		stars := ""
		for i := 0; i < item.Priority; i++ {
			stars += "⭐"
		}

		msg := fmt.Sprintf("**%s** ได้เพิ่มของที่อยากได้ใหม่!\n🎁 **รายการ:** %s\n🔥 **ความอยากได้:** %s\n💰 **งบประมาณ:** %s\n📝 **รายละเอียด:** %s",
			username, item.ItemName, stars, item.PriceRange, item.Description)

		if item.ItemURL != "" {
			msg += "\n🔗 **ลิงก์สินค้า:** " + item.ItemURL
		}
		services.SendDiscordEmbed("Wishlist Added! ✨", msg, 16753920, nil, item.ImageURL)
	}()
	w.WriteHeader(http.StatusCreated)
}

func HandleGetWishlist(w http.ResponseWriter, r *http.Request) {
	if utils.EnableCORS(&w, r) {
		return
	}
	client, _ := supabase.NewClient(os.Getenv("SUPABASE_URL"), os.Getenv("SUPABASE_KEY"), nil)
	var results []map[string]interface{}
	client.From("wishlists").Select("*", "exact", false).Order("created_at", &postgrest.OrderOpts{Ascending: false}).ExecuteTo(&results)
	json.NewEncoder(w).Encode(results)
}

// ✅ แก้ไข: เพิ่มรายละเอียดการแจ้งเตือนตอน Complete ให้ครบเหมือนตอนบันทึกใหม่
func HandleCompleteWish(w http.ResponseWriter, r *http.Request) {
	if utils.EnableCORS(&w, r) {
		return
	}
	id := r.URL.Query().Get("id")
	client, _ := supabase.NewClient(os.Getenv("SUPABASE_URL"), os.Getenv("SUPABASE_KEY"), nil)

	// 1. ดึงข้อมูลครบทุกฟิลด์ก่อนอัปเดต
	var data []map[string]interface{}
	client.From("wishlists").Select("*", "", false).Eq("id", id).ExecuteTo(&data)

	// 2. อัปเดตสถานะ
	client.From("wishlists").Update(map[string]interface{}{"is_received": true}, "", "").Eq("id", id).Execute()

	// 3. ส่ง Discord แบบละเอียด
	if len(data) > 0 {
		d := data[0]
		stars := ""
		p, _ := d["priority"].(float64) // Supabase return numeric as float64
		for i := 0; i < int(p); i++ {
			stars += "⭐"
		}

		msg := fmt.Sprintf("เย้! รายการ Wishlist สำเร็จแล้วหนึ่งอย่าง:\n🎁 **รายการ:** %s\n🔥 **ความอยากได้:** %s\n💰 **งบประมาณ:** %s\n📝 **รายละเอียด:** %s",
			d["item_name"], stars, d["price_range"], d["item_description"])

		if url, ok := d["item_url"].(string); ok && url != "" {
			msg += "\n🔗 **ลิงก์สินค้า:** " + url
		}

		img := ""
		if val, ok := d["image_url"].(string); ok {
			img = val
		}
		go services.SendDiscordEmbed("Wish Completed! 🎉", msg, 5763719, nil, img)
	}
	w.WriteHeader(http.StatusOK)
}

// ✅ แก้ไข: เพิ่มรายละเอียดการแจ้งเตือนตอน Delete ให้ครบเหมือนตอนบันทึกใหม่
func HandleDeleteWishlist(w http.ResponseWriter, r *http.Request) {
	if utils.EnableCORS(&w, r) {
		return
	}
	id := r.URL.Query().Get("id")
	client, _ := supabase.NewClient(os.Getenv("SUPABASE_URL"), os.Getenv("SUPABASE_KEY"), nil)

	// 1. ดึงข้อมูลครบทุกฟิลด์ก่อนลบ
	var data []map[string]interface{}
	client.From("wishlists").Select("*", "", false).Eq("id", id).ExecuteTo(&data)

	// 2. ทำการลบ
	client.From("wishlists").Delete("", "").Eq("id", id).Execute()

	// 3. ส่ง Discord แบบละเอียด
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
		go services.SendDiscordEmbed("Wishlist Deleted 🗑️", msg, 16729149, nil, img)
	}
	w.WriteHeader(http.StatusOK)
}
