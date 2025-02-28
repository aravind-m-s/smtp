package handler

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strconv"

	"gopkg.in/gomail.v2"
)

func Handler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{ "error": "Only POST method allowed" }`, http.StatusMethodNotAllowed)
		return
	}

	// Parse form data without size limit
	err := r.ParseMultipartForm(0)
	if err != nil {
		http.Error(w, `{ "error": "Failed to parse form data" }`, http.StatusBadRequest)
		return
	}

	// Extract fields from the form
	email := r.FormValue("email")
	password := r.FormValue("password")
	recipient := r.FormValue("recipient")
	subject := r.FormValue("subject")
	message := r.FormValue("message")
	alias := r.FormValue("alias")

	if email == "" || password == "" || recipient == "" || subject == "" || message == "" {
		http.Error(w, `{ "error": "Missing required fields" }`, http.StatusBadRequest)
		return
	}

	// Initialize email message
	m := gomail.NewMessage()
	m.SetHeader("From", m.FormatAddress(email, alias))
	m.SetHeader("To", recipient)
	m.SetHeader("Subject", subject)
	m.SetBody("text/html", message)

	// Process file uploads dynamically (matching attachment-<index>)
	attachmentRegex := regexp.MustCompile(`^attachment-(\d+)$`)
	tempFiles := []string{}
	var attachmentKeys []int

	// Extract and sort attachment indices
	for key := range r.MultipartForm.File {
		matches := attachmentRegex.FindStringSubmatch(key)
		if matches != nil {
			index, _ := strconv.Atoi(matches[1])
			attachmentKeys = append(attachmentKeys, index)
		}
	}
	sort.Ints(attachmentKeys) // Ensure correct order

	// Process attachments in order
	for _, index := range attachmentKeys {
		key := fmt.Sprintf("attachment-%d", index)
		fileHeaders := r.MultipartForm.File[key]

		for _, fileHeader := range fileHeaders {
			// Open the uploaded file
			file, err := fileHeader.Open()
			if err != nil {
				http.Error(w, `{ "error": "Failed to read uploaded file" }`, http.StatusInternalServerError)
				return
			}
			defer file.Close()

			// Save file to a temporary location
			tempPath := "/tmp/" + fileHeader.Filename
			tempFiles = append(tempFiles, tempPath)

			tempFile, err := os.Create(tempPath)
			if err != nil {
				http.Error(w, `{ "error": "Failed to save uploaded file" }`, http.StatusInternalServerError)
				return
			}
			defer tempFile.Close()

			// Copy the uploaded file content to the new file
			_, err = io.Copy(tempFile, file)
			if err != nil {
				http.Error(w, `{ "error": "Failed to copy uploaded file" }`, http.StatusInternalServerError)
				return
			}

			// Attach the file to the email
			m.Attach(tempPath)
		}
	}

	// Send email using SMTP
	d := gomail.NewDialer("smtp.gmail.com", 587, email, password)
	d.SSL = false

	if err := d.DialAndSend(m); err != nil {
		http.Error(w, fmt.Sprintf(`{ "error": "Failed to send email", "details": "%s" }`, err.Error()), http.StatusInternalServerError)
		return
	}

	// Cleanup temporary files after sending
	for _, tempPath := range tempFiles {
		_ = os.Remove(tempPath)
	}

	// Respond with success
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{ "success": "Email sent successfully!" }`))
}
