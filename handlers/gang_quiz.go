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

	// Prompt ใหม่
	prompt := fmt.Sprintf(`ในฐานะมหาปราชญ์ผู้รอบรู้ จงสร้างคำถามเกี่ยวกับ "%s" 
โดยมีเงื่อนไขเหล็ก:
1. ความยาวโจทย์: ต้องสั้นและกระชับที่สุด (ห้ามเกิน 15 คำไทย) ห้ามบรรยายเยอะ
2. เนื้อหา: ต้องเป็นเกร็ดความรู้ระดับโลกที่ "น่าทึ่ง" (Mind-blowing) แต่จบในประโยคเดียว
3. บริบท: สากล (Global Context) ห้ามถามแค่เรื่องในไทย
4. ความถูกต้อง: ข้อมูลต้องเป็นความจริงสากล 100%%
"5. รายการคำถามที่เคยเล่นไปแล้ว (ห้ามสร้างซ้ำเด็ดขาด): [%s]"
Return JSON ONLY: {"question": string, "options": [4 strings], "answer_index": 0-3, "sweet_comment": string}`, category, exclude)
	quiz, err := services.GenerateGangQuiz(prompt)
	if err != nil {
		http.Error(w, "AI Error: "+err.Error(), 500)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(quiz)
}
