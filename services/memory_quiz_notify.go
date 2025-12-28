package services

import (
	"fmt"
)

// NotifyQuizSuccess ทำหน้าที่ส่งแจ้งเตือนทั้ง PWA และ Discord แยกมาเพื่อการจัดการที่ง่าย
func NotifyQuizSuccess(partnerID string, question string, wrongCount int) {
	// 1. หัวข้อและเนื้อหาแจ้งเตือน
	title := "💖 แฟนทายใจคุณถูก!"
	body := fmt.Sprintf("แฟนจำเรื่องนี้ได้: %s\n(ผิดไป %d ครั้งกว่าจะถูก)", question, wrongCount)

	// 2. ส่ง Push Notification (PWA)
	TriggerPushNotification(partnerID, title, body)

	// 3. ส่งเข้า Discord (รันแบบ Background เพื่อไม่ให้หน้าเว็บค้าง)
	go func() {
		discordMsg := fmt.Sprintf("✨ **%s**\n💭 **คำถาม:** %s\n❌ **ตอบผิดไป:** %d ครั้ง", title, question, wrongCount)
		// สีชมพู (16738740) สำหรับความรัก
		SendDiscordEmbed("Memory Quiz Success! ❤️", discordMsg, 16738740, nil, "")
	}()
}
