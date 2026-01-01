package main // ประกาศ package main เพื่อบอกว่าเป็นจุดเริ่มต้นของโปรแกรม

import (
	"couple-app/handlers" // นำเข้า package handlers (รวมฟังก์ชัน API ทั้งหมดที่เราเขียนไว้)
	"couple-app/services" // นำเข้า package services (เพื่อใช้ CheckAndNotify สำหรับ Cron Job)
	"fmt"                 // ใช้พิมพ์ข้อความออกทาง Console
	"log"                 // ใช้พิมพ์ Log แบบมี Timestamp (เหมาะสำหรับ Server Log)
	"net/http"            // ใช้สร้าง Web Server และกำหนด Routing
	"os"                  // ใช้สำหรับอ่าน Environment Variable (เช่น PORT)
	"time"                // ใช้สำหรับจัดการเวลาและ Ticker

	"github.com/joho/godotenv" // Library สำหรับโหลดไฟล์ .env (ใช้ตอน Run บนเครื่องตัวเอง)
)

func main() {
	// โหลดค่า Config จากไฟล์ .env (ถ้าไฟล์ไม่มีก็ไม่ Error จะข้ามไป)
	godotenv.Load()

	// ✅ เปิดเครื่องตั้งเวลาเช็คนัดหมายทุก 1 นาที (Background Task)
	// ใช้ go func() เพื่อรันเป็น Thread แยก (Goroutine) ไม่ให้บล็อกการทำงานหลักของ Server
	go func() {
		fmt.Println("⏰ [SYSTEM] Ticker Started: Checking events every minute...")

		// สร้าง Ticker ที่จะส่งสัญญาณทุกๆ 1 นาที
		ticker := time.NewTicker(1 * time.Minute)

		// วนลูปตลอดไป (Infinite Loop) รอรับสัญญาณจาก ticker.C
		for range ticker.C {
			// เมื่อครบ 1 นาที ให้เรียกฟังก์ชันเช็คนัดหมายและแจ้งเตือน
			services.CheckAndNotify()
		}
	}()

	// --- รวบรวม Handler เดิมทั้งหมด ห้ามลบ ---
	// กำหนด URL Path ให้วิ่งไปหาฟังก์ชันที่ถูกต้องใน handlers

	// Auth & Users (จัดการผู้ใช้)
	http.HandleFunc("/api/register", handlers.HandleRegister)          // API สมัครสมาชิก
	http.HandleFunc("/api/login", handlers.HandleLogin)                // API เข้าสู่ระบบ
	http.HandleFunc("/api/users", handlers.HandleGetAllUsers)          // API ดึงรายชื่อผู้ใช้
	http.HandleFunc("/api/users/update", handlers.HandleUpdateProfile) // API แก้ไขโปรไฟล์

	// Mood (บันทึกอารมณ์)
	http.HandleFunc("/api/save-mood", handlers.HandleSaveMood)          // บันทึกอารมณ์ใหม่
	http.HandleFunc("/api/get-moods", handlers.HandleGetMoods)          // ดึง Feed อารมณ์
	http.HandleFunc("/api/mood/delete", handlers.HandleDeleteMood)      // ลบอารมณ์
	http.HandleFunc("/api/mood/insight", handlers.HandleGetMoodInsight) // ให้ AI วิเคราะห์อารมณ์

	// Wishlist (รายการของที่อยากได้)
	http.HandleFunc("/api/wishlist/save", handlers.HandleSaveWishlist)     // บันทึก Wishlist
	http.HandleFunc("/api/wishlist/get", handlers.HandleGetWishlist)       // ดึงรายการ Wishlist
	http.HandleFunc("/api/wishlist/complete", handlers.HandleCompleteWish) // กดสำเร็จ (ได้รับของแล้ว)
	http.HandleFunc("/api/wishlist/delete", handlers.HandleDeleteWishlist) // ลบรายการ

	// Requests (ระบบขออนุมัติ)
	http.HandleFunc("/api/request", handlers.HandleCreateRequest)      // สร้างคำขอใหม่
	http.HandleFunc("/api/my-requests", handlers.HandleGetMyRequests)  // ดูคำขอของฉัน
	http.HandleFunc("/api/update-status", handlers.HandleUpdateStatus) // กดอนุมัติ/ปฏิเสธ

	// Calendar & Events (ปฏิทินนัดหมาย)
	http.HandleFunc("/api/events", handlers.HandleGetMyEvents)        // ดึงข้อมูลปฏิทิน
	http.HandleFunc("/api/events/create", handlers.HandleCreateEvent) // สร้างนัดหมาย
	http.HandleFunc("/api/events/delete", handlers.HandleDeleteEvent) // ลบนัดหมาย
	http.HandleFunc("/api/highlights", handlers.HandleGetHighlights)  // ดึงวันสำคัญ (Highlight)

	// PWA Push Notifications (ระบบแจ้งเตือน)
	http.HandleFunc("/api/save-subscription", handlers.SaveSubscriptionHandler)  // บันทึก Token ของเครื่องผู้ใช้
	http.HandleFunc("/api/unsubscribe", handlers.HandleUnsubscribe)              // ยกเลิกการรับแจ้งเตือน
	http.HandleFunc("/api/check-subscription", handlers.HandleCheckSubscription) // เช็คว่าเครื่องนี้เปิดแจ้งเตือนหรือยัง

	// Home Config & Games (ตั้งค่าหน้าโฮม และเกมทายใจ Mind Game)
	http.HandleFunc("/api/home-config/get", handlers.HandleGetHomeConfig)                   // ดึงการตั้งค่าหน้าแรก
	http.HandleFunc("/api/home-config/update", handlers.HandleUpdateHomeConfig)             // อัปเดตการตั้งค่า
	http.HandleFunc("/api/game/start", handlers.HandleStartHeartGame)                       // เริ่มเกม Mind Game
	http.HandleFunc("/api/game/ask", handlers.HandleAskQuestion)                            // ส่งคำถามทายใจ (AI ตอบ)
	http.HandleFunc("/api/game/create", handlers.HandleCreateGame)                          // สร้างห้องเกม
	http.HandleFunc("/api/game/generate-description", handlers.HandleGenerateAIDescription) // ให้ AI สร้างคำอธิบายคำลับ
	http.HandleFunc("/api/game/bot-auto-create", handlers.HandleBotAutoCreateGame)          // บอทสร้างเกมอัตโนมัติ

	// Memory Quiz (เกมทายความทรงจำ)
	http.HandleFunc("/api/memory-quiz/save", handlers.HandleSaveMemory)           // บันทึกความทรงจำใหม่
	http.HandleFunc("/api/memory-quiz/random", handlers.HandleGetRandomQuiz)      // สุ่มคำถามจากความทรงจำ
	http.HandleFunc("/api/memory-quiz/all", handlers.HandleGetAllMemories)        // ดูความทรงจำทั้งหมด
	http.HandleFunc("/api/memory-quiz/submit", handlers.HandleSubmitQuizResponse) // ส่งผลคะแนน (เพื่อแจ้งเตือนแฟน)
	http.HandleFunc("/api/memory-quiz/delete", handlers.HandleDeleteMemory)       // ลบความทรงจำ

	// Gang Quiz (เกมตอบปัญหาทั่วไป)
	http.HandleFunc("/api/gang-quiz/random", handlers.HandleGetGangQuiz) // สุ่มคำถาม Gang Quiz จาก AI

	// ตั้งค่า Port สำหรับ Server
	port := os.Getenv("PORT")
	if port == "" {
		port = "10000" // ปรับเป็น 10000 เป็นค่า Default (ถ้าไม่ได้รันบน Cloud ที่กำหนด Port มาให้)
	}

	log.Printf("🚀 Server live on %s", port) // แจ้งเตือนว่า Server เริ่มทำงานแล้ว

	// เริ่มรัน Server รอรับ Request
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err) // ถ้า Server ล่มหรือไม่เริ่มทำงาน ให้แสดง Error และปิดโปรแกรม
	}
}
