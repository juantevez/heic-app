// internal/domain/ports/repositories.go
package ports

import (
    "context"
    "heic-photo-processor/internal/domain/entities"
)

// PhotoRepository defines the interface for photo data persistence
type PhotoRepository interface {
    SavePhotoExif(ctx context.Context, exif *entities.PhotoExif) error
    SavePhotoFile(ctx context.Context, file *entities.PhotoFile) error
    GetPhotoExif(ctx context.Context, id string) (*entities.PhotoExif, error)
    GetPhotoFile(ctx context.Context, id string) (*entities.PhotoFile, error)
    GetPhotosByBikeID(ctx context.Context, bikeID int64) ([]entities.PhotoExif, error)
}

// internal/domain/ports/services.go
package ports

import (
    "context"
    "heic-photo-processor/internal/domain/entities"
    "mime/multipart"
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