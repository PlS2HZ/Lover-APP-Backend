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

func HandleSaveWishlist(w http.ResponseWriter, r *http.Request) {
	if utils.EnableCORS(&w, r) {
		return
	}
	var item struct {
		UserID      string   `json:"user_id"`
		ItemName    string   `json:"item_name"`
		Description string   `json:"item_description"`
		ItemURL     string   `json:"item_url"`
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

		// ✅ ปรับข้อความตามที่นายต้องการ
		msg := fmt.Sprintf("**%s** ได้เพิ่มของที่อยากได้:\n🎁 **รายการที่อยากได้:** %s\n📝 **รายละเอียด:** %s",
			username, item.ItemName, item.Description)
		if item.ItemURL != "" {
			msg += "\n🔗 **ลิงก์สินค้า:** " + item.ItemURL
		}
		msg += "\n\n🔗 จัดการ Wishlist: " + APP_URL

		services.SendDiscordEmbed("Wishlist Added! ✨", msg, 16753920, nil, "")
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

func HandleCompleteWish(w http.ResponseWriter, r *http.Request) {
	if utils.EnableCORS(&w, r) {
		return
	}
	id := r.URL.Query().Get("id")
	client, _ := supabase.NewClient(os.Getenv("SUPABASE_URL"), os.Getenv("SUPABASE_KEY"), nil)

	// ✅ ดึงข้อมูลก่อนอัปเดต
	var data []map[string]interface{}
	client.From("wishlists").Select("item_name, item_description", "", false).Eq("id", id).ExecuteTo(&data)

	client.From("wishlists").Update(map[string]interface{}{"is_received": true}, "", "").Eq("id", id).Execute()

	if len(data) > 0 {
		name := data[0]["item_name"].(string)
		desc := data[0]["item_description"].(string)
		msg := fmt.Sprintf("เย้! รายการ Wishlist สำเร็จแล้วหนึ่งอย่าง:\n🎁 **รายการสินค้า:** %s\n📝 **รายละเอียด:** %s", name, desc)
		go services.SendDiscordEmbed("Wish Completed! 🎉", msg, 5763719, nil, "")
	}
	w.WriteHeader(http.StatusOK)
}

func HandleDeleteWishlist(w http.ResponseWriter, r *http.Request) {
	if utils.EnableCORS(&w, r) {
		return
	}
	id := r.URL.Query().Get("id")
	client, _ := supabase.NewClient(os.Getenv("SUPABASE_URL"), os.Getenv("SUPABASE_KEY"), nil)

	// ✅ ดึงข้อมูลก่อนลบ
	var data []map[string]interface{}
	client.From("wishlists").Select("item_name, item_description", "", false).Eq("id", id).ExecuteTo(&data)

	client.From("wishlists").Delete("", "").Eq("id", id).Execute()

	if len(data) > 0 {
		name := data[0]["item_name"].(string)
		desc := data[0]["item_description"].(string)
		go services.SendDiscordEmbed("Wish Deleted 🗑️", fmt.Sprintf("ลบรายการ Wishlist ออกแล้ว:\n🎁 **รายการ:** %s\n📝 **รายละเอียด:** %s", name, desc), 16729149, nil, "")
	}
	w.WriteHeader(http.StatusOK)
}
