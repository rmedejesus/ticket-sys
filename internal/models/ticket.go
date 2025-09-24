package models

// User represents our database user
type Ticket struct {
	ID                            int     `json:"id"`
	ReportedBy                    string  `json:"reported_by"`
	AccommodationName             string  `json:"accommodation_name"`
	AccommodationRoomNumber       int     `json:"accommodation_room_number"`
	AccommodationSpecificLocation string  `json:"accommodation_specific_location"`
	AccommodationType             string  `json:"accommodation_type"`
	RequestType                   string  `json:"request_type"`
	RequestDetail                 string  `json:"request_detail"`
	TaskStatus                    string  `json:"task_status"`
	TaskPriority                  string  `json:"task_priority"`
	AlertLevel                    int     `json:"alert_level"`
	AssignedTo                    int     `json:"assigned_to"`
	Note                          string  `json:"note"`
	Image                         []byte  `json:"image"`
	CreatedDate                   string  `json:"created_date"`
	CompletionDate                *string `json:"completion_date"`
}

// UserRegister represents registration request data
type TicketCreate struct {
	ReportedBy                    string `json:"reported_by"`
	AccommodationName             string `json:"accommodation_name,omitempty" binding:"required"`
	AccommodationRoomNumber       int    `json:"accommodation_room_number,string"`
	AccommodationSpecificLocation string `json:"accommodation_specific_location,omitempty" binding:"required"`
	AccommodationType             string `json:"accommodation_type,omitempty" binding:"required"`
	RequestType                   string `json:"request_type,omitempty" binding:"required"`
	RequestDetail                 string `json:"request_detail,omitempty" binding:"required"`
	TaskPriority                  string `json:"task_priority,omitempty" binding:"required"`
	AssignedTo                    int    `json:"assigned_to,string,omitempty" binding:"required"`
	Note                          string `json:"note"`
	Image                         []byte `json:"image"`
}

// UserRegister represents registration request data
type TicketUpdate struct {
	ReportedBy                    string `json:"reported_by"`
	AccommodationName             string `json:"accommodation_name"`
	AccommodationRoomNumber       int    `json:"accommodation_room_number,string"`
	AccommodationSpecificLocation string `json:"accommodation_specific_location"`
	AccommodationType             string `json:"accommodation_type"`
	RequestType                   string `json:"request_type"`
	RequestDetail                 string `json:"request_detail"`
	TaskStatus                    string `json:"task_status"`
	TaskPriority                  string `json:"task_priority"`
	AssignedTo                    int    `json:"assigned_to,string,omitempty"`
	Note                          string `json:"note"`
	Image                         []byte `json:"image"`
}
