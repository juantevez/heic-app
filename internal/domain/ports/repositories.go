package ports

import (
	"context"

	"github.com/juantevez/heic-app/internal/domain/entities"
)

// PhotoRepository defines the interface for photo data persistence
type PhotoRepository interface {
	SavePhotoExif(ctx context.Context, exif *entities.PhotoExif) error
	SavePhotoFile(ctx context.Context, file *entities.PhotoFile) error
	GetPhotoExif(ctx context.Context, id string) (*entities.PhotoExif, error)
	GetPhotoFile(ctx context.Context, id string) (*entities.PhotoFile, error)
	GetPhotosByBikeID(ctx context.Context, bikeID int64) ([]entities.PhotoExif, error)
}
