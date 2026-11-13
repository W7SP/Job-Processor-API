CREATE TABLE bikes (
      id UUID PRIMARY KEY,
      bike_type TEXT NOT NULL CHECK (bike_type IN ('road', 'gravel', 'XC', 'trail', 'enduro', 'downhill')),
      build_status TEXT NOT NULL CHECK (build_status IN ('in assembly...', 'assembled', 'shipped', 'delivered'))
);