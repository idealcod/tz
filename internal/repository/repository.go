package repository

import (
	"database/sql"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
	"github.com/sirupsen/logrus"

	"effective-mobile.test/internal/config"
	"effective-mobile.test/internal/model"
)

// Repository handles database operations.
type Repository struct {
	db     *sql.DB
	logger *logrus.Logger
}

// NewRepository creates a new Repository instance.
func NewRepository(db *sql.DB, logger *logrus.Logger) *Repository {
	return &Repository{db: db, logger: logger}
}

// NewPostgresDB establishes a connection to the PostgreSQL database.
func NewPostgresDB(cfg *config.Config) (*sql.DB, error) {
	db, err := sql.Open("postgres", cfg.DSN())
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return db, nil
}

// RunMigrations applies database migrations.
func RunMigrations(cfg *config.Config) error {
	db, err := sql.Open("postgres", cfg.DSN())
	if err != nil {
		return err
	}
	defer db.Close()

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return err
	}

	m, err := migrate.NewWithDatabaseInstance(
		fmt.Sprintf("file://%s", cfg.MigrationPath),
		"postgres", driver)
	if err != nil {
		return err
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return err
	}

	return nil
}

// CreatePerson creates a new person in the database.
func (r *Repository) CreatePerson(person *model.Person) (int, error) {
	r.logger.WithFields(logrus.Fields{
		"name":        person.Name,
		"surname":     person.Surname,
		"patronymic":  person.Patronymic,
		"age":         person.Age,
		"gender":      person.Gender,
		"nationality": person.Nationality,
	}).Debug("Creating person")

	query := `
        INSERT INTO people (name, surname, patronymic, age, gender, nationality)
        VALUES ($1, $2, $3, $4, $5, $6)
        RETURNING id`
	r.logger.WithFields(logrus.Fields{
		"query": query,
		"args":  []interface{}{person.Name, person.Surname, person.Patronymic, person.Age, person.Gender, person.Nationality},
	}).Debug("Executing SQL query")

	var id int
	err := r.db.QueryRow(query, person.Name, person.Surname, person.Patronymic, person.Age, person.Gender, person.Nationality).Scan(&id)
	if err != nil {
		r.logger.WithFields(logrus.Fields{
			"error": err,
		}).Error("Failed to create person")
		return 0, err
	}

	r.logger.WithFields(logrus.Fields{
		"id": id,
	}).Info("Person created")
	return id, nil
}

// GetPeople retrieves people based on filters and pagination.
func (r *Repository) GetPeople(filter model.Filter) ([]model.Person, error) {
	r.logger.WithFields(logrus.Fields{
		"filter": filter,
	}).Debug("Fetching people")

	query := "SELECT id, name, surname, patronymic, age, gender, nationality FROM people WHERE 1=1"
	args := []interface{}{}
	count := 1

	if filter.Name != "" {
		query += fmt.Sprintf(" AND name ILIKE $%d", count)
		args = append(args, "%"+filter.Name+"%")
		count++
	}
	if filter.Surname != "" {
		query += fmt.Sprintf(" AND surname ILIKE $%d", count)
		args = append(args, "%"+filter.Surname+"%")
		count++
	}
	if filter.Patronymic != "" {
		query += fmt.Sprintf(" AND patronymic ILIKE $%d", count)
		args = append(args, "%"+filter.Patronymic+"%")
		count++
	}
	if filter.Age != nil {
		query += fmt.Sprintf(" AND age = $%d", count)
		args = append(args, *filter.Age)
		count++
	}
	if filter.Gender != "" {
		query += fmt.Sprintf(" AND gender = $%d", count)
		args = append(args, filter.Gender)
		count++
	}
	if filter.Nationality != "" {
		query += fmt.Sprintf(" AND nationality = $%d", count)
		args = append(args, filter.Nationality)
		count++
	}

	offset := (filter.Page - 1) * filter.Limit
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", count, count+1)
	args = append(args, filter.Limit, offset)

	r.logger.WithFields(logrus.Fields{
		"query": query,
		"args":  args,
	}).Debug("Executing SQL query")

	rows, err := r.db.Query(query, args...)
	if err != nil {
		r.logger.WithFields(logrus.Fields{
			"error": err,
		}).Error("Failed to fetch people")
		return nil, err
	}
	defer rows.Close()

	var people []model.Person
	for rows.Next() {
		var p model.Person
		if err := rows.Scan(&p.ID, &p.Name, &p.Surname, &p.Patronymic, &p.Age, &p.Gender, &p.Nationality); err != nil {
			r.logger.WithFields(logrus.Fields{
				"error": err,
			}).Error("Failed to scan person")
			return nil, err
		}
		people = append(people, p)
	}

	r.logger.WithFields(logrus.Fields{
		"count": len(people),
	}).Info("Fetched people")
	return people, nil
}

// DeletePerson deletes a person by ID.
func (r *Repository) DeletePerson(id int) error {
	r.logger.WithFields(logrus.Fields{
		"id": id,
	}).Debug("Deleting person")

	query := "DELETE FROM people WHERE id = $1"
	r.logger.WithFields(logrus.Fields{
		"query": query,
		"args":  []interface{}{id},
	}).Debug("Executing SQL query")

	result, err := r.db.Exec(query, id)
	if err != nil {
		r.logger.WithFields(logrus.Fields{
			"error": err,
		}).Error("Failed to delete person")
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		r.logger.WithFields(logrus.Fields{
			"error": err,
		}).Error("Failed to check rows affected")
		return err
	}
	if rowsAffected == 0 {
		r.logger.WithFields(logrus.Fields{
			"id": id,
		}).Warn("No person found")
		return sql.ErrNoRows
	}

	r.logger.WithFields(logrus.Fields{
		"id": id,
	}).Info("Person deleted")
	return nil
}

// UpdatePerson updates a person by ID.
func (r *Repository) UpdatePerson(id int, person *model.Person) error {
	r.logger.WithFields(logrus.Fields{
		"id":          id,
		"name":        person.Name,
		"surname":     person.Surname,
		"patronymic":  person.Patronymic,
		"age":         person.Age,
		"gender":      person.Gender,
		"nationality": person.Nationality,
	}).Debug("Updating person")

	query := `
        UPDATE people
        SET name = $1, surname = $2, patronymic = $3, age = $4, gender = $5, nationality = $6
        WHERE id = $7`
	r.logger.WithFields(logrus.Fields{
		"query": query,
		"args":  []interface{}{person.Name, person.Surname, person.Patronymic, person.Age, person.Gender, person.Nationality, id},
	}).Debug("Executing SQL query")

	result, err := r.db.Exec(query, person.Name, person.Surname, person.Patronymic, person.Age, person.Gender, person.Nationality, id)
	if err != nil {
		r.logger.WithFields(logrus.Fields{
			"error": err,
		}).Error("Failed to update person")
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		r.logger.WithFields(logrus.Fields{
			"error": err,
		}).Error("Failed to check rows affected")
		return err
	}
	if rowsAffected == 0 {
		r.logger.WithFields(logrus.Fields{
			"id": id,
		}).Warn("No person found")
		return sql.ErrNoRows
	}

	r.logger.WithFields(logrus.Fields{
		"id": id,
	}).Info("Person updated")
	return nil
}
