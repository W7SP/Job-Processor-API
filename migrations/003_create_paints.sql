CREATE TABLE paints (
       id UUID PRIMARY KEY,
       color TEXT NOT NULL CHECK (color IN ('red', 'green', 'blue')),
       build_status TEXT NOT NULL
);