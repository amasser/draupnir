package store

import (
	"database/sql"

	"github.com/gocardless/draupnir/pkg/models"
	_ "github.com/lib/pq" // used to setup the PG driver
)

type ImageStore interface {
	List() ([]models.Image, error)
	Create(models.Image) (models.Image, error)
	Get(id int) (models.Image, error)
	Destroy(image models.Image) error
	MarkAsReady(models.Image) (models.Image, error)
}

type DBImageStore struct {
	DB *sql.DB
}

func (s DBImageStore) List() ([]models.Image, error) {
	images := make([]models.Image, 0)

	rows, err := s.DB.Query(
		`SELECT id, backed_up_at, ready, created_at, updated_at FROM images ORDER BY id ASC`,
	)
	if err != nil {
		return images, err
	}

	defer rows.Close()

	var image models.Image
	for rows.Next() {
		err = rows.Scan(
			&image.ID,
			&image.BackedUpAt,
			&image.Ready,
			&image.CreatedAt,
			&image.UpdatedAt,
		)

		if err != nil {
			return images, err
		}

		images = append(images, image)
	}

	return images, nil
}

func (s DBImageStore) Get(id int) (models.Image, error) {
	image := models.Image{}

	row := s.DB.QueryRow(
		`SELECT id, backed_up_at, ready, anon, created_at, updated_at
		FROM images
		WHERE id = $1`,
		id,
	)
	err := row.Scan(
		&image.ID,
		&image.BackedUpAt,
		&image.Ready,
		&image.Anon,
		&image.CreatedAt,
		&image.UpdatedAt,
	)
	if err != nil {
		return image, err
	}

	return image, nil
}

func (s DBImageStore) Create(image models.Image) (models.Image, error) {
	row := s.DB.QueryRow(
		`INSERT INTO images (backed_up_at, ready, anon, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, backed_up_at, ready, created_at, updated_at`,
		image.BackedUpAt,
		image.Ready,
		image.Anon,
		image.CreatedAt,
		image.UpdatedAt,
	)

	err := row.Scan(
		&image.ID,
		&image.BackedUpAt,
		&image.Ready,
		&image.CreatedAt,
		&image.UpdatedAt,
	)
	if err != nil {
		return image, err
	}
	return image, nil
}

func (s DBImageStore) MarkAsReady(image models.Image) (models.Image, error) {
	row := s.DB.QueryRow(
		`UPDATE images
		 SET ready = TRUE,
				 updated_at = now()
		 WHERE id = $1
		 AND ready = $2
		 RETURNING id, backed_up_at, ready, created_at, updated_at`,
		image.ID,
		image.Ready,
	)

	err := row.Scan(
		&image.ID,
		&image.BackedUpAt,
		&image.Ready,
		&image.CreatedAt,
		&image.UpdatedAt,
	)
	if err != nil {
		return image, err
	}
	return image, nil
}

func (s DBImageStore) Destroy(image models.Image) error {
	_, err := s.DB.Exec("DELETE FROM images WHERE id = $1", image.ID)
	return err
}
