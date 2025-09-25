package handlers

import (
	"context"
	"fmt"
	"net/http"
	"net/smtp"
	"strconv"
	"strings"
	"ticket-sys/internal/models"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

// Create ticket
// @Summary Create New Ticket
// @Description Create a new ticket
// @ID create-ticket
// @Produce json
// @Success 200 "Successful response"
// @Failure 400 "Bad request"
// @Router /tickets [post]
// @Security Bearer
func (h *AuthHandler) CreateTicket(c *gin.Context) {
	var ticket models.TicketCreate

	// Validate input JSON
	if err := c.ShouldBindJSON(&ticket); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid input format",
			"details": err.Error(),
		})
		return
	}

	// Insert user with transaction
	tx, err := h.db.Begin(context.Background())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction start failed"})
		return
	}

	//layout := "2006-01-02"
	loc, _ := time.LoadLocation("Asia/Tokyo")
	now := time.Now().In(loc)
	// 	parsedDate, err := time.Parse(layout, now.Format("2006-01-02"))
	//  if err != nil {
	//   fmt.Println("Error:", err)
	//  }

	var id int
	err = tx.QueryRow(context.Background(), `
        INSERT INTO ticket 
        VALUES (DEFAULT, $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14) 
        RETURNING id`,
		ticket.ReportedBy,
		ticket.AccommodationName,
		ticket.AccommodationRoomNumber,
		ticket.AccommodationSpecificLocation,
		ticket.AccommodationType,
		ticket.RequestType,
		ticket.RequestDetail,
		"Assigned",
		ticket.TaskPriority,
		0,
		ticket.AssignedTo,
		ticket.Note,
		ticket.Image,
		now,
	).Scan(&id)

	if err != nil {
		tx.Rollback(context.Background())
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ticket creation failed"})
		return
	}

	if err = tx.Commit(context.Background()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failed"})
		return
	}

	var email string
	var firstName string
	var lastName string
	dbErr := h.db.QueryRow(context.Background(), `
        SELECT first_name, last_name, email 
        FROM staff_user 
        WHERE id = $1`,
		ticket.AssignedTo,
	).Scan(
		&firstName,
		&lastName,
		&email)

	if dbErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	var accomm_room_no string
	if ticket.AccommodationRoomNumber == 0 {
		accomm_room_no = ""
	} else {
		accomm_room_no = strconv.Itoa(ticket.AccommodationRoomNumber)
	}

	body := `
		<html>
		<body>
			<h1 style="font-weight:700;">A ticket has been assigned to you</h1>
			<p>Please see ticket information or visit the link below.</p>
			<ul style="list-style-type:none;">
			<li style="padding-bottom:5px;">
					<h2 style="font-weight:700;padding-bottom:0px;">#` + strconv.Itoa(id) + ` ` + ticket.RequestDetail + `</h2>` +
		`</li>
				<li style="padding-bottom:20px;">
					Reported By: ` + ticket.ReportedBy +
		`</li>
				<li style="padding-bottom:20px;">
					Assigned To: ` + firstName + ` ` + lastName +
		`</li>
				<li style="padding-bottom:20px;">
					Accommodation Name: ` + ticket.AccommodationName +
		`</li>
				<li style="padding-bottom:20px;">
					Accommodation Room Number: ` + accomm_room_no +
		`</li>
				<li style="padding-bottom:20px;">
					Specific Location: ` + ticket.AccommodationSpecificLocation +
		`</li>
				<li style="padding-bottom:20px;">
					Accommodation Type: ` + ticket.AccommodationType +
		`</li>
				<li style="padding-bottom:20px;">
					Requst Type: ` + ticket.RequestType +
		`</li>
				<li style="padding-bottom:20px;">
					Task Priority: ` + ticket.TaskPriority +
		`</li>
				<li style="padding-bottom:30px;">
					Notes: ` + ticket.Note +
		`</li>
			</ul>
			<a href="http://192.168.1.57:9000/dashboard">Go To Ticketing Management System</a>
	`

	server := "smtp.office365.com:587"
	sender := "rmdejesus@skijapan.com"
	password := "Rmdj1q2w3e!"
	subject := "Ticket #" + strconv.Itoa(id) + " Has Been Assigned To You"
	message := body
	to := []string{email}

	auth := Auth(sender, password)

	mailBody := []byte("Subject:" + subject + "\r\nMIME-version: 1.0;\r\nContent-Type: text/html; charset=\"UTF-8\";\r\n\r\n" + message)
	mailerErr := smtp.SendMail(server, auth, sender, to, mailBody)
	if mailerErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Mailer error"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":   "Ticket registered successfully",
		"ticket_id": id,
	})
}

// Get all tickets
// @Summary Get All Tickets
// @Description List all tickets available
// @ID get-tickets
// @Produce json
// @Success 200 "Successful response"
// @Failure 500 "Database error"
// @Router /tickets [get]
// @Security Bearer
func (h *AuthHandler) GetTickets(c *gin.Context) {
	var tickets []models.Ticket

	rows, err := h.db.Query(context.Background(), "SELECT * FROM ticket ORDER BY creation_date DESC")

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	for rows.Next() {
		var ticket models.Ticket

		err := rows.Scan(
			&ticket.ID,
			&ticket.ReportedBy,
			&ticket.AccommodationName,
			&ticket.AccommodationRoomNumber,
			&ticket.AccommodationSpecificLocation,
			&ticket.AccommodationType,
			&ticket.RequestType,
			&ticket.RequestDetail,
			&ticket.TaskStatus,
			&ticket.TaskPriority,
			&ticket.AlertLevel,
			&ticket.AssignedTo,
			&ticket.Note,
			&ticket.Image,
			&ticket.CreatedDate,
			&ticket.CompletionDate)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Process failed"})
			return
		}

		//layout := "2025-09-17T16:08:10.468588+09:00"
		t, parseErr := time.Parse(time.RFC3339Nano, ticket.CreatedDate)
		if parseErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Process failed"})
			return
		}
		ticket.CreatedDate = t.Format("2006-01-02 15:04:05")

		tickets = append(tickets, ticket)
	}

	if err = rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ticket iteration failed"})
		return
	}

	// Return ticket list
	c.JSON(http.StatusOK, tickets)
}

// Get single tickets
// @Summary Get a Ticket
// @Description Get single ticket
// @ID get-ticket
// @Produce json
// @Param id path int true "Ticket ID"
// @Success 200 "Successful response"
// @Failure 400 "Invalid ticket ID"
// @Failure 500 "Database error"
// @Router /tickets/{id} [get]
// @Security Bearer
func (h *AuthHandler) GetTicket(c *gin.Context) {
	var ticket models.Ticket

	idStr := c.Param("id")         // Retrieve the path parameter "id" as a string
	id, err := strconv.Atoi(idStr) // Convert it to an integer

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ticket ID"})
		return
	}

	dbErr := h.db.QueryRow(context.Background(), `
        SELECT * 
        FROM ticket 
        WHERE id = $1`,
		id,
	).Scan(
		&ticket.ID,
		&ticket.ReportedBy,
		&ticket.AccommodationName,
		&ticket.AccommodationRoomNumber,
		&ticket.AccommodationSpecificLocation,
		&ticket.AccommodationType,
		&ticket.RequestType,
		&ticket.RequestDetail,
		&ticket.TaskStatus,
		&ticket.TaskPriority,
		&ticket.AlertLevel,
		&ticket.AssignedTo,
		&ticket.Note,
		&ticket.Image,
		&ticket.CreatedDate,
		&ticket.CompletionDate)

	if dbErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	// Return ticket list
	c.JSON(http.StatusOK, ticket)
}

// Update a ticket
// @Summary Update a Ticket
// @Description Update single ticket
// @ID update-ticket
// @Produce json
// @Param id path int true "Ticket ID"
// @Success 200 "Successful response"
// @Failure 400 "Invalid ticket ID"
// @Failure 404 "Ticket not found"
// @Failure 500 "Database error"
// @Router /tickets/{id} [patch]
// @Security Bearer
func (h *AuthHandler) UpdateTicket(c *gin.Context) {
	var ticketUpdate models.TicketUpdate

	idStr := c.Param("id")         // Retrieve the path parameter "id" as a string
	id, err := strconv.Atoi(idStr) // Convert it to an integer

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ticket ID"})
		return
	}

	if dataErr := c.ShouldBindJSON(&ticketUpdate); dataErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ticket data"})
		return
	}

	var exists bool
	databaseErr := h.db.QueryRow(context.Background(), "SELECT EXISTS(SELECT 1 FROM ticket WHERE id = $1)",
		id).Scan(&exists)
	if databaseErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Ticket not found"})
		return
	}

	result := patchUserRecord(h.db, ticketUpdate, id)
	if result == http.StatusNotFound {
		c.JSON(http.StatusNotFound, gin.H{"error": "No data to update for ticket"})
		return
	}
	if result == http.StatusBadRequest {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Database error"})
		return
	}

	// Verify the update
	var updatedTicket models.Ticket
	dbErr := h.db.QueryRow(context.Background(), `
        SELECT * 
        FROM ticket 
        WHERE id = $1`,
		id,
	).Scan(
		&updatedTicket.ID,
		&updatedTicket.ReportedBy,
		&updatedTicket.AccommodationName,
		&updatedTicket.AccommodationRoomNumber,
		&updatedTicket.AccommodationSpecificLocation,
		&updatedTicket.AccommodationType,
		&updatedTicket.RequestType,
		&updatedTicket.RequestDetail,
		&updatedTicket.TaskStatus,
		&updatedTicket.TaskPriority,
		&updatedTicket.AlertLevel,
		&updatedTicket.AssignedTo,
		&updatedTicket.Note,
		&updatedTicket.Image,
		&updatedTicket.CreatedDate,
		&updatedTicket.CompletionDate)

	if dbErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error getting updated ticket"})
		return
	}

	var email string
	var firstName string
	var lastName string
	userErr := h.db.QueryRow(context.Background(), `
        SELECT first_name, last_name, email 
        FROM staff_user 
        WHERE id = $1`,
		updatedTicket.AssignedTo,
	).Scan(
		&firstName,
		&lastName,
		&email)

	if userErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	var accomm_room_no string
	if updatedTicket.AccommodationRoomNumber == 0 {
		accomm_room_no = ""
	} else {
		accomm_room_no = strconv.Itoa(updatedTicket.AccommodationRoomNumber)
	}

	body := `
		<html>
		<body>
			<h1 style="font-weight:700;">A ticket has been updated</h1>
			<p>Please see ticket information or visit the link below.</p>
			<ul style="list-style-type:none;">
			<li style="padding-bottom:5px;">
					<h2 style="font-weight:700;padding-bottom:0px;">#` + strconv.Itoa(updatedTicket.ID) + ` ` + updatedTicket.RequestDetail + `</h2>` +
		`</li>
				<li style="padding-bottom:20px;">
					Reported By: ` + updatedTicket.ReportedBy +
		`</li>
				<li style="padding-bottom:20px;">
					Assigned To: ` + firstName + ` ` + lastName +
		`</li>
				<li style="padding-bottom:20px;">
					Accommodation Name: ` + updatedTicket.AccommodationName +
		`</li>
				<li style="padding-bottom:20px;">
					Accommodation Room Number: ` + accomm_room_no +
		`</li>
				<li style="padding-bottom:20px;">
					Specific Location: ` + updatedTicket.AccommodationSpecificLocation +
		`</li>
				<li style="padding-bottom:20px;">
					Accommodation Type: ` + updatedTicket.AccommodationType +
		`</li>
				<li style="padding-bottom:20px;">
					Requst Type: ` + updatedTicket.RequestType +
		`</li>
				<li style="padding-bottom:20px;">
					Task Priority: ` + updatedTicket.TaskPriority +
		`</li>
				<li style="padding-bottom:30px;">
					Notes: ` + updatedTicket.Note +
		`</li>
			</ul>
			<a href="http://192.168.1.57:9000/dashboard">Go To Ticketing Management System</a>
	`

	server := "smtp.office365.com:587"
	sender := "rmdejesus@skijapan.com"
	password := "Rmdj1q2w3e!"
	subject := "Ticket #" + strconv.Itoa(updatedTicket.ID) + " Updated"
	message := body
	to := []string{email}

	auth := Auth(sender, password)

	mailBody := []byte("Subject:" + subject + "\r\nMIME-version: 1.0;\r\nContent-Type: text/html; charset=\"UTF-8\";\r\n\r\n" + message)
	mailerErr := smtp.SendMail(server, auth, sender, to, mailBody)
	if mailerErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Mailer error"})
		return
	}

	// Return ticket list
	c.JSON(result, updatedTicket)
}

// Update a ticket status to pending
// @Summary Update a Ticket status to pending
// @Description Update ticket status to pending
// @ID update-ticket-pending
// @Produce json
// @Param id path int true "Ticket ID"
// @Success 200 "Successful response"
// @Failure 400 "Invalid ticket ID"
// @Failure 404 "Ticket not found"
// @Failure 500 "Database error"
// @Router /tickets/{id}/pending [patch]
// @Security Bearer
func (h *AuthHandler) UpdatePendingTicket(c *gin.Context) {
	idStr := c.Param("id")         // Retrieve the path parameter "id" as a string
	id, err := strconv.Atoi(idStr) // Convert it to an integer

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ticket ID"})
		return
	}

	var exists bool
	databaseErr := h.db.QueryRow(context.Background(), "SELECT EXISTS(SELECT 1 FROM ticket WHERE id = $1)",
		id).Scan(&exists)
	if databaseErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Ticket not found"})
		return
	}

	tx, err := h.db.Begin(context.Background())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction start failed"})
		return
	}

	var updatedTicket models.Ticket
	err = tx.QueryRow(context.Background(), `
        UPDATE ticket 
        SET task_status = 'Pending'
				WHERE id = $1
        RETURNING *`, id).Scan(&updatedTicket.ID,
		&updatedTicket.ReportedBy,
		&updatedTicket.AccommodationName,
		&updatedTicket.AccommodationRoomNumber,
		&updatedTicket.AccommodationSpecificLocation,
		&updatedTicket.AccommodationType,
		&updatedTicket.RequestType,
		&updatedTicket.RequestDetail,
		&updatedTicket.TaskStatus,
		&updatedTicket.TaskPriority,
		&updatedTicket.AlertLevel,
		&updatedTicket.AssignedTo,
		&updatedTicket.Note,
		&updatedTicket.Image,
		&updatedTicket.CreatedDate,
		&updatedTicket.CompletionDate)

	if err != nil {
		tx.Rollback(context.Background())
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ticket creation failed"})
		return
	}

	if err = tx.Commit(context.Background()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failed"})
		return
	}

	var email string
	var firstName string
	var lastName string
	userErr := h.db.QueryRow(context.Background(), `
        SELECT first_name, last_name, email 
        FROM staff_user 
        WHERE id = $1`,
		updatedTicket.AssignedTo,
	).Scan(
		&firstName,
		&lastName,
		&email)

	if userErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	var accomm_room_no string
	if updatedTicket.AccommodationRoomNumber == 0 {
		accomm_room_no = ""
	} else {
		accomm_room_no = strconv.Itoa(updatedTicket.AccommodationRoomNumber)
	}

	body := `
		<html>
		<body>
			<h1 style="font-weight:700;">A ticket status has been updated to pending</h1>
			<p>Please see ticket information or visit the link below.</p>
			<ul style="list-style-type:none;">
			<li style="padding-bottom:5px;">
					<h2 style="font-weight:700;padding-bottom:0px;">#` + strconv.Itoa(updatedTicket.ID) + ` ` + updatedTicket.RequestDetail + `</h2>` +
		`</li>
				<li style="padding-bottom:20px;">
					Reported By: ` + updatedTicket.ReportedBy +
		`</li>
				<li style="padding-bottom:20px;">
					Assigned To: ` + firstName + ` ` + lastName +
		`</li>
				<li style="padding-bottom:20px;">
					Accommodation Name: ` + updatedTicket.AccommodationName +
		`</li>
				<li style="padding-bottom:20px;">
					Accommodation Room Number: ` + accomm_room_no +
		`</li>
				<li style="padding-bottom:20px;">
					Specific Location: ` + updatedTicket.AccommodationSpecificLocation +
		`</li>
				<li style="padding-bottom:20px;">
					Accommodation Type: ` + updatedTicket.AccommodationType +
		`</li>
				<li style="padding-bottom:20px;">
					Requst Type: ` + updatedTicket.RequestType +
		`</li>
				<li style="padding-bottom:20px;">
					Task Priority: ` + updatedTicket.TaskPriority +
		`</li>
				<li style="padding-bottom:30px;">
					Notes: ` + updatedTicket.Note +
		`</li>
			</ul>
			<a href="http://192.168.1.57:9000/dashboard">Go To Ticketing Management System</a>
	`

	server := "smtp.office365.com:587"
	sender := "rmdejesus@skijapan.com"
	password := "Rmdj1q2w3e!"
	subject := "Ticket #" + strconv.Itoa(updatedTicket.ID) + " Updated To Pending"
	message := body
	to := []string{email}

	auth := Auth(sender, password)

	mailBody := []byte("Subject:" + subject + "\r\nMIME-version: 1.0;\r\nContent-Type: text/html; charset=\"UTF-8\";\r\n\r\n" + message)
	mailerErr := smtp.SendMail(server, auth, sender, to, mailBody)
	if mailerErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Mailer error"})
		return
	}

	// Return ticket list
	c.JSON(http.StatusOK, "")
}

// Update a ticket status to completed
// @Summary Update a Ticket status to completed
// @Description Update ticket status to completed
// @ID update-ticket-completed
// @Produce json
// @Param id path int true "Ticket ID"
// @Success 200 "Successful response"
// @Failure 400 "Invalid ticket ID"
// @Failure 404 "Ticket not found"
// @Failure 500 "Database error"
// @Router /tickets/{id}/completed [patch]
// @Security Bearer
func (h *AuthHandler) UpdateCompletedTicket(c *gin.Context) {
	idStr := c.Param("id")         // Retrieve the path parameter "id" as a string
	id, err := strconv.Atoi(idStr) // Convert it to an integer

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ticket ID"})
		return
	}

	var exists bool
	databaseErr := h.db.QueryRow(context.Background(), "SELECT EXISTS(SELECT 1 FROM ticket WHERE id = $1)",
		id).Scan(&exists)
	if databaseErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Ticket not found"})
		return
	}

	tx, err := h.db.Begin(context.Background())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction start failed"})
		return
	}

	var updatedTicket models.Ticket
	err = tx.QueryRow(context.Background(), `
        UPDATE ticket 
        SET task_status = 'Completed'
				WHERE id = $1
        RETURNING *`, id).Scan(&updatedTicket.ID,
		&updatedTicket.ReportedBy,
		&updatedTicket.AccommodationName,
		&updatedTicket.AccommodationRoomNumber,
		&updatedTicket.AccommodationSpecificLocation,
		&updatedTicket.AccommodationType,
		&updatedTicket.RequestType,
		&updatedTicket.RequestDetail,
		&updatedTicket.TaskStatus,
		&updatedTicket.TaskPriority,
		&updatedTicket.AlertLevel,
		&updatedTicket.AssignedTo,
		&updatedTicket.Note,
		&updatedTicket.Image,
		&updatedTicket.CreatedDate,
		&updatedTicket.CompletionDate)

	if err != nil {
		tx.Rollback(context.Background())
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ticket creation failed"})
		return
	}

	if err = tx.Commit(context.Background()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction commit failed"})
		return
	}

	var email string
	var firstName string
	var lastName string
	userErr := h.db.QueryRow(context.Background(), `
        SELECT first_name, last_name, email 
        FROM staff_user 
        WHERE id = $1`,
		updatedTicket.AssignedTo,
	).Scan(
		&firstName,
		&lastName,
		&email)

	if userErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	var accomm_room_no string
	if updatedTicket.AccommodationRoomNumber == 0 {
		accomm_room_no = ""
	} else {
		accomm_room_no = strconv.Itoa(updatedTicket.AccommodationRoomNumber)
	}

	body := `
		<html>
		<body>
			<h1 style="font-weight:700;">A ticket status has been updated to completed</h1>
			<p>Please see ticket information or visit the link below.</p>
			<ul style="list-style-type:none;">
			<li style="padding-bottom:5px;">
					<h2 style="font-weight:700;padding-bottom:0px;">#` + strconv.Itoa(updatedTicket.ID) + ` ` + updatedTicket.RequestDetail + `</h2>` +
		`</li>
				<li style="padding-bottom:20px;">
					Reported By: ` + updatedTicket.ReportedBy +
		`</li>
				<li style="padding-bottom:20px;">
					Assigned To: ` + firstName + ` ` + lastName +
		`</li>
				<li style="padding-bottom:20px;">
					Accommodation Name: ` + updatedTicket.AccommodationName +
		`</li>
				<li style="padding-bottom:20px;">
					Accommodation Room Number: ` + accomm_room_no +
		`</li>
				<li style="padding-bottom:20px;">
					Specific Location: ` + updatedTicket.AccommodationSpecificLocation +
		`</li>
				<li style="padding-bottom:20px;">
					Accommodation Type: ` + updatedTicket.AccommodationType +
		`</li>
				<li style="padding-bottom:20px;">
					Requst Type: ` + updatedTicket.RequestType +
		`</li>
				<li style="padding-bottom:20px;">
					Task Priority: ` + updatedTicket.TaskPriority +
		`</li>
				<li style="padding-bottom:30px;">
					Notes: ` + updatedTicket.Note +
		`</li>
			</ul>
			<a href="http://192.168.1.57:9000/dashboard">Go To Ticketing Management System</a>
	`

	server := "smtp.office365.com:587"
	sender := "rmdejesus@skijapan.com"
	password := "Rmdj1q2w3e!"
	subject := "Ticket #" + strconv.Itoa(updatedTicket.ID) + " Updated To Completed"
	message := body
	to := []string{email}

	auth := Auth(sender, password)

	mailBody := []byte("Subject:" + subject + "\r\nMIME-version: 1.0;\r\nContent-Type: text/html; charset=\"UTF-8\";\r\n\r\n" + message)
	mailerErr := smtp.SendMail(server, auth, sender, to, mailBody)
	if mailerErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Mailer error"})
		return
	}

	// Return ticket list
	c.JSON(http.StatusOK, "")
}

// Delete a ticket
// @Summary Delete a Ticket
// @Description Delete single ticket
// @ID delete-ticket
// @Produce json
// @Param id path int true "Ticket ID"
// @Success 200 "Successful response"
// @Failure 400 "Invalid ticket ID"
// @Router /tickets/{id} [delete]
// @Security Bearer
func (h *AuthHandler) DeleteTicket(c *gin.Context) {
	idStr := c.Param("id")         // Retrieve the path parameter "id" as a string
	id, err := strconv.Atoi(idStr) // Convert it to an integer

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ticket ID"})
		return
	}

	result, dbErr := h.db.Exec(context.Background(), "DELETE FROM ticket WHERE id = $1", id)
	if dbErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Database error"})
		return
	}
	// rowsAffected, dbErr := result.RowsAffected()
	// if dbErr != nil {
	// 	c.JSON(http.StatusBadRequest, gin.H{"error": "Database error"})
	// 	return
	// }

	if result.RowsAffected() == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No record deleted"})
		return
	}

	// Return ticket list
	c.JSON(http.StatusOK, gin.H{"id": id})
}

func patchUserRecord(db *pgx.Conn, ticketUpdate models.TicketUpdate, id int) int {
	setClauses := []string{}

	if ticketUpdate.ReportedBy != "" {
		setClauses = append(setClauses, fmt.Sprintf("reported_by = %s", "'"+ticketUpdate.ReportedBy+"'"))
	}
	if ticketUpdate.AccommodationName != "" {
		setClauses = append(setClauses, fmt.Sprintf("accommodation_name = %s", "'"+ticketUpdate.AccommodationName+"'"))
	}
	if ticketUpdate.AccommodationRoomNumber != -1 {
		setClauses = append(setClauses, fmt.Sprintf("accommodation_room_number = %d", ticketUpdate.AccommodationRoomNumber))
	}
	if ticketUpdate.AccommodationSpecificLocation != "" {
		setClauses = append(setClauses, fmt.Sprintf("accommodation_specific_location = %s", "'"+ticketUpdate.AccommodationSpecificLocation+"'"))
	}
	if ticketUpdate.AccommodationType != "" {
		setClauses = append(setClauses, fmt.Sprintf("accommodation_type = %s", "'"+ticketUpdate.AccommodationType+"'"))
	}
	if ticketUpdate.RequestType != "" {
		setClauses = append(setClauses, fmt.Sprintf("request_type = %s", "'"+ticketUpdate.RequestType+"'"))
	}
	if ticketUpdate.RequestDetail != "" {
		setClauses = append(setClauses, fmt.Sprintf("request_detail = %s", "'"+ticketUpdate.RequestDetail+"'"))
	}
	if ticketUpdate.TaskStatus != "" {
		setClauses = append(setClauses, fmt.Sprintf("task_status = %s", "'"+ticketUpdate.TaskStatus+"'"))
	}
	if ticketUpdate.TaskPriority != "" {
		setClauses = append(setClauses, fmt.Sprintf("task_priority = %s", "'"+ticketUpdate.TaskPriority+"'"))
	}
	if ticketUpdate.AssignedTo != -1 {
		setClauses = append(setClauses, fmt.Sprintf("assigned_to = %d", ticketUpdate.AssignedTo))
	}
	if ticketUpdate.Note != "" {
		setClauses = append(setClauses, fmt.Sprintf("note = %s", "'"+ticketUpdate.Note+"'"))
	}
	if len(ticketUpdate.Image) != 0 {
		setClauses = append(setClauses, fmt.Sprintf("image = %s", ticketUpdate.Image))
	}

	if len(setClauses) == 0 {
		return http.StatusNotFound
	}

	query := fmt.Sprintf("UPDATE ticket SET %s WHERE id = %d", strings.Join(setClauses, ", "), id)

	// Prepare the statement
	// stmtName := "updateTicketByID"
	// _, err := db.Prepare(context.Background(), stmtName, query)
	// if err != nil {
	// 	return http.StatusBadRequest
	// }

	_, err := db.Exec(context.Background(), query)

	// stmt, err := db.Prepare(context.Background(), "updateUserByID", query)
	// if err != nil {
	// 	return http.StatusBadRequest
	// }
	// defer stmt.Close(context.Background())

	// _, err = stmt.Exec()
	if err != nil {
		return http.StatusBadRequest
	}

	return http.StatusOK
}
