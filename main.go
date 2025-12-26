package main

import (
	"couple-app/handlers"
	_ "couple-app/services"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()

	// ✅ ระบบแจ้งเตือนอัตโนมัติเบื้องหลัง
	go func() {
		// เรียกใช้จาก handlers เพราะเราย้ายฟังก์ชันมาไว้ที่ event_handlers.go แล้ว
		handlers.CheckAndNotify()
		ticker := time.NewTicker(1 * time.Minute)
		for range ticker.C {
			handlers.CheckAndNotify()
		}
	}()

	// --- Auth & Users ---
	http.HandleFunc("/api/register", handlers.HandleRegister)
	http.HandleFunc("/api/login", handlers.HandleLogin)
	http.HandleFunc("/api/users", handlers.HandleGetAllUsers)
	http.HandleFunc("/api/users/update", handlers.HandleUpdateProfile)

	// --- Mood ---
	http.HandleFunc("/api/save-mood", handlers.HandleSaveMood)
	http.HandleFunc("/api/get-moods", handlers.HandleGetMoods)
	http.HandleFunc("/api/mood/delete", handlers.HandleDeleteMood)

	// --- Wishlist ---
	http.HandleFunc("/api/wishlist/save", handlers.HandleSaveWishlist)
	http.HandleFunc("/api/wishlist/get", handlers.HandleGetWishlist)
	http.HandleFunc("/api/wishlist/complete", handlers.HandleCompleteWish)
	http.HandleFunc("/api/wishlist/delete", handlers.HandleDeleteWishlist)

	// --- Moments ---
	http.HandleFunc("/api/moment/save", handlers.HandleSaveMoment)
	http.HandleFunc("/api/moment/get", handlers.HandleGetMoments)
	http.HandleFunc("/api/moment/delete", handlers.HandleDeleteMoment)

	// --- Requests ---
	http.HandleFunc("/api/request", handlers.HandleCreateRequest)
	http.HandleFunc("/api/my-requests", handlers.HandleGetMyRequests)
	http.HandleFunc("/api/update-status", handlers.HandleUpdateStatus)

	// --- Calendar & Events ---
	http.HandleFunc("/api/events", handlers.HandleGetMyEvents)
	http.HandleFunc("/api/events/create", handlers.HandleCreateEvent)
	http.HandleFunc("/api/events/delete", handlers.HandleDeleteEvent)
	http.HandleFunc("/api/highlights", handlers.HandleGetHighlights)

	// --- PWA Push Notifications ---
	http.HandleFunc("/api/save-subscription", handlers.SaveSubscriptionHandler)
	http.HandleFunc("/api/unsubscribe", handlers.HandleUnsubscribe)
	http.HandleFunc("/api/check-subscription", handlers.HandleCheckSubscription)

	// --- Home Config & Games ---
	http.HandleFunc("/api/home-config/get", handlers.HandleGetHomeConfig)
	http.HandleFunc("/api/home-config/update", handlers.HandleUpdateHomeConfig)
	http.HandleFunc("/api/game/start", handlers.HandleStartHeartGame)
	http.HandleFunc("/api/game/ask", handlers.HandleAskQuestion)
	http.HandleFunc("/api/game/create", handlers.HandleCreateGame)
	http.HandleFunc("/api/game/generate-description", handlers.HandleGenerateAIDescription)

	http.HandleFunc("/api/game/bot-auto-create", handlers.HandleBotAutoCreateGame)

	// main.go
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // ใช้ 8080 เฉพาะตอนอยู่บนเครื่องตัวเอง
	}

	log.Printf("🚀 Server live on %s", port)

	// ต้องมั่นใจว่าใช้ nil และไม่มี router ตัวอื่นมาขวาง
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}
