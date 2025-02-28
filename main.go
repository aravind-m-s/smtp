package main

import (
	"encoding/json"
	"net/http"

	"gopkg.in/gomail.v2"
)

type EmailRequest struct {
	Email     string `json:"email"`
	Password  string `json:"password"`
	Recipient string `json:"recipient"`
	Subject   string `json:"subject"`
	Message   string `json:"message"`
	Alias     string `json:"alias"`
}

func sendEmailHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{ "error": "Only POST method allowed" }`, http.StatusMethodNotAllowed)
		return
	}

	var req EmailRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{ "error": "Invalid request body" }`, http.StatusBadRequest)
		return
	}

	if req.Email == "" || req.Password == "" || req.Recipient == "" || req.Subject == "" || req.Message == "" {
		http.Error(w, `{ "error": "Missing required fields" }`, http.StatusBadRequest)
		return
	}

	m := gomail.NewMessage()
	m.SetHeader("From", m.FormatAddress(req.Email, req.Alias))
	m.SetHeader("To", req.Recipient)
	m.SetHeader("Subject", req.Subject)
	m.SetBody("text/html", req.Message)

	d := gomail.NewDialer("smtp.gmail.com", 587, req.Email, req.Password)
	d.SSL = false

	if err := d.DialAndSend(m); err != nil {
		http.Error(w, `{ "error": "Failed to send email", "details": "`+err.Error()+`" }`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{ "success": "Email sent successfully!" }`))
}

func main() {
	http.HandleFunc("/send-email", sendEmailHandler)
	http.ListenAndServe(":8080", nil)
}
