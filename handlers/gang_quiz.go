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
	if category == "" {
		category = "ความรู้รอบตัว"
	}

	prompt := fmt.Sprintf(`Create a single fun Thai trivia quiz about %s. 
Return exactly one JSON object. Do not put it in a list. 
Structure: {"question": string, "options": [4 strings], "answer_index": 0-3, "sweet_comment": string}`, category)

	quiz, err := services.GenerateQuizFromGroq(prompt)
	if err != nil {
		// ส่ง Error กลับไปให้ Frontend รู้เรื่อง
		http.Error(w, "AI มึนตึ๊บ: "+err.Error(), 500)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(quiz)
}
