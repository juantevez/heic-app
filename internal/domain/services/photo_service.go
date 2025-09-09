package services

import (
	"context"
	"fmt"
	"time"

	"github.com/juantevez/heic-app/internal/domain/entities"
	"github.com/juantevez/heic-app/internal/domain/ports"

	"github.com/google/uuid"
)

type PhotoServiceImpl struct {
	photoRepo     ports.PhotoRepository
	exifExtractor ports.ExifExtractorService
}

func NewPhotoService(photoRepo ports.PhotoRepository, exifExtractor ports.ExifExtractorService) *PhotoServiceImpl {
	return &PhotoServiceImpl{
		photoRepo:     photoRepo,
		exifExtractor: exifExtractor,
	}
}

func (s *PhotoServiceImpl) ProcessPhoto(ctx context.Context, req entities.UploadRequest) (*entities.PhotoResponse, error) {
	fmt.Printf("=== Processing Photo: %s ===\n", req.FileName)

	// Extract EXIF data
	exifData, err := s.exifExtractor.ExtractExif(req.FileData, req.FileName)
	if err != nil {
		return nil, fmt.Errorf("failed to extract EXIF data: %w", err)
	}

	fmt.Printf("EXIF Data extracted:\n")
	fmt.Printf("- ID: %s\n", exifData.ID)
	fmt.Printf("- FileName: %s\n", exifData.FileName)
	fmt.Printf("- Latitude: %v\n", exifData.Latitude)
	fmt.Printf("- Longitude: %v\n", exifData.Longitude)
	fmt.Printf("- DateTime: %v\n", exifData.DateTime)
	fmt.Printf("- CameraModel: %v\n", exifData.CameraModel)
	fmt.Printf("- CameraMaker: %v\n", exifData.CameraMaker)

	if exifData.Latitude != nil {
		fmt.Printf("- Latitude Value: %f\n", *exifData.Latitude)
	} else {
		fmt.Printf("- Latitude is NIL\n")
	}

	if exifData.Longitude != nil {
		fmt.Printf("- Longitude Value: %f\n", *exifData.Longitude)
	} else {
		fmt.Printf("- Longitude is NIL\n")
	}

	exifData.CreatedAt = time.Now()

	// Create photo file entity
	photoFile := &entities.PhotoFile{
		ID:        uuid.New().String(),
		IDExif:    exifData.ID,
		FileName:  req.FileName,
		FileData:  req.FileData,
		BikeID:    req.BikeID,
		CreatedAt: time.Now(),
	}

	// Save EXIF data first
	fmt.Printf("Saving EXIF data to database...\n")
	if err := s.photoRepo.SavePhotoExif(ctx, exifData); err != nil {
		return nil, fmt.Errorf("failed to save EXIF data: %w", err)
	}

	// Save photo file
	fmt.Printf("Saving photo file to database...\n")
	if err := s.photoRepo.SavePhotoFile(ctx, photoFile); err != nil {
		return nil, fmt.Errorf("failed to save photo file: %w", err)
	}

	// Create response
	response := &entities.PhotoResponse{
		ID:       photoFile.ID,
		FileName: req.FileName,
		Message:  "Photo processed and saved successfully",
		ExifData: *exifData,
	}

	fmt.Printf("Response EXIF Data:\n")
	fmt.Printf("- Response Latitude: %v\n", response.ExifData.Latitude)
	fmt.Printf("- Response Longitude: %v\n", response.ExifData.Longitude)

	return response, nil
}

func (s *PhotoServiceImpl) GetPhoto(ctx context.Context, id string) (*entities.Photo, error) {
	// Get photo file first
	photoFile, err := s.photoRepo.GetPhotoFile(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get photo file: %w", err)
	}

	// Get corresponding EXIF data
	exifData, err := s.photoRepo.GetPhotoExif(ctx, photoFile.IDExif)
	if err != nil {
		return nil, fmt.Errorf("failed to get EXIF data: %w", err)
	}

	return &entities.Photo{
		Exif: *exifData,
		File: *photoFile,
	}, nil
}

func (s *PhotoServiceImpl) GetPhotosByBike(ctx context.Context, bikeID int64) ([]entities.PhotoExif, error) {
	photos, err := s.photoRepo.GetPhotosByBikeID(ctx, bikeID)
	if err != nil {
		return nil, fmt.Errorf("failed to get photos by bike ID: %w", err)
	}

	return photos, nil
}

// ValidateUploadRequest validates the upload request
func (s *PhotoServiceImpl) ValidateUploadRequest(req entities.UploadRequest) error {
	if req.FileName == "" {
		return fmt.Errorf("file name is required")
	}

	if len(req.FileData) == 0 {
		return fmt.Errorf("file data is required")
	}

	// Check if it's a HEIC file (basic check)
	if !s.isHEICFile(req.FileName, req.FileData) {
		return fmt.Errorf("file must be a HEIC image")
	}

	// Check file size (limit to 10MB)
	if len(req.FileData) > 10*1024*1024 {
		return fmt.Errorf("file size exceeds 10MB limit")
	}

	return nil
}

func (s *PhotoServiceImpl) isHEICFile(fileName string, fileData []byte) bool {
	// Check file extension
	if len(fileName) < 5 {
		return false
	}

	extension := fileName[len(fileName)-5:]
	if extension != ".heic" && extension != ".HEIC" {
		// Also check for .heif
		if len(fileName) >= 5 {
			heifExt := fileName[len(fileName)-5:]
			if heifExt != ".heif" && heifExt != ".HEIF" {
				return false
			}
		} else {
			return false
		}
	}

	// Check HEIC/HEIF magic bytes
	if len(fileData) < 12 {
		return false
	}

	// HEIC files start with specific byte patterns
	// Check for 'ftyp' box and HEIC brand
	if fileData[4] == 'f' && fileData[5] == 't' &&
		fileData[6] == 'y' && fileData[7] == 'p' {
		// Look for HEIC or HEIF brand
		for i := 8; i < len(fileData)-4 && i < 32; i++ {
			if fileData[i] == 'h' && fileData[i+1] == 'e' &&
				fileData[i+2] == 'i' && fileData[i+3] == 'c' {
				return true
			}
			if fileData[i] == 'h' && fileData[i+1] == 'e' &&
				fileData[i+2] == 'i' && fileData[i+3] == 'f' {
				return true
			}
		}
	}

	return false
}
