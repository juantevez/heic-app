package services

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/juantevez/heic-app/internal/domain/entities"
	"github.com/dsoprea/go-exif/v3"	
	exifcommon "github.com/dsoprea/go-exif/v3/common"
	"github.com/google/uuid"
)

type ExifExtractorServiceImpl struct{}


// GPSCoordinate representa una coordenada GPS en formato [grados, minutos, segundos]
type GPSCoordinate []exif.Rational


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
	im := exif.NewStandardIfdMapping()


	ti := exif.NewTagIndex()
	_, index, err := exif.Collect(im, ti, rawExif)
	if err != nil {
		return err
	}

	fmt.Printf("=== EXIF Parsing Started ===\n")

	// MÉTODO 1: Intentar acceso directo a GPS usando la librería
	e.tryDirectGPSExtraction(index, photoExif)

	// MÉTODO 2: Extraer información del Root IFD
	if rootIfd := index.RootIfd; rootIfd != nil {
		fmt.Printf("Root IFD found, extracting tags...\n")

		// Extraer información básica de cámara
		e.extractCameraInfo(rootIfd, photoExif)

		// Búsqueda exhaustiva de GPS en TODAS las entradas
		e.exhaustiveGPSSearch(rootIfd, photoExif)
	}

	fmt.Printf("=== EXIF Parsing Completed ===\n")
	fmt.Printf("Final GPS - Lat: %v, Lon: %v\n", photoExif.Latitude, photoExif.Longitude)

	return nil
}

func (e *ExifExtractorServiceImpl) tryDirectGPSExtraction(index exif.IfdIndex, photoExif *entities.PhotoExif) {
	fmt.Printf("=== Trying Direct GPS Extraction ===\n")

	// Intentar obtener GPS info directamente del RootIfd
	if rootIfd := index.RootIfd; rootIfd != nil {
		// GpsInfo() devuelve (*exif.GpsInfo, error) - NO lat/lon directamente
		gpsInfo, err := rootIfd.GpsInfo()
		if err == nil && gpsInfo != nil {
			fmt.Printf("SUCCESS: GPS Info found in Root IFD\n")
			fmt.Printf("GPS Info struct: %+v\n", gpsInfo)

			// Por ahora solo logueamos que encontramos GpsInfo
			// La extracción real la haremos en exhaustiveGPSSearch
			fmt.Printf("GPS Info available but will extract coordinates manually\n")
		} else {
			fmt.Printf("No GPS Info in Root IFD: %v\n", err)
		}
	}

	fmt.Printf("Proceeding to exhaustive GPS search...\n")
}

func (e *ExifExtractorServiceImpl) exhaustiveGPSSearch(ifd *exif.Ifd, photoExif *entities.PhotoExif) {
	fmt.Printf("=== Exhaustive GPS Search ===\n")

	// Variables para almacenar datos GPS encontrados
	var gpsLatRef, gpsLonRef string
	var gpsLatCoords, gpsLonCoords []exif.Rational
	var foundGPSPointer bool

	entries := ifd.Entries()
	fmt.Printf("Searching %d entries for GPS data...\n", len(entries))

	// Buscar TODOS los tags, no solo los del rango GPS
	for _, entry := range entries {
		tagId := entry.TagId()
		value, err := entry.Value()
		if err != nil {
			continue
		}

		// Solo mostrar entries relevantes para no saturar el log
		if tagId <= 0x001F || tagId == 0x8825 || tagId == 0x8769 {
			fmt.Printf("Entry: 0x%04X = %+v (type: %T)\n", tagId, value, value)
		}

		switch tagId {
		case 0x0001: // GPSLatitudeRef
			if str, ok := value.(string); ok {
				gpsLatRef = strings.TrimSpace(str)
				fmt.Printf("*** GPS Latitude Ref found: %s\n", gpsLatRef)
			}
		case 0x0002: // GPSLatitude
			if coords, ok := value.([]exif.Rational); ok {
				gpsLatCoords = coords
				fmt.Printf("*** GPS Latitude coords found: %+v\n", coords)
			}
		case 0x0003: // GPSLongitudeRef
			if str, ok := value.(string); ok {
				gpsLonRef = strings.TrimSpace(str)
				fmt.Printf("*** GPS Longitude Ref found: %s\n", gpsLonRef)
			}
		case 0x0004: // GPSLongitude
			if coords, ok := value.([]exif.Rational); ok {
				gpsLonCoords = coords
				fmt.Printf("*** GPS Longitude coords found: %+v\n", coords)
			}
		case 0x8825: // GPS SubIFD pointer
			foundGPSPointer = true
			fmt.Printf("*** GPS SubIFD pointer found: %+v\n", value)

			// Procesar el offset y seguir al SubIFD
			var offset uint32
			if off, ok := value.(uint32); ok {
				offset = off
			} else if offs, ok := value.([]uint32); ok && len(offs) > 0 {
				offset = offs[0] // Tomamos el primer offset
				fmt.Printf("*** Using first GPS SubIFD offset: %d\n", offset)
			}

			if offset > 0 {
			// Extraer GPS del SubIFD
				e.extractGPSFromSubIFD(*index, offset, photoExif)
			}
	}

	// Procesar coordenadas GPS si se encontraron EN ESTE IFD
	if len(gpsLatCoords) >= 3 && len(gpsLonCoords) >= 3 {
		lat := GPSCoordinate(gpsLatCoords).ToDecimal()
		if gpsLatRef == "S" {
			lat = -lat
		}
		photoExif.Latitude = &lat

		lon := GPSCoordinate(gpsLonCoords).ToDecimal()
		if gpsLonRef == "W" {
			lon = -lon
		}
		photoExif.Longitude = &lon

		fmt.Printf("*** FINAL GPS Latitude: %f\n", lat)
		fmt.Printf("*** FINAL GPS Longitude: %f\n", lon)
	}

	// Si encontramos el pointer pero no las coordenadas en este IFD
	if foundGPSPointer {
		fmt.Printf("*** GPS SubIFD pointer found but coordinates are in separate SubIFD\n")
		fmt.Printf("*** This is normal for HEIC files - GPS data is in a separate IFD\n")
		fmt.Printf("*** Current go-exif library version may not support automatic SubIFD following\n")
	} else {
		fmt.Printf("*** No GPS data or pointer found in Root IFD\n")
	}

	// Intentar último método: verificar si tenemos GPS parcial
	if len(gpsLatCoords) > 0 || len(gpsLonCoords) > 0 || gpsLatRef != "" || gpsLonRef != "" {
		fmt.Printf("*** Partial GPS data found:\n")
		fmt.Printf("    Lat Ref: %s, Lat Coords: %v\n", gpsLatRef, gpsLatCoords)
		fmt.Printf("    Lon Ref: %s, Lon Coords: %v\n", gpsLonRef, gpsLonCoords)
	}
}

// ToDecimal convierte la coordenada a grados decimales
func (g GPSCoordinate) ToDecimal() float64 {
	if len(g) < 3 {
		return 0.0
	}

	degrees := float64(g[0].Numerator) / float64(g[0].Denominator)
	minutes := float64(g[1].Numerator) / float64(g[1].Denominator)
	seconds := float64(g[2].Numerator) / float64(g[2].Denominator)

	fmt.Printf("GPS Conversion - Degrees: %f, Minutes: %f, Seconds: %f\n", degrees, minutes, seconds)

	return degrees + minutes/60.0 + seconds/3600.0
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
		case 0x9003: // DateTimeOriginal (preferred)
			if value, err := entry.Value(); err == nil {
				if datetime, ok := value.(string); ok {
					cleaned := strings.TrimSpace(strings.Trim(datetime, "\x00"))
					if cleaned != "" {
						photoExif.DateTime = &cleaned // Override previous DateTime
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

func (e *ExifExtractorServiceImpl) extractGPSFromSubIFD(index exif.IfdIndex, offset uint32, photoExif *entities.PhotoExif) {
	fmt.Printf("=== Extracting GPS from SubIFD at offset %d ===\n", offset)

	// Crear el path al SubIFD usando el offset
	gpsIfd, err := index.RootIfd.ChildWithIfdPath([]uint32{offset})
	if err != nil {
		fmt.Printf("Error accessing GPS SubIFD at offset %d: %v\n", offset, err)
		return
	}

	if gpsIfd == nil {
		fmt.Printf("GPS SubIFD at offset %d is nil\n", offset)
		return
	}

	fmt.Printf("Successfully accessed GPS SubIFD at offset %d\n", offset)

	// Ahora extraer los tags GPS de este SubIFD
	var gpsLatRef, gpsLonRef string
	var gpsLatCoords, gpsLonCoords []exif.Rational

	entries := gpsIfd.Entries()
	fmt.Printf("Found %d entries in GPS SubIFD\n", len(entries))

	for _, entry := range entries {
		tagId := entry.TagId()
		value, err := entry.Value()
		if err != nil {
			continue
		}

		fmt.Printf("GPS SubIFD Entry: 0x%04X = %+v (type: %T)\n", tagId, value, value)

		switch tagId {
		case 0x0001: // GPSLatitudeRef
			if str, ok := value.(string); ok {
				gpsLatRef = strings.TrimSpace(str)
				fmt.Printf("*** GPS Latitude Ref found in SubIFD: %s\n", gpsLatRef)
			}
		case 0x0002: // GPSLatitude
			if coords, ok := value.([]exif.Rational); ok {
				gpsLatCoords = coords
				fmt.Printf("*** GPS Latitude coords found in SubIFD: %+v\n", coords)
			}
		case 0x0003: // GPSLongitudeRef
			if str, ok := value.(string); ok {
				gpsLonRef = strings.TrimSpace(str)
				fmt.Printf("*** GPS Longitude Ref found in SubIFD: %s\n", gpsLonRef)
			}
		case 0x0004: // GPSLongitude
			if coords, ok := value.([]exif.Rational); ok {
				gpsLonCoords = coords
				fmt.Printf("*** GPS Longitude coords found in SubIFD: %+v\n", coords)
			}
		}
	}

	// Procesar coordenadas si están completas
	if len(gpsLatCoords) >= 3 && len(gpsLonCoords) >= 3 {
		lat := GPSCoordinate(gpsLatCoords).ToDecimal()
		if gpsLatRef == "S" {
			lat = -lat
		}
		photoExif.Latitude = &lat

		lon := GPSCoordinate(gpsLonCoords).ToDecimal()
		if gpsLonRef == "W" {
			lon = -lon
		}
		photoExif.Longitude = &lon

		fmt.Printf("*** FINAL GPS Latitude: %f\n", lat)
		fmt.Printf("*** FINAL GPS Longitude: %f\n", lon)
	} else {
		fmt.Printf("*** Incomplete GPS data in SubIFD\n")
	}
}
