// internal/adapters/services/exif_extractor.go
package services

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"github.com/juantevez/heic-app/internal/domain/entities"

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
	rawExif, err := exif.SearchAndExtractExif(fileData)
	if err != nil {
		// Método alternativo: buscar manualmente el marcador EXIF
		return e.findExifInHEIC(fileData)
	}

	return rawExif, nil
}

func (e *ExifExtractorServiceImpl) findExifInHEIC(fileData []byte) ([]byte, error) {
	// Buscar el marcador EXIF en el archivo HEIC
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

	// Extraer información del Root IFD
	if rootIfd := index.RootIfd; rootIfd != nil {
		// Extraer información básica de cámara del Root IFD
		e.extractCameraInfo(rootIfd, photoExif)

		// Buscar y extraer GPS info
		e.extractGPSInfo(rootIfd, photoExif)

		// Buscar EXIF SubIFD para información adicional
		e.extractExifSubIfdInfo(rootIfd, photoExif)
	}

	return nil
}

func (e *ExifExtractorServiceImpl) extractGPSInfo(rootIfd *exif.Ifd, photoExif *entities.PhotoExif) {
	// Buscar GPS IFD entre los children del root IFD
	for _, childIfd := range rootIfd.Children {
		// Intentar extraer GPS info directamente
		if lat, lon, err := childIfd.GpsInfo(); err == nil {
			photoExif.Latitude = &lat
			photoExif.Longitude = &lon
			return
		}
	}

	// Si no funciona el método directo, buscar tags GPS manualmente
	e.extractGPSFromTags(rootIfd, photoExif)
}

func (e *ExifExtractorServiceImpl) extractGPSFromTags(ifd *exif.Ifd, photoExif *entities.PhotoExif) {
	entries := ifd.Entries()
	gpsData := make(map[uint16]interface{})

	// Buscar tags GPS (normalmente están en el rango 0x0000-0x001F)
	for _, entry := range entries {
		tagId := entry.TagId()

		// Tags GPS conocidos
		switch tagId {
		case 0x0001, 0x0002, 0x0003, 0x0004: // GPS Lat/Lon Ref and values
			if value, err := entry.Value(); err == nil {
				gpsData[tagId] = value
			}
		}
	}

	// Si encontramos datos GPS, parsearlos
	if len(gpsData) > 0 {
		e.parseGPSCoordinates(gpsData, photoExif)
	}
}

func (e *ExifExtractorServiceImpl) parseGPSCoordinates(gpsData map[uint16]interface{}, photoExif *entities.PhotoExif) {
	var latRef, lonRef string
	var lat, lon float64

	// GPS Latitude Reference (N/S)
	if val, ok := gpsData[0x0001]; ok {
		if str, ok := val.(string); ok {
			latRef = strings.TrimSpace(str)
		}
	}

	// GPS Latitude
	if val, ok := gpsData[0x0002]; ok {
		lat = e.parseGPSCoordinate(val)
		if latRef == "S" {
			lat = -lat
		}
		if lat != 0 {
			photoExif.Latitude = &lat
		}
	}

	// GPS Longitude Reference (E/W)
	if val, ok := gpsData[0x0003]; ok {
		if str, ok := val.(string); ok {
			lonRef = strings.TrimSpace(str)
		}
	}

	// GPS Longitude
	if val, ok := gpsData[0x0004]; ok {
		lon = e.parseGPSCoordinate(val)
		if lonRef == "W" {
			lon = -lon
		}
		if lon != 0 {
			photoExif.Longitude = &lon
		}
	}
}

func (e *ExifExtractorServiceImpl) parseGPSCoordinate(value interface{}) float64 {
	// Intentar diferentes tipos de valores GPS
	switch v := value.(type) {
	case []exifcommon.Rational:
		if len(v) >= 3 {
			deg := float64(v[0].Numerator) / float64(v[0].Denominator)
			min := float64(v[1].Numerator) / float64(v[1].Denominator)
			sec := float64(v[2].Numerator) / float64(v[2].Denominator)
			return deg + (min / 60.0) + (sec / 3600.0)
		}
	case []float64:
		if len(v) >= 3 {
			return v[0] + (v[1] / 60.0) + (v[2] / 3600.0)
		}
	case string:
		// Intentar parsear string con formato "40/1,26/1,4630/100"
		if coord, err := e.parseGPSString(v); err == nil {
			return coord
		}
	}
	return 0
}

func (e *ExifExtractorServiceImpl) parseGPSString(coordStr string) (float64, error) {
	parts := strings.Split(coordStr, ",")
	if len(parts) != 3 {
		return 0, fmt.Errorf("invalid GPS coordinate format")
	}

	var coord float64
	for i, part := range parts {
		fraction := strings.Split(strings.TrimSpace(part), "/")
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

	return coord, nil
}

func (e *ExifExtractorServiceImpl) extractExifSubIfdInfo(rootIfd *exif.Ifd, photoExif *entities.PhotoExif) {
	// Buscar EXIF SubIFD (tag 0x8769) para información adicional
	entries := rootIfd.Entries()

	for _, entry := range entries {
		if entry.TagId() == 0x8769 { // EXIF SubIFD pointer
			// Buscar entre los children IFDs
			for _, childIfd := range rootIfd.Children {
				e.extractExifInfo(childIfd, photoExif)
			}
			break
		}
	}
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
