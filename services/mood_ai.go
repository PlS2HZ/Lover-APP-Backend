package services // ประกาศ Package services

import (
	"fmt"  // นำเข้า package fmt เพื่อใช้ฟังก์ชัน Sprintf (จัดรูปแบบข้อความ)
	_ "os" // (Blank Identifier) นำเข้า os ไว้เผื่อใช้ แต่ในฟังก์ชันนี้ไม่ได้เรียกใช้ตรงๆ (ตามโค้ดเดิม)
)

// GetMoodInsight: วิเคราะห์อารมณ์จากประวัติย้อนหลังเพื่อให้คำแนะนำที่ตรงจุด
// รับค่า: targetName (ชื่อของคนที่เราอยากเอาใจ), historyText (ประวัติอารมณ์ที่แปลงเป็น Text แล้ว)
func GetMoodInsight(targetName string, historyText string) (string, error) {
	// สร้าง Prompt (คำสั่ง) ที่จะส่งให้ AI
	// โดยการแทรกชื่อ (targetName) และประวัติอารมณ์ (historyText) ลงไปในข้อความ
	prompt := fmt.Sprintf(`ในฐานะผู้เชี่ยวชาญด้านความสัมพันธ์ 
นี่คือบันทึกอารมณ์ย้อนหลังของ "%s" ในช่วง 3 วันที่ผ่านมา:
%s

จงวิเคราะห์ว่าตอนนี้เขารู้สึกอย่างไร และแนะนำวิธีที่ฉันควรจะปฏิบัติต่อเขาเพื่อให้เขารู้สึกประทับใจ
(ตอบสั้นๆ ไม่เกิน 2 ประโยค เน้นความอบอุ่นและจริงใจ)`, targetName, historyText)

	// เรียกใช้ฟังก์ชัน AskGroqHint (ที่อยู่ในไฟล์ groq.go)
	// ซึ่งฟังก์ชันนี้เหมาะกับการขอคำแนะนำสั้นๆ (Token น้อย)
	insight := AskGroqHint(prompt)

	// ส่งคืนคำแนะนำ (Insight) ที่ได้จาก AI กลับไป
	return insight, nil
}
