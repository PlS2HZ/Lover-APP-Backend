package services

import (
	"fmt"
	_ "os"
)

// GetMoodInsight: วิเคราะห์อารมณ์จากประวัติย้อนหลังเพื่อให้คำแนะนำที่ตรงจุด
func GetMoodInsight(targetName string, historyText string) (string, error) {
	// สร้าง Prompt ที่ส่งประวัติย้อนหลัง 3 วันไปด้วย
	prompt := fmt.Sprintf(`ในฐานะผู้เชี่ยวชาญด้านความสัมพันธ์ 
นี่คือบันทึกอารมณ์ย้อนหลังของ "%s" ในช่วง 3 วันที่ผ่านมา:
%s

จงวิเคราะห์ว่าตอนนี้เขารู้สึกอย่างไร และแนะนำวิธีที่ฉันควรจะปฏิบัติต่อเขาเพื่อให้เขารู้สึกประทับใจ
(ตอบสั้นๆ ไม่เกิน 2 ประโยค เน้นความอบอุ่นและจริงใจ)`, targetName, historyText)

	insight := AskGroqHint(prompt)
	return insight, nil
}
