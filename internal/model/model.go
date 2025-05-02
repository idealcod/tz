package model

// PersonInput represents the input data for creating or updating a person.
type PersonInput struct {
	Name        string `json:"name" binding:"required"`    // Required field: person's name
	Surname     string `json:"surname" binding:"required"` // Required field: person's surname
	Patronymic  string `json:"patronymic"`                 // Optional field: person's patronymic
	Age         int    `json:"age"`                        // Optional field: person's age, enriched if not provided
	Gender      string `json:"gender"`                     // Optional field: person's gender, enriched if not provided
	Nationality string `json:"nationality"`                // Optional field: person's nationality, enriched if not provided
}

// Person represents a person entity stored in the database.
type Person struct {
	ID          int    `json:"id"`          // Unique identifier of the person
	Name        string `json:"name"`        // Person's name
	Surname     string `json:"surname"`     // Person's surname
	Patronymic  string `json:"patronymic"`  // Person's patronymic
	Age         int    `json:"age"`         // Person's age
	Gender      string `json:"gender"`      // Person's gender
	Nationality string `json:"nationality"` // Person's nationality
}

// Filter represents the filtering and pagination criteria for retrieving people.
type Filter struct {
	Name        string // Filter by name (partial match)
	Surname     string // Filter by surname (partial match)
	Patronymic  string // Filter by patronymic (partial match)
	Age         *int   // Filter by exact age
	Gender      string // Filter by gender
	Nationality string // Filter by nationality
	Page        int    // Pagination: page number
	Limit       int    // Pagination: items per page
}
