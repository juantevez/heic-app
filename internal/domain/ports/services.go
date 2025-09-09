package ports

import (
	"context"
	"mime/multipart"

	"github.com/juantevez/heic-app/internal/domain/entities"
)

// ExifExtractorService defines the interface for EXIF data extraction
type ExifExtractorService interface {
	ExtractExif(fileData []byte, fileName string) (*entities.PhotoExif, error)
}

// PhotoService defines the interface for photo business logic
type PhotoService interface {
	ProcessPhoto(ctx context.Context, req entities.UploadRequest) (*entities.PhotoResponse, error)
	GetPhoto(ctx context.Context, id string) (*entities.Photo, error)
	GetPhotosByBike(ctx context.Context, bikeID int64) ([]entities.PhotoExif, error)
}

// FileHandler defines the interface for handling multipart files
type FileHandler interface {
	ExtractFileData(file multipart.File, header *multipart.FileHeader) ([]byte, error)
	ValidateHEICFile(fileName string, fileData []byte) error
}
