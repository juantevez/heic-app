package entities

import (
	"time"
)

// PhotoExif represents EXIF data extracted from a photo
type PhotoExif struct {
	ID          string    `json:"id"`
	FileName    string    `json:"file_name"`
	Latitude    *float64  `json:"latitude"`  // Removido omitempty
	Longitude   *float64  `json:"longitude"` // Removido omitempty
	DateTime    *string   `json:"date_time,omitempty"`
	CameraModel *string   `json:"camera_model,omitempty"`
	CameraMaker *string   `json:"camera_maker,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// PhotoFile represents the binary file data with reference to EXIF
type PhotoFile struct {
	ID        string    `json:"id"`
	IDExif    string    `json:"id_exif"`
	FileName  string    `json:"file_name"`
	FileData  []byte    `json:"-"` // Excluded from JSON for security
	BikeID    *int64    `json:"bike_id,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// Photo represents the complete photo with EXIF and file data
type Photo struct {
	Exif PhotoExif `json:"exif"`
	File PhotoFile `json:"file"`
}

// UploadRequest represents a photo upload request
type UploadRequest struct {
	FileName string `json:"file_name"`
	FileData []byte `json:"file_data"`
	BikeID   *int64 `json:"bike_id,omitempty"`
}

// PhotoResponse represents the API response after photo processing
type PhotoResponse struct {
	ID       string    `json:"id"`
	FileName string    `json:"file_name"`
	Message  string    `json:"message"`
	ExifData PhotoExif `json:"exif_data"`
}
