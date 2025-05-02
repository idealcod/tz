package service

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"

	"effective-mobile.test/internal/model"
)

// Service handles business logic and interacts with the repository.
type Service struct {
	repo   Repository
	logger *logrus.Logger
}

// Repository defines the interface for repository layer methods.
type Repository interface {
	CreatePerson(person *model.Person) (int, error)
	GetPeople(filter model.Filter) ([]model.Person, error)
	DeletePerson(id int) error
	UpdatePerson(id int, person *model.Person) error
}

// NewService creates a new Service instance.
func NewService(repo Repository, logger *logrus.Logger) *Service {
	return &Service{repo: repo, logger: logger}
}

// CreatePerson creates a new person in the database, enriching the data if necessary.
func (s *Service) CreatePerson(input model.PersonInput) (int, error) {
	s.logger.WithFields(logrus.Fields{
		"name":        input.Name,
		"surname":     input.Surname,
		"patronymic":  input.Patronymic,
		"age":         input.Age,
		"gender":      input.Gender,
		"nationality": input.Nationality,
	}).Debug("Creating person with input")

	person := &model.Person{
		Name:       input.Name,
		Surname:    input.Surname,
		Patronymic: input.Patronymic,
	}

	// Enrich data only if Age, Gender, and Nationality are not provided
	enrichedInput := input
	if enrichedInput.Age == 0 && enrichedInput.Gender == "" && enrichedInput.Nationality == "" {
		s.logger.Debug("All enrichment fields are empty, proceeding with enrichment")
		if err := s.enrichPerson(&enrichedInput); err != nil {
			s.logger.Warn("Failed to enrich person data: ", err)
		}
	} else {
		s.logger.WithFields(logrus.Fields{
			"age":         enrichedInput.Age,
			"gender":      enrichedInput.Gender,
			"nationality": enrichedInput.Nationality,
		}).Info("Skipping enrichment as some fields are provided")
	}

	person.Age = enrichedInput.Age
	person.Gender = enrichedInput.Gender
	person.Nationality = enrichedInput.Nationality

	return s.repo.CreatePerson(person)
}

// GetPeople retrieves people based on filters and pagination.
func (s *Service) GetPeople(filter model.Filter) ([]model.Person, error) {
	s.logger.WithFields(logrus.Fields{
		"filter": filter,
	}).Debug("Fetching people with filter")
	return s.repo.GetPeople(filter)
}

// DeletePerson deletes a person by ID.
func (s *Service) DeletePerson(id int) error {
	s.logger.WithFields(logrus.Fields{
		"id": id,
	}).Debug("Deleting person")
	return s.repo.DeletePerson(id)
}

// UpdatePerson updates a person by ID, enriching the data if necessary.
func (s *Service) UpdatePerson(id int, input model.PersonInput) error {
	s.logger.WithFields(logrus.Fields{
		"id":          id,
		"name":        input.Name,
		"surname":     input.Surname,
		"patronymic":  input.Patronymic,
		"age":         input.Age,
		"gender":      input.Gender,
		"nationality": input.Nationality,
	}).Debug("Updating person with input")

	person := &model.Person{
		Name:       input.Name,
		Surname:    input.Surname,
		Patronymic: input.Patronymic,
	}

	// Enrich data only if Age, Gender, and Nationality are not provided
	enrichedInput := input
	if enrichedInput.Age == 0 && enrichedInput.Gender == "" && enrichedInput.Nationality == "" {
		s.logger.Debug("All enrichment fields are empty, proceeding with enrichment")
		if err := s.enrichPerson(&enrichedInput); err != nil {
			s.logger.Warn("Failed to enrich person data: ", err)
		}
	} else {
		s.logger.WithFields(logrus.Fields{
			"age":         enrichedInput.Age,
			"gender":      enrichedInput.Gender,
			"nationality": enrichedInput.Nationality,
		}).Info("Skipping enrichment as some fields are provided")
	}

	person.Age = enrichedInput.Age
	person.Gender = enrichedInput.Gender
	person.Nationality = enrichedInput.Nationality

	return s.repo.UpdatePerson(id, person)
}

// enrichPerson enriches the person's data using external APIs for age, gender, and nationality.
func (s *Service) enrichPerson(person *model.PersonInput) error {
	s.logger.WithFields(logrus.Fields{
		"name": person.Name,
	}).Debug("Starting enrichment for person")

	// Возраст
	ageResp := struct {
		Age int `json:"age"`
	}{}
	if err := s.fetchJSON("https://api.agify.io/?name="+person.Name, &ageResp); err == nil && ageResp.Age != 0 {
		person.Age = ageResp.Age
		s.logger.WithFields(logrus.Fields{
			"name": person.Name,
			"age":  person.Age,
		}).Info("Enriched age")
	} else {
		s.logger.WithFields(logrus.Fields{
			"name":         person.Name,
			"error":        err,
			"age_response": ageResp.Age,
		}).Warn("Failed to enrich age, setting default value")
		person.Age = 0 // Дефолтное значение
	}

	// Пол
	genderResp := struct {
		Gender string `json:"gender"`
	}{}
	if err := s.fetchJSON("https://api.genderize.io/?name="+person.Name, &genderResp); err == nil && genderResp.Gender != "" {
		person.Gender = genderResp.Gender
		s.logger.WithFields(logrus.Fields{
			"name":   person.Name,
			"gender": person.Gender,
		}).Info("Enriched gender")
	} else {
		s.logger.WithFields(logrus.Fields{
			"name":            person.Name,
			"error":           err,
			"gender_response": genderResp.Gender,
		}).Warn("Failed to enrich gender, setting default value")
		person.Gender = "unknown" // Дефолтное значение
	}

	// Национальность
	nationalityResp := struct {
		Country []struct {
			CountryID string `json:"country_id"`
		} `json:"country"`
	}{}
	if err := s.fetchJSON("https://api.nationalize.io/?name="+person.Name, &nationalityResp); err == nil && len(nationalityResp.Country) > 0 && nationalityResp.Country[0].CountryID != "" {
		person.Nationality = nationalityResp.Country[0].CountryID
		s.logger.WithFields(logrus.Fields{
			"name":        person.Name,
			"nationality": person.Nationality,
		}).Info("Enriched nationality")
	} else {
		s.logger.WithFields(logrus.Fields{
			"name":          person.Name,
			"error":         err,
			"country_count": len(nationalityResp.Country),
		}).Warn("Failed to enrich nationality, setting default value")
		person.Nationality = "unknown" // Дефолтное значение
	}

	return nil
}

// fetchJSON fetches and decodes JSON data from an external API with a timeout.
func (s *Service) fetchJSON(url string, target interface{}) error {
	s.logger.WithFields(logrus.Fields{
		"url": url,
	}).Debug("Fetching data from external API")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		s.logger.WithFields(logrus.Fields{
			"url":   url,
			"error": err,
		}).Error("Failed to fetch data from external API")
		return err
	}
	defer resp.Body.Close()

	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		s.logger.WithFields(logrus.Fields{
			"url":   url,
			"error": err,
		}).Error("Failed to decode response from external API")
		return err
	}

	s.logger.WithFields(logrus.Fields{
		"url": url,
	}).Debug("Successfully fetched and decoded data from external API")
	return nil
}
