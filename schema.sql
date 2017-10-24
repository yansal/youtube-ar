BEGIN;

CREATE TYPE job_status AS ENUM ('running', 'done', 'error');
CREATE TABLE jobs (
    id serial PRIMARY KEY,
    url text NOT NULL,
    status job_status DEFAULT 'running' NOT NULL,
    started_at timestamp with time zone DEFAULT now() NOT NULL,
    downloaded_at timestamp with time zone,
    uploaded_at timestamp with time zone,
    output text,
    error text,
    retries integer DEFAULT 0 NOT NULL,
    geoip jsonb,
    torlog text,
    feed xml,
    CONSTRAINT status_running CHECK (status <> 'running' OR (error IS NULL AND uploaded_at IS NULL)),
    CONSTRAINT status_done CHECK (status <> 'done' OR (error IS NULL AND output IS NOT NULL AND uploaded_at IS NOT NULL)),
    CONSTRAINT status_error CHECK (status <> 'error' OR (error IS NOT NULL AND uploaded_at IS NULL))
);

-- Forbid more than one running job per url
CREATE UNIQUE INDEX jobs_url_status_running_idx ON jobs (url) WHERE status = 'running';

-- Notify worker after insert
CREATE FUNCTION
  notify_jobs()
  RETURNS trigger
  AS $$
  BEGIN
    NOTIFY jobs;
    RETURN NULL;
  END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER notify_jobs AFTER INSERT
  ON jobs
  EXECUTE PROCEDURE notify_jobs();

COMMIT;