ALTER TABLE paints
    ALTER COLUMN prepared TYPE BOOLEAN
    USING (prepared::BOOLEAN);