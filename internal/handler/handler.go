package handler

import (
	"database/sql"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"effective-mobile.test/internal/model"
)

// Service defines the interface for service layer methods.
type Service interface {
	CreatePerson(input model.PersonInput) (int, error)
	GetPeople(filter model.Filter) ([]model.Person, error)
	DeletePerson(id int) error
	UpdatePerson(id int, input model.PersonInput) error
}

// Handler handles HTTP requests and responses.
type Handler struct {
	service Service
	logger  *logrus.Logger
}

// NewHandler creates a new Handler instance.
func NewHandler(service Service, logger *logrus.Logger) *Handler {
	return &Handler{service: service, logger: logger}
}

// InitRoutes initializes the API routes.
func (h *Handler) InitRoutes() *gin.Engine {
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())

	api := router.Group("/api")
	{
		people := api.Group("/people")
		{
			people.POST("", h.createPerson)
			people.GET("", h.getPeople)
			people.PUT("/:id", h.updatePerson)
			people.DELETE("/:id", h.deletePerson)
		}
	}

	return router
}

// @Summary Create a new person
// @Tags people
// @Accept json
// @Produce json
// @Param input body model.PersonInput true "Person input"
// @Success 200 {object} map[string]int
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/people [post]
func (h *Handler) createPerson(c *gin.Context) {
	var input model.PersonInput
	if err := c.ShouldBindJSON(&input); err != nil {
		h.logger.WithFields(logrus.Fields{
			"error": err,
		}).Warn("Invalid input for creating person")
		// Форматируем ошибки валидации
		var validationErrors string
		if ginErr, ok := err.(*gin.Error); ok {
			validationErrors = ginErr.Err.Error()
		} else {
			validationErrors = err.Error()
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": "validation failed", "details": validationErrors})
		return
	}

	h.logger.WithFields(logrus.Fields{
		"name":        input.Name,
		"surname":     input.Surname,
		"patronymic":  input.Patronymic,
		"age":         input.Age,
		"gender":      input.Gender,
		"nationality": input.Nationality,
	}).Debug("Received request to create person")

	id, err := h.service.CreatePerson(input)
	if err != nil {
		h.logger.WithFields(logrus.Fields{
			"error": err,
		}).Error("Failed to create person")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	h.logger.WithFields(logrus.Fields{
		"id": id,
	}).Info("Person created successfully")
	c.JSON(http.StatusOK, gin.H{"id": id})
}

// @Summary Get people with filters and pagination
// @Tags people
// @Produce json
// @Param name query string false "Name filter"
// @Param surname query string false "Surname filter"
// @Param patronymic query string false "Patronymic filter"
// @Param age query int false "Age filter"
// @Param gender query string false "Gender filter"
// @Param nationality query string false "Nationality filter"
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(10)
// @Success 200 {array} model.Person
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/people [get]
func (h *Handler) getPeople(c *gin.Context) {
	var filter model.Filter

	filter.Name = c.Query("name")
	filter.Surname = c.Query("surname")
	filter.Patronymic = c.Query("patronymic")
	if ageStr := c.Query("age"); ageStr != "" {
		age, err := strconv.Atoi(ageStr)
		if err != nil {
			h.logger.WithFields(logrus.Fields{
				"age":   ageStr,
				"error": err,
			}).Warn("Invalid age parameter")
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid age parameter"})
			return
		}
		filter.Age = &age
	}
	filter.Gender = c.Query("gender")
	filter.Nationality = c.Query("nationality")

	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil || page < 1 {
		page = 1
	}
	limit, err := strconv.Atoi(c.DefaultQuery("limit", "10"))
	if err != nil || limit < 1 {
		limit = 10
	}
	filter.Page = page
	filter.Limit = limit

	h.logger.WithFields(logrus.Fields{
		"filter": filter,
	}).Debug("Received request to get people")

	people, err := h.service.GetPeople(filter)
	if err != nil {
		h.logger.WithFields(logrus.Fields{
			"error": err,
		}).Error("Failed to get people")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if people == nil {
		people = []model.Person{}
	}

	h.logger.WithFields(logrus.Fields{
		"count": len(people),
	}).Info("Retrieved people")
	c.JSON(http.StatusOK, people)
}

// @Summary Update a person
// @Tags people
// @Accept json
// @Produce json
// @Param id path int true "Person ID"
// @Param input body model.PersonInput true "Person input"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/people/{id} [put]
func (h *Handler) updatePerson(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		h.logger.WithFields(logrus.Fields{
			"id":    c.Param("id"),
			"error": err,
		}).Warn("Invalid ID for updating person")
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var input model.PersonInput
	if err := c.ShouldBindJSON(&input); err != nil {
		h.logger.WithFields(logrus.Fields{
			"error": err,
		}).Warn("Invalid input for updating person")
		// Форматируем ошибки валидации
		var validationErrors string
		if ginErr, ok := err.(*gin.Error); ok {
			validationErrors = ginErr.Err.Error()
		} else {
			validationErrors = err.Error()
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": "validation failed", "details": validationErrors})
		return
	}

	h.logger.WithFields(logrus.Fields{
		"id":          id,
		"name":        input.Name,
		"surname":     input.Surname,
		"patronymic":  input.Patronymic,
		"age":         input.Age,
		"gender":      input.Gender,
		"nationality": input.Nationality,
	}).Debug("Received request to update person")

	if err := h.service.UpdatePerson(id, input); err != nil {
		if err == sql.ErrNoRows {
			h.logger.WithFields(logrus.Fields{
				"id": id,
			}).Warn("Person not found")
			c.JSON(http.StatusNotFound, gin.H{"error": "person not found"})
			return
		}
		h.logger.WithFields(logrus.Fields{
			"id":    id,
			"error": err,
		}).Error("Failed to update person")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	h.logger.WithFields(logrus.Fields{
		"id": id,
	}).Info("Person updated successfully")
	c.JSON(http.StatusOK, gin.H{"message": "person updated"})
}

// @Summary Delete a person
// @Tags people
// @Produce json
// @Param id path int true "Person ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/people/{id} [delete]
func (h *Handler) deletePerson(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		h.logger.WithFields(logrus.Fields{
			"id":    c.Param("id"),
			"error": err,
		}).Warn("Invalid ID for deleting person")
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	h.logger.WithFields(logrus.Fields{
		"id": id,
	}).Debug("Received request to delete person")

	if err := h.service.DeletePerson(id); err != nil {
		if err == sql.ErrNoRows {
			h.logger.WithFields(logrus.Fields{
				"id": id,
			}).Warn("Person not found")
			c.JSON(http.StatusNotFound, gin.H{"error": "person not found"})
			return
		}
		h.logger.WithFields(logrus.Fields{
			"id":    id,
			"error": err,
		}).Error("Failed to delete person")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	h.logger.WithFields(logrus.Fields{
		"id": id,
	}).Info("Person deleted successfully")
	c.JSON(http.StatusOK, gin.H{"message": "person deleted"})
}
