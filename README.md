Una aplicación en Go 1.24 con arquitectura hexagonal que procesa archivos de imagen HEIC, extrae metadatos EXIF y los almacena en PostgreSQL.

## Características

- **Arquitectura Hexagonal**: Separación clara entre dominio, puertos y adaptadores
- **Procesamiento HEIC**: Extracción completa de metadatos EXIF de archivos HEIC
- **PostgreSQL**: Almacenamiento optimizado con índices para consultas geoespaciales
- **REST API**: Endpoints completos para subir, consultar y descargar fotos
- **Docker**: Configuración completa con docker-compose
- **Validación**: Validación de archivos HEIC y limitaciones de tamaño

## Arquitectura

```
cmd/
├── server/              # Punto de entrada de la aplicación
internal/
├── domain/
│   ├── entities/        # Entidades de dominio
│   ├── ports/          # Interfaces (puertos)
│   └── services/       # Lógica de negocio
├── adapters/
│   ├── http/           # Adaptador HTTP (REST API)
│   ├── repositories/   # Adaptador PostgreSQL
│   └── services/       # Servicios externos (EXIF extractor)
└── config/             # Configuración
```

## Requisitos

- Go 1.24+
- PostgreSQL 12+
- Docker & Docker Compose (opcional)
- libheif (para procesamiento HEIC)

## Instalación

### Desarrollo Local

1. **Clonar el repositorio**
```bash
git clone <repository-url>
cd heic-photo-processor
```

2. **Configurar variables de entorno**
```bash
cp .env.example .env
# Editar .env con tus configuraciones
```

3. **Instalar dependencias**
```bash
make deps
```

4. **Configurar base de datos**
```bash
# Crear base de datos PostgreSQL
createdb photo_db

# Ejecutar migraciones
make db-migrate
```

5. **Ejecutar aplicación**
```bash
make run
```

### Con Docker

```bash
# Iniciar todos los servicios
make docker-run

# Ver logs
make docker-logs

# Detener servicios
make docker-down
```

## API Endpoints

### Subir Foto HEIC
```bash
POST /api/v1/photos
Content-Type: multipart/form-data

# Parámetros:
# - photo: archivo HEIC (requerido)
# - bike_id: ID de la bicicleta (opcional)

curl -X POST http://localhost:8080/api/v1/photos \
  -F "photo=@imagen.heic" \
  -F "bike_id=1"
```

### Obtener Información de Foto
```bash
GET /api/v1/photos/{id}

curl http://localhost:8080/api/v1/photos/uuid-de-la-foto
```

### Descargar Archivo Original
```bash
GET /api/v1/photos/{id}/download

curl -O http://localhost:8080/api/v1/photos/uuid-de-la-foto/download
```

### Obtener Fotos por Bicicleta
```bash
GET /api/v1/bikes/{bike_id}/photos

curl http://localhost:8080/api/v1/bikes/1/photos
```

### Health Check
```bash
GET /api/v1/health

curl http://localhost:8080/api/v1/health
```

## Estructura de Base de Datos

### Tabla photo_exif
- Almacena metadatos EXIF extraídos
- Incluye coordenadas GPS, información de cámara, fecha/hora
- Índices optimizados para consultas geoespaciales

### Tabla photo_file
- Almacena datos binarios del archivo
- Referencia a metadatos EXIF
- Relación opcional con bicicletas

## Ejemplo de Respuesta

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "file_name": "IMG_1234.heic",
  "message": "Photo processed and saved successfully",
  "exif_data": {
    "id": "550e8400-e29b-41d4-a716-446655440001",
    "file_name": "IMG_1234.heic",
    "latitude": 40.7128,
    "longitude": -74.0060,
    "date_time": "2024:01:15 14:30:25",
    "camera_model": "iPhone 15 Pro",
    "camera_maker": "Apple"
  }
}
```

## Desarrollo

### Ejecutar Tests
```bash
make test
```

### Linting
```bash
make lint
```

### Estructura del Proyecto
- **Domain**: Lógica de negocio pura, sin dependencias externas
- **Ports**: Interfaces que definen contratos
- **Adapters**: Implementaciones concretas de los puertos
- **Config**: Gestión centralizada de configuración

## Configuración

Variables de entorno disponibles:

```bash
# Servidor
SERVER_PORT=8080
SERVER_HOST=0.0.0.0
SERVER_READ_TIMEOUT=10
SERVER_WRITE_TIMEOUT=10

# Base de datos
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=tu_password
DB_NAME=photo_db
DB_SSLMODE=disable
```

## Limitaciones

- Archivos HEIC máximo 10MB
- Soporte específico para formato HEIC/HEIF
- Requiere libheif instalado en el sistema

## Contribuir

1. Fork del proyecto
2. Crear rama feature (`git checkout -b feature/nueva-funcionalidad`)
3. Commit cambios (`git commit -am 'Agregar nueva funcionalidad'`)
4. Push a la rama (`git push origin feature/nueva-funcionalidad`)
5. Crear Pull Request

## Licencia

MIT License
