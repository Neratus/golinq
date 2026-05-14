-- Таблица стран
CREATE TABLE IF NOT EXISTS country (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    population BIGINT,
    area NUMERIC(12,2)
);

-- Таблица городов (один-ко-многим)
CREATE TABLE IF NOT EXISTS city (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    population BIGINT,
    country_id UUID NOT NULL REFERENCES country(id) ON DELETE CASCADE
);

-- Таблица языков (многие-ко-многим через связующую таблицу)
CREATE TABLE IF NOT EXISTS language (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL UNIQUE,
    code VARCHAR(10) NOT NULL UNIQUE
);

CREATE TABLE IF NOT EXISTS country_language (
    country_id UUID NOT NULL REFERENCES country(id) ON DELETE CASCADE,
    language_id UUID NOT NULL REFERENCES language(id) ON DELETE CASCADE,
    is_official BOOLEAN DEFAULT false,
    PRIMARY KEY (country_id, language_id)
);

-- Вставка тестовых данных
INSERT INTO country (id, name, population, area) VALUES 
    ('11111111-1111-1111-1111-111111111111', 'USA', 331000000, 9833517),
    ('22222222-2222-2222-2222-222222222222', 'Canada', 38000000, 9984670),
    ('33333333-3333-3333-3333-333333333333', 'Mexico', 128900000, 1964375);

INSERT INTO city (name, population, country_id) VALUES 
    ('New York', 8419000, '11111111-1111-1111-1111-111111111111'),
    ('Los Angeles', 3980000, '11111111-1111-1111-1111-111111111111'),
    ('Toronto', 2930000, '22222222-2222-2222-2222-222222222222'),
    ('Mexico City', 9200000, '33333333-3333-3333-3333-333333333333');

INSERT INTO language (id, name, code) VALUES
    ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', 'English', 'en'),
    ('bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb', 'Spanish', 'es'),
    ('cccccccc-cccc-cccc-cccc-cccccccccccc', 'French', 'fr');

INSERT INTO country_language (country_id, language_id, is_official) VALUES
    ('11111111-1111-1111-1111-111111111111', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', true),
    ('22222222-2222-2222-2222-222222222222', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', true),
    ('22222222-2222-2222-2222-222222222222', 'cccccccc-cccc-cccc-cccc-cccccccccccc', true),
    ('33333333-3333-3333-3333-333333333333', 'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb', true);
