CREATE TABLE IF NOT EXISTS ngos (
    id SERIAL PRIMARY KEY,
    name VARCHAR(150) NOT NULL,
    email VARCHAR(100) UNIQUE NOT NULL,
    cause VARCHAR(100) NOT NULL, -- Ex: Proteção Animal, Educação, Fome
    city VARCHAR(100) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO ngos (name, email, cause, city) VALUES
('Anjos de Patas', 'contato@anjosdepatas.org', 'Proteção Animal', 'Osasco'),
('Educa Mais', 'info@educamais.org', 'Educação', 'São Paulo');
