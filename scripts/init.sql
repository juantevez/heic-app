-- Database initialization script

CREATE TABLE IF NOT EXISTS public.photo_exif (
    id varchar(36) NOT NULL,
    file_name varchar(255) NOT NULL,
    latitude float8 NULL,
    longitude float8 NULL,
    date_time varchar(50) NULL,
    camera_model varchar(100) NULL,
    camera_maker varchar(255) NULL,
    created_at timestamp DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT photo_exif_pkey PRIMARY KEY (id)
);

CREATE INDEX IF NOT EXISTS idx_photo_exif_coordinates ON public.photo_exif USING btree (latitude, longitude) 
WHERE ((latitude IS NOT NULL) AND (longitude IS NOT NULL));

CREATE INDEX IF NOT EXISTS idx_photo_exif_datetime ON public.photo_exif USING btree (date_time);

-- Create bike table (simplified for demo)
CREATE TABLE IF NOT EXISTS public.bike (
    bike_id BIGSERIAL PRIMARY KEY,
    name varchar(255) NOT NULL,
    model varchar(255),
    created_at timestamp DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS public.photo_file (
    id varchar(36) NOT NULL,
    id_exif varchar(36) NOT NULL,
    file_name varchar(255) NOT NULL,
    file_data bytea NOT NULL,
    bike_id int8 NULL,
    created_at timestamp DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT photo_file_pkey PRIMARY KEY (id),
    CONSTRAINT fk_photo_exif FOREIGN KEY (id_exif) REFERENCES public.photo_exif(id) ON DELETE CASCADE,
    CONSTRAINT fk_photo_bike FOREIGN KEY (bike_id) REFERENCES public.bike(bike_id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_photo_file_bike_id ON public.photo_file USING btree (bike_id);
CREATE INDEX IF NOT EXISTS idx_photo_file_exif_id ON public.photo_file USING btree (id_exif);

-- Insert sample bike data
INSERT INTO public.bike (name, model) VALUES 
    ('Mountain Explorer', 'Trek X-Caliber 9'),
    ('City Cruiser', 'Specialized Sirrus X'),
    ('Road Racer', 'Canyon Aeroad CF SL')
ON CONFLICT DO NOTHING;