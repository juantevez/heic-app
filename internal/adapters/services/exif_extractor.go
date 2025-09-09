package services

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"heic-photo-processor/internal/domain/entities"

	"github.com/dsoprea/go-exif/v3"
	exifcommon "github.com/dsoprea/go-exif/v3/common"
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

	// Verificar que es un archivo HEIC válido
	if !e.isValidHEICFile(fileData) {
		return nil, fmt.Errorf("invalid HEIC file format")
	}

	// Extraer EXIF data directamente del archivo HEIC
	exifData, err := e.extractExifFromHEIC(fileData)
	if err != nil {
		// Si no se puede extraer EXIF, devolver estructura básica
		fmt.Printf("Warning: Could not extract EXIF data: %v\n", err)
		return photoExif, nil
	}

	// Parsear datos EXIF
	if exifData != nil {
		err := e.parseExifData(exifData, photoExif)
		if err != nil {
			fmt.Printf("Warning: Could not parse EXIF data: %v\n", err)
		}
	}

	return photoExif, nil
}

func (e *ExifExtractorServiceImpl) extractExifFromHEIC(fileData []byte) ([]byte, error) {
	// Buscar directamente el segmento EXIF en el archivo HEIC
	// Los archivos HEIC pueden contener datos EXIF embebidos

	// Método 1: Usar go-exif para buscar y extraer EXIF
	rawExif, err := exif.SearchAndExtractExif(fileData)
	if err != nil {
		// Método 2: Buscar manualmente el marcador EXIF
		return e.findExifInHEIC(fileData)
	}

	return rawExif, nil
}

func (e *ExifExtractorServiceImpl) findExifInHEIC(fileData []byte) ([]byte, error) {
	// Buscar el marcador EXIF en el archivo HEIC
	// Los archivos HEIC/HEIF pueden tener EXIF embebido en diferentes ubicaciones

	// Buscar patrones comunes de EXIF
	exifMarkers := [][]byte{
		{0x45, 0x78, 0x69, 0x66}, // "Exif"
		{0xFF, 0xE1},             // APP1 marker
	}

	for _, marker := range exifMarkers {
		if idx := bytes.Index(fileData, marker); idx != -1 {
			// Intentar extraer datos EXIF desde esta posición
			if idx+8 < len(fileData) {
				// Buscar el inicio de los datos TIFF
				start := idx
				for i := start; i < len(fileData)-4; i++ {
					if (fileData[i] == 0x4D && fileData[i+1] == 0x4D) || // Big endian TIFF
						(fileData[i] == 0x49 && fileData[i+1] == 0x49) { // Little endian TIFF
						// Encontrado inicio de TIFF, extraer hasta el final probable
						maxLen := 65536 // Máximo tamaño EXIF típico
						end := i + maxLen
						if end > len(fileData) {
							end = len(fileData)
						}
						return fileData[i:end], nil
					}
				}
			}
		}
	}

	return nil, fmt.Errorf("no EXIF data found in HEIC file")
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

	// Extraer coordenadas GPS
	if gpsIfd, err := index.RootIfd.ChildWithIfdPath(exifcommon.IfdPathStandardGps); err == nil {
		if lat, lon, err := gpsIfd.GpsInfo(); err == nil {
			photoExif.Latitude = &lat
			photoExif.Longitude = &lon
		}
	}

	// Extraer información de cámara y datetime desde IFD0
	if ifd0, err := index.RootIfd.ChildWithIfdPath(exifcommon.IfdPathStandardIfd0); err == nil {
		e.extractCameraInfo(ifd0, photoExif)
	}

	// Extraer información adicional desde EXIF IFD
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
					cleaned := strings.TrimSpace(strings.Trim(maker, "\x00"))
					if cleaned != "" {
						photoExif.CameraMaker = &cleaned
					}
				}
			}
		case 0x0110: // Model
			if value, err := entry.Value(); err == nil {
				if model, ok := value.(string); ok {
					cleaned := strings.TrimSpace(strings.Trim(model, "\x00"))
					if cleaned != "" {
						photoExif.CameraModel = &cleaned
					}
				}
			}
		case 0x0132: // DateTime
			if value, err := entry.Value(); err == nil {
				if datetime, ok := value.(string); ok {
					cleaned := strings.TrimSpace(strings.Trim(datetime, "\x00"))
					if cleaned != "" {
						photoExif.DateTime = &cleaned
					}
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
					cleaned := strings.TrimSpace(strings.Trim(datetime, "\x00"))
					if cleaned != "" {
						// Preferir DateTimeOriginal sobre DateTime
						photoExif.DateTime = &cleaned
					}
				}
			}
		case 0x9004: // DateTimeDigitized
			if value, err := entry.Value(); err == nil && photoExif.DateTime == nil {
				if datetime, ok := value.(string); ok {
					cleaned := strings.TrimSpace(strings.Trim(datetime, "\x00"))
					if cleaned != "" {
						photoExif.DateTime = &cleaned
					}
				}
			}
		}
	}
}

func (e *ExifExtractorServiceImpl) isValidHEICFile(fileData []byte) bool {
	if len(fileData) < 12 {
		return false
	}

	// Verificar signature HEIC/HEIF
	// Los archivos HEIC comienzan con un box 'ftyp'
	if fileData[4] == 'f' && fileData[5] == 't' &&
		fileData[6] == 'y' && fileData[7] == 'p' {

		// Buscar brand HEIC, HEIF, mif1, etc.
		brands := []string{"heic", "heix", "hevc", "hevx", "heim", "heis", "hevm", "hevs", "mif1", "msf1"}

		// Buscar en los primeros 32 bytes después del header ftyp
		searchArea := fileData[8:]
		if len(searchArea) > 32 {
			searchArea = searchArea[:32]
		}

		content := strings.ToLower(string(searchArea))
		for _, brand := range brands {
			if strings.Contains(content, brand) {
				return true
			}
		}
	}

	return false
}

// Helper function para parsear coordenadas GPS (si se necesita implementación manual)
func (e *ExifExtractorServiceImpl) parseGPSCoordinate(coordStr string, ref string) (float64, error) {
	// Parse coordinates como "40/1,26/1,4630/100"
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

	// Aplicar referencia de hemisferio
	if ref == "S" || ref == "W" {
		coord = -coord
	}

	return coord, nil
}
