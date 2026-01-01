package utils // ประกาศ Package utils (รวมฟังก์ชันอเนกประสงค์)

import (
	"net/http" // นำเข้า package สำหรับจัดการ HTTP Server (Header, Status Code)
	"time"     // นำเข้า package สำหรับจัดการวันและเวลา
)

// EnableCORS: ฟังก์ชันสำหรับตั้งค่า Header เพื่ออนุญาตให้ Frontend (ต่างโดเมน) เรียก API ได้
// คืนค่า bool: true ถ้าเป็น Preflight Request (OPTIONS) เพื่อให้ Handler หลักหยุดทำงาน
//
//	false ถ้าเป็น Request ปกติ (GET, POST, etc.) เพื่อให้ทำงานต่อ
func EnableCORS(w *http.ResponseWriter, r *http.Request) bool {
	// อนุญาตให้ทุกโดเมน (*) เรียกใช้งาน API นี้ได้
	(*w).Header().Set("Access-Control-Allow-Origin", "*")

	// อนุญาตให้ใช้ Method เหล่านี้ในการเรียก
	(*w).Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, PUT, DELETE, OPTIONS")

	// อนุญาตให้ส่ง Header เหล่านี้มาได้ (เช่น Content-Type สำหรับ JSON, Authorization สำหรับ Token)
	(*w).Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	// ตรวจสอบว่าเป็น Preflight Request (Method OPTIONS) หรือไม่
	// (Browser จะส่ง OPTIONS มาเช็คก่อนส่งข้อมูลจริง)
	if r.Method == "OPTIONS" {
		(*w).WriteHeader(http.StatusOK) // ตอบกลับ 200 OK ทันที
		return true                     // บอก Handler หลักว่า "ไม่ต้องทำอะไรต่อแล้วนะ จบแค่นี้"
	}

	return false // บอก Handler หลักว่า "ทำงานต่อได้เลย นี่คือ Request จริง"
}

// FormatDisplayTime: แปลงเวลาจากรูปแบบ RFC3339 (UTC) เป็นเวลาไทยที่อ่านง่าย
func FormatDisplayTime(t string) string {
	// ลองแปลง string เวลาที่รับมา ให้เป็น Time Object มาตรฐาน
	parsedTime, err := time.Parse(time.RFC3339, t)

	// ถ้าแปลงไม่ได้ (Format ผิด) ให้ส่งค่าเดิมกลับไปเลย กันโปรแกรมพัง
	if err != nil {
		return t
	}

	// แปลงเวลาเป็น Timezone Asia/Bangkok (UTC+7)
	// 7 * 3600 วินาที = 7 ชั่วโมง
	thailandTime := parsedTime.In(time.FixedZone("Asia/Bangkok", 7*3600))

	// จัดรูปแบบการแสดงผลเป็น "ปี-เดือน-วัน เวลา ชม:นาที น."
	// (Go ใช้เลขวันเกิดของภาษา Go ในการกำหนด Format: 2006-01-02 15:04:05)
	return thailandTime.Format("2006-01-02 เวลา 15:04 น.")
}
