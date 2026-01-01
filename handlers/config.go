package handlers // ประกาศ Package handlers

import (
	"couple-app/models" // นำเข้า models เพื่อใช้งาน Struct (เช่น HomeConfig)
	"couple-app/utils"  // นำเข้า utils เพื่อใช้งานฟังก์ชัน CORS
	"encoding/json"     // นำเข้า package สำหรับจัดการ JSON (Encode/Decode)
	"net/http"          // นำเข้า package สำหรับจัดการ HTTP Request/Response
	"os"                // นำเข้า package สำหรับอ่าน Environment Variables

	"github.com/supabase-community/supabase-go" // นำเข้า Library Supabase
)

// ✅ ต้องมี jwtKey ประกาศที่นี่ตัวเดียวพอ เพื่อให้ auth_handlers เรียกใช้ได้
// กำหนด Secret Key สำหรับใช้ในการ Sign และ Verify JWT Token (ใน Package เดียวกันจะเรียกใช้ตัวแปรนี้ได้)
var jwtKey = []byte("your_secret_key_2025")

// HandleGetHomeConfig ฟังก์ชันสำหรับดึงข้อมูลการตั้งค่าหน้า Home (รูปภาพ, ข้อความ)
func HandleGetHomeConfig(w http.ResponseWriter, r *http.Request) {
	// เรียกใช้ฟังก์ชันจัดการ CORS เพื่อให้ Frontend ต่างโดเมนเรียกใช้ได้
	if utils.EnableCORS(&w, r) {
		return // ถ้าเป็น OPTIONS Request ให้จบการทำงาน
	}

	// สร้าง Client เชื่อมต่อ Supabase โดยอ่าน URL และ Key จาก Env
	client, _ := supabase.NewClient(os.Getenv("SUPABASE_URL"), os.Getenv("SUPABASE_KEY"), nil)

	var results []map[string]interface{} // ตัวแปรสำหรับเก็บผลลัพธ์จาก Database
	// สั่ง Query ข้อมูลทั้งหมด (*) จากตาราง "home_configs"
	client.From("home_configs").Select("*", "exact", false).ExecuteTo(&results)

	// แปลงผลลัพธ์เป็น JSON และส่งกลับไปยัง Client
	json.NewEncoder(w).Encode(results)
}

// HandleUpdateHomeConfig ฟังก์ชันสำหรับอัปเดตการตั้งค่าหน้า Home
func HandleUpdateHomeConfig(w http.ResponseWriter, r *http.Request) {
	// จัดการ CORS
	if utils.EnableCORS(&w, r) {
		return
	}

	var config models.HomeConfig            // ประกาศตัวแปร config ตามโครงสร้างใน models
	json.NewDecoder(r.Body).Decode(&config) // อ่าน JSON จาก Body Request แล้วแปลงใส่ตัวแปร config

	// สร้าง Client เชื่อมต่อ Supabase
	client, _ := supabase.NewClient(os.Getenv("SUPABASE_URL"), os.Getenv("SUPABASE_KEY"), nil)

	// ลบข้อมูลเก่าที่มี config_type เดียวกันออกก่อน (เพื่อให้ข้อมูลไม่อัปเดตซ้อนกันหรือซ้ำซ้อน)
	client.From("home_configs").Delete("", "").Eq("config_type", config.ConfigType).Execute()

	// เพิ่มข้อมูลใหม่ (config ที่รับมา) ลงไปในตาราง "home_configs"
	client.From("home_configs").Insert(config, false, "", "", "").Execute()

	// ส่ง HTTP Status 200 OK กลับไปบอกว่าสำเร็จ
	w.WriteHeader(http.StatusOK)
}
