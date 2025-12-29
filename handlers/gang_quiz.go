package handlers

import (
	"couple-app/services"
	"couple-app/utils"
	"encoding/json"
	"fmt"
	"net/http"
)

func HandleGetGangQuiz(w http.ResponseWriter, r *http.Request) {
	if utils.EnableCORS(&w, r) {
		return
	}

	category := r.URL.Query().Get("category")
	exclude := r.URL.Query().Get("exclude") // ✅ รับรายการคำถามที่เคยเล่นไปแล้ว

	// pormpt เก่า
	// 	prompt := fmt.Sprintf(`ในฐานะมหาปราชญ์ผู้รอบรู้ จงสร้างคำถามเกี่ยวกับ "%s"
	// โดยมีเงื่อนไขเพิ่มเติม: **ห้ามสร้างคำถามที่ซ้ำหรือใกล้เคียงกับรายการต่อไปนี้เด็ดขาด: [%s]**
	// เน้นความแปลกใหม่และไม่เคยเห็นมาก่อนในระดับสากล
	// Return JSON ONLY: {"question": string, "options": [4 strings], "answer_index": 0-3, "sweet_comment": string}`, category, exclude)

	// Prompt ใหม่
	prompt := fmt.Sprintf(`ในฐานะมหาปราชญ์ผู้รอบรู้ จงสร้างคำถามเกี่ยวกับ "%s" 
โดยมีเงื่อนไข:
1. ความยาวโจทย์: ต้องสั้นและกระชับ (ไม่เกิน 1-2 บรรทัด) แต่อ่านแล้วน่าทึ่ง
2. บริบท: สากล (Global Context) ไม่จำกัดแค่ในไทย
3. เน้นความแปลกใหม่และไม่เคยเห็นมาก่อนในระดับสากล
4. ความถูกต้อง: แม่นยำ 100%% ห้ามมีข้อก้ำกึ่ง
5. ห้ามสร้างคำถามที่ซ้ำหรือใกล้เคียงกับรายการต่อไปนี้เด็ดขาด: [%s]
Return JSON ONLY: {"question": string, "options": [4 strings], "answer_index": 0-3, "sweet_comment": string}`, category, exclude)

	quiz, err := services.GenerateGangQuiz(prompt)
	if err != nil {
		http.Error(w, "AI Error: "+err.Error(), 500)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(quiz)
}
