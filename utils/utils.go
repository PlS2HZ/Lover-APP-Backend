package utils // ประกาศ package utils

// LevenshteinDistance: ฟังก์ชันคำนวณระยะห่างของตัวอักษรเพื่อเช็คคำสะกดผิด
// หลักการคือ: การเปลี่ยน s1 เป็น s2 ต้องแก้ไขกี่ครั้ง (ลบ, เพิ่ม, หรือเปลี่ยนตัวอักษร)
// ยิ่งค่า Return น้อย แปลว่ายิ่งเหมือนกันมาก
func LevenshteinDistance(s1, s2 string) int {
	// แปลง String เป็น Rune Slice เพื่อให้รองรับภาษาไทย, สระ, วรรณยุกต์ และ Emoji ได้ถูกต้อง
	// (ถ้าใช้ string ปกติ ภาษาไทย 1 ตัวอาจถูกนับเป็นหลาย byte ทำให้คำนวณผิด)
	r1 := []rune(s1)
	r2 := []rune(s2)
	len1 := len(r1)
	len2 := len(r2)

	// สร้าง Slice เพื่อเก็บค่าคำนวณ (ใช้เทคนิค Dynamic Programming แบบประหยัด Memory)
	// แทนที่จะสร้าง Matrix ใหญ่ๆ ก็สร้างแค่แถวเดียวแล้วอัปเดตค่าเอา
	column := make([]int, len1+1)
	for y := 1; y <= len1; y++ {
		column[y] = y
	}

	// เริ่มวนลูปเปรียบเทียบทีละตัวอักษร
	for x := 1; x <= len2; x++ {
		column[0] = x
		lastkey := x - 1
		for y := 1; y <= len1; y++ {
			oldkey := column[y]
			var incr int

			// เช็คว่าตัวอักษรตำแหน่งนี้เหมือนกันไหม
			if r1[y-1] != r2[x-1] {
				incr = 1 // ถ้าไม่เหมือน ให้เพิ่มค่า Cost (ความต่าง) ขึ้น 1
			}

			// คำนวณหาค่าที่น้อยที่สุดจาก 3 กรณี:
			// 1. column[y]+1   -> กรณีการลบ (Deletion)
			// 2. column[y-1]+1 -> กรณีการเพิ่ม (Insertion)
			// 3. lastkey+incr  -> กรณีการแทนที่ (Substitution) หรือถ้าตัวอักษรเหมือนกัน (Match)
			column[y] = min(column[y]+1, column[y-1]+1, lastkey+incr)
			lastkey = oldkey
		}
	}
	// คืนค่าสุดท้ายในตาราง ซึ่งคือจำนวนความต่างทั้งหมด
	return column[len1]
}

// min: ฟังก์ชัน Helper สำหรับหาค่าน้อยที่สุดจากตัวเลข Integer 3 ตัว
func min(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}
