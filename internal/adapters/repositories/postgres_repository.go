package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/juantevez/heic-app/internal/domain/entities"

	_ "github.com/lib/pq"
)

type PostgresPhotoRepository struct {
	db *sql.DB
}

func NewPostgresPhotoRepository(db *sql.DB) *PostgresPhotoRepository {
	return &PostgresPhotoRepository{
		db: db,
	}
}

func (r *PostgresPhotoRepository) SavePhotoExif(ctx context.Context, exif *entities.PhotoExif) error {
	query := `
        INSERT INTO photo_exif (id, file_name, latitude, longitude, date_time, camera_model, camera_maker)
        VALUES ($1, $2, $3, $4, $5, $6, $7)
    `

	_, err := r.db.ExecContext(ctx, query,
		exif.ID,
		exif.FileName,
		exif.Latitude,
		exif.Longitude,
		exif.DateTime,
		exif.CameraModel,
		exif.CameraMaker,
	)

	if err != nil {
		return fmt.Errorf("failed to save photo EXIF: %w", err)
	}

	return nil
}

func (r *PostgresPhotoRepository) SavePhotoFile(ctx context.Context, file *entities.PhotoFile) error {
	query := `
        INSERT INTO photo_file (id, id_exif, file_name, file_data, bike_id)
        VALUES ($1, $2, $3, $4, $5)
    `

	_, err := r.db.ExecContext(ctx, query,
		file.ID,
		file.IDExif,
		file.FileName,
		file.FileData,
		file.BikeID,
	)

	if err != nil {
		return fmt.Errorf("failed to save photo file: %w", err)
	}

	return nil
}

func (r *PostgresPhotoRepository) GetPhotoExif(ctx context.Context, id string) (*entities.PhotoExif, error) {
	query := `
        SELECT id, file_name, latitude, longitude, date_time, camera_model, camera_maker
        FROM photo_exif 
        WHERE id = $1
    `

	exif := &entities.PhotoExif{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&exif.ID,
		&exif.FileName,
		&exif.Latitude,
		&exif.Longitude,
		&exif.DateTime,
		&exif.CameraModel,
		&exif.CameraMaker,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("photo EXIF not found with ID: %s", id)
		}
		return nil, fmt.Errorf("failed to get photo EXIF: %w", err)
	}

	return exif, nil
}

func (r *PostgresPhotoRepository) GetPhotoFile(ctx context.Context, id string) (*entities.PhotoFile, error) {
	query := `
        SELECT id, id_exif, file_name, file_data, bike_id
        FROM photo_file 
        WHERE id = $1
    `

	file := &entities.PhotoFile{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&file.ID,
		&file.IDExif,
		&file.FileName,
		&file.FileData,
		&file.BikeID,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("photo file not found with ID: %s", id)
		}
		return nil, fmt.Errorf("failed to get photo file: %w", err)
	}

	return file, nil
}

func (r *PostgresPhotoRepository) GetPhotosByBikeID(ctx context.Context, bikeID int64) ([]entities.PhotoExif, error) {
	query := `
        SELECT pe.id, pe.file_name, pe.latitude, pe.longitude, pe.date_time, pe.camera_model, pe.camera_maker
        FROM photo_exif pe
        INNER JOIN photo_file pf ON pe.id = pf.id_exif
        WHERE pf.bike_id = $1
        ORDER BY pe.date_time DESC
    `

	rows, err := r.db.QueryContext(ctx, query, bikeID)
	if err != nil {
		return nil, fmt.Errorf("failed to get photos by bike ID: %w", err)
	}
	defer rows.Close()

	var photos []entities.PhotoExif

	for rows.Next() {
		exif := entities.PhotoExif{}
		err := rows.Scan(
			&exif.ID,
			&exif.FileName,
			&exif.Latitude,
			&exif.Longitude,
			&exif.DateTime,
			&exif.CameraModel,
			&exif.CameraMaker,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan photo EXIF: %w", err)
		}

		photos = append(photos, exif)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating photo rows: %w", err)
	}

	return photos, nil
}

// Database connection and initialization
func NewPostgresConnection(connectionString string) (*sql.DB, error) {
	db, err := sql.Open("postgres", connectionString)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(time.Hour)

	// Test the connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}
