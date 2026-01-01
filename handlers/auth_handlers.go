package handlers // ประกาศชื่อ package ว่า handlers (เอาไว้รวมฟังก์ชันที่จัดการ Request)

import (
	"couple-app/utils" // นำเข้า package utils ของโปรเจกต์ (น่าจะใช้สำหรับ CORS)
	"encoding/json"    // นำเข้า package สำหรับจัดการข้อมูลแบบ JSON (Encode/Decode)
	"net/http"         // นำเข้า package สำหรับทำ Web Server และจัดการ HTTP Request/Response
	"os"               // นำเข้า package สำหรับอ่าน Environment Variables (เช่น URL, KEY)
	"time"             // นำเข้า package สำหรับจัดการเวลา (ใช้ตั้งเวลาหมดอายุ Token)

	"couple-app/models" // นำเข้า models ของโปรเจกต์ (โครงสร้างข้อมูล User)

	"github.com/golang-jwt/jwt/v5"              // นำเข้า Library สำหรับสร้างและตรวจสอบ JWT Token
	"github.com/supabase-community/supabase-go" // นำเข้า Library สำหรับเชื่อมต่อ Supabase
	"golang.org/x/crypto/bcrypt"                // นำเข้า Library สำหรับเข้ารหัสรหัสผ่าน (Hash)
)

// HandleRegister จัดการการลงทะเบียนผู้ใช้ใหม่
func HandleRegister(w http.ResponseWriter, r *http.Request) {
	// เรียกใช้ฟังก์ชัน EnableCORS จาก utils เพื่ออนุญาตให้ Frontend (คนละโดเมน) เรียกใช้งานได้
	if utils.EnableCORS(&w, r) {
		return // ถ้าเป็น Preflight request (OPTIONS) ให้จบการทำงานตรงนี้
	}

	var u models.User                  // ประกาศตัวแปร u ตามโครงสร้าง User
	json.NewDecoder(r.Body).Decode(&u) // อ่านข้อมูล JSON จาก Body ของ Request แล้วแปลงใส่ตัวแปร u

	// ทำการ Hash รหัสผ่านด้วย bcrypt (cost = 10) เพื่อความปลอดภัย (ไม่เก็บรหัสจริง)
	hashed, _ := bcrypt.GenerateFromPassword([]byte(u.Password), 10)

	// สร้าง Client เชื่อมต่อ Supabase โดยอ่าน URL และ Key จาก Environment Variables
	client, _ := supabase.NewClient(os.Getenv("SUPABASE_URL"), os.Getenv("SUPABASE_KEY"), nil)

	// สั่ง Insert ข้อมูลลงตาราง "users" (ชื่อผู้ใช้ และ รหัสผ่านที่ Hash แล้ว)
	client.From("users").Insert(map[string]interface{}{"username": u.Username, "password": string(hashed)}, false, "", "", "").Execute()

	w.WriteHeader(201) // ส่ง Status Code 201 (Created) กลับไปบอกว่าสร้างสำเร็จ
}

// HandleLogin จัดการการเข้าสู่ระบบและสร้าง JWT
func HandleLogin(w http.ResponseWriter, r *http.Request) {
	// จัดการ CORS เพื่อให้ Frontend เรียกได้
	if utils.EnableCORS(&w, r) {
		return
	}

	// ประกาศโครงสร้างตัวแปรชั่วคราวเพื่อรับ Username/Password จาก Login Form
	var c struct{ Username, Password string }
	json.NewDecoder(r.Body).Decode(&c) // แปลง JSON จาก Request Body ใส่ตัวแปร c

	// เชื่อมต่อ Supabase
	client, _ := supabase.NewClient(os.Getenv("SUPABASE_URL"), os.Getenv("SUPABASE_KEY"), nil)

	var users []map[string]interface{} // ตัวแปรเก็บผลลัพธ์การค้นหา User
	// ค้นหา User จากตาราง "users" ที่มี username ตรงกับที่ส่งมา
	client.From("users").Select("*", "exact", false).Eq("username", c.Username).ExecuteTo(&users)

	// ตรวจสอบว่าเจอ User ไหม (len > 0) และ รหัสผ่านที่กรอกมา ตรงกับ Hash ในฐานข้อมูลไหม
	if len(users) > 0 && bcrypt.CompareHashAndPassword([]byte(users[0]["password"].(string)), []byte(c.Password)) == nil {
		// ถ้าถูกต้อง: สร้าง JWT Token
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"user_id": users[0]["id"],                        // ใส่ user_id ลงใน Token
			"exp":     time.Now().Add(72 * time.Hour).Unix(), // ตั้งวันหมดอายุ 72 ชั่วโมง (3 วัน)
		})

		// เซ็นลายเซ็น Token ด้วย Key ลับ (jwtKey ต้องถูกประกาศไว้ใน package นี้แล้ว)
		t, _ := token.SignedString(jwtKey)

		// ส่ง JSON กลับไปหา Frontend (ประกอบด้วย Token, UserID, Username)
		json.NewEncoder(w).Encode(map[string]interface{}{"token": t, "user_id": users[0]["id"], "username": users[0]["username"]})
		return
	}

	// ถ้าไม่เจอ User หรือรหัสผิด ส่ง Error 401 (Unauthorized)
	http.Error(w, "Unauthorized", 401)
}

// HandleGetAllUsers ดึงรายชื่อผู้ใช้ทั้งหมด
func HandleGetAllUsers(w http.ResponseWriter, r *http.Request) {
	// จัดการ CORS
	if utils.EnableCORS(&w, r) {
		return
	}

	// เชื่อมต่อ Supabase
	client, _ := supabase.NewClient(os.Getenv("SUPABASE_URL"), os.Getenv("SUPABASE_KEY"), nil)

	var users []map[string]interface{} // ตัวแปรเก็บรายการ User
	// ดึงข้อมูล User ทั้งหมด แต่เลือกเฉพาะ field ที่จำเป็น (ไม่ดึง password)
	client.From("users").Select("id, username, avatar_url, description, gender", "exact", false).ExecuteTo(&users)

	// ตั้งค่า Header ว่าตอบกลับเป็น JSON
	w.Header().Set("Content-Type", "application/json")
	// ส่งข้อมูล JSON กลับไป
	json.NewEncoder(w).Encode(users)
}

// HandleUpdateProfile อัปเดตข้อมูลส่วนตัว
func HandleUpdateProfile(w http.ResponseWriter, r *http.Request) {
	// จัดการ CORS
	if utils.EnableCORS(&w, r) {
		return
	}

	// ประกาศโครงสร้างรับข้อมูลที่ส่งมาจากหน้าแก้ไขโปรไฟล์ (รวมถึง confirm_password)
	var body struct {
		ID              string `json:"id"`
		Username        string `json:"username"`
		Description     string `json:"description"`
		Gender          string `json:"gender"`
		AvatarURL       string `json:"avatar_url"`
		ConfirmPassword string `json:"confirm_password"`
	}
	json.NewDecoder(r.Body).Decode(&body) // แปลง JSON ใส่ตัวแปร body

	// เชื่อมต่อ Supabase
	client, _ := supabase.NewClient(os.Getenv("SUPABASE_URL"), os.Getenv("SUPABASE_KEY"), nil)

	var users []map[string]interface{} // ตัวแปรเก็บข้อมูล User ปัจจุบัน
	// ดึงข้อมูล User ปัจจุบันจาก DB เพื่อมาตรวจสอบรหัสผ่าน
	client.From("users").Select("*", "exact", false).Eq("id", body.ID).ExecuteTo(&users)

	// ตรวจสอบความปลอดภัย: ถ้ามีการเปลี่ยนชื่อ Username ต้องเช็ครหัสผ่านยืนยันเสมอ
	if len(users) > 0 && body.Username != users[0]["username"].(string) {
		// เช็คว่า confirm_password ที่ส่งมา ตรงกับ password hash ในฐานข้อมูลไหม
		if err := bcrypt.CompareHashAndPassword([]byte(users[0]["password"].(string)), []byte(body.ConfirmPassword)); err != nil {
			// ถ้ารหัสผิด ส่ง Error 401
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
	}

	// เตรียมข้อมูลที่จะอัปเดต
	updateData := map[string]interface{}{"username": body.Username, "description": body.Description, "gender": body.Gender, "avatar_url": body.AvatarURL}
	// สั่ง Update ลงฐานข้อมูล where id = body.ID
	client.From("users").Update(updateData, "", "").Eq("id", body.ID).Execute()

	// ส่ง Status 200 OK กลับไปบอกว่าสำเร็จ
	w.WriteHeader(http.StatusOK)
}
