// internal/adapters/services/exif_extractor.go
package services

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"heic-photo-processor/internal/domain/entities"

	"github.com/adrium/goheif"
	"github.com/dsoprea/go-exif/v3"
	"github.com/google/uuid"
)

type ExifExtractorServiceImpl struct{}

func NewExifExtractorService() *ExifExtractorServiceImpl {
	return &ExifExtractorServiceImpl{}
}

func (e *ExifExtractorServiceImpl) ExtractExif(fileData []byte, fileName string) (*entities.PhotoExif, error) {
	photoExif := &entities.PhotoExif{
		ID:       uuid.New().String(),
		FileName: fileName,
	}

	// Try to extract EXIF data from HEIC file
	exifData, err := e.extractFromHEIC(fileData)
	if err != nil {
		return nil, fmt.Errorf("failed to extract EXIF from HEIC: %w", err)
	}

	// Parse EXIF data
	if exifData != nil {
		e.parseExifData(exifData, photoExif)
	}

	return photoExif, nil
}

func (e *ExifExtractorServiceImpl) extractFromHEIC(fileData []byte) ([]byte, error) {
	reader := bytes.NewReader(fileData)

	// Decode HEIC to get access to metadata
	img, err := goheif.Decode(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to decode HEIC: %w", err)
	}

	// Try to extract EXIF from the decoded image
	// Note: goheif might not expose EXIF directly, so we try alternative approach
	reader.Seek(0, 0)
	rawExif, err := exif.SearchAndExtractExif(fileData)
	if err != nil {
		// If direct EXIF extraction fails, try to extract from the image structure
		return nil, fmt.Errorf("no EXIF data found in HEIC file: %w", err)
	}

	return rawExif, nil
}

func (e *ExifExtractorServiceImpl) parseExifData(rawExif []byte, photoExif *entities.PhotoExif) error {
	im, err := exifcommon.NewIfdMappingWithStandard()
	if err != nil {
		return err
	}

	ti := exif.NewTagIndex()
	_, index, err := exif.Collect(im, ti, rawExif)
	if err != nil {
		return err
	}

	// Extract GPS coordinates
	if gpsIfd, err := index.RootIfd.ChildWithIfdPath(exifcommon.IfdPathStandardGps); err == nil {
		if lat, lon, err := gpsIfd.GpsInfo(); err == nil {
			photoExif.Latitude = &lat
			photoExif.Longitude = &lon
		}
	}

	// Extract camera information and datetime
	if ifd0, err := index.RootIfd.ChildWithIfdPath(exifcommon.IfdPathStandardIfd0); err == nil {
		e.extractCameraInfo(ifd0, photoExif)
	}

	if exifIfd, err := index.RootIfd.ChildWithIfdPath(exifcommon.IfdPathStandardExif); err == nil {
		e.extractExifInfo(exifIfd, photoExif)
	}

	return nil
}

func (e *ExifExtractorServiceImpl) extractCameraInfo(ifd *exif.Ifd, photoExif *entities.PhotoExif) {
	entries := ifd.Entries()

	for _, entry := range entries {
		switch entry.TagId() {
		case 0x010F: // Make
			if value, err := entry.Value(); err == nil {
				if maker, ok := value.(string); ok {
					photoExif.CameraMaker = &maker
				}
			}
		case 0x0110: // Model
			if value, err := entry.Value(); err == nil {
				if model, ok := value.(string); ok {
					photoExif.CameraModel = &model
				}
			}
		case 0x0132: // DateTime
			if value, err := entry.Value(); err == nil {
				if datetime, ok := value.(string); ok {
					photoExif.DateTime = &datetime
				}
			}
		}
	}
}

func (e *ExifExtractorServiceImpl) extractExifInfo(ifd *exif.Ifd, photoExif *entities.PhotoExif) {
	entries := ifd.Entries()

	for _, entry := range entries {
		switch entry.TagId() {
		case 0x9003: // DateTimeOriginal
			if value, err := entry.Value(); err == nil {
				if datetime, ok := value.(string); ok {
					// Prefer DateTimeOriginal over DateTime
					photoExif.DateTime = &datetime
				}
			}
		}
	}
}

// Helper function to convert GPS coordinate from EXIF format
func (e *ExifExtractorServiceImpl) parseGPSCoordinate(coordStr string, ref string) (float64, error) {
	// Parse coordinates like "40/1,26/1,4630/100"
	parts := strings.Split(coordStr, ",")
	if len(parts) != 3 {
		return 0, fmt.Errorf("invalid GPS coordinate format")
	}

	var coord float64
	for i, part := range parts {
		fraction := strings.Split(part, "/")
		if len(fraction) != 2 {
			continue
		}

		numerator, err := strconv.ParseFloat(fraction[0], 64)
		if err != nil {
			continue
		}

		denominator, err := strconv.ParseFloat(fraction[1], 64)
		if err != nil || denominator == 0 {
			continue
		}

		value := numerator / denominator

		switch i {
		case 0: // degrees
			coord += value
		case 1: // minutes
			coord += value / 60
		case 2: // seconds
			coord += value / 3600
		}
	}

	// Apply hemisphere reference
	if ref == "S" || ref == "W" {
		coord = -coord
	}

	return coord, nil
}
