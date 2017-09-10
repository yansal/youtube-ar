CREATE TABLE jobs (
    id serial PRIMARY KEY,
    url text NOT NULL,
    started_at timestamp with time zone DEFAULT now() NOT NULL,
    downloaded_at timestamp with time zone,
    uploaded_at timestamp with time zone,
    output text,
    error text,
    retries integer DEFAULT 0 NOT NULL,
    geoip jsonb,
    torlog text
);

-- Forbid more than one running job per url
CREATE UNIQUE INDEX jobs_url_idx ON jobs (url) WHERE ((error IS NULL) AND (uploaded_at IS NULL));

-- Notify worker after insert
CREATE FUNCTION
  f()
  RETURNS trigger
  AS $$
  BEGIN
    PERFORM pg_notify('job', json_build_object(
        'id', NEW.id,
        'retries', NEW.retries,
        'url', NEW.url
    )::text);
    RETURN NEW;
  END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER tr AFTER INSERT
  ON jobs
  FOR EACH ROW
  EXECUTE PROCEDURE f();