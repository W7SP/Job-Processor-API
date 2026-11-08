CREATE TABLE jobs (
    id UUID PRIMARY KEY,
    status TEXT NOT NULL CHECK (status IN ('pending', 'running', 'done', 'failed')),
    payload TEXT NOT NULL
);
