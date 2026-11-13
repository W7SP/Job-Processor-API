-- To apply a migration run: Get-Content -Raw migrations/002_create_bikes.sql | docker exec -i job-processor-api-postgres-1 psql -U postgres -d jobs_db -v ON_ERROR_STOP=1
CREATE TABLE jobs (
    id UUID PRIMARY KEY,
    status TEXT NOT NULL CHECK (status IN ('pending', 'running', 'done', 'failed')),
    payload TEXT NOT NULL
);
