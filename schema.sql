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

CREATE UNIQUE INDEX jobs_url_idx ON jobs (url) WHERE ((error IS NULL) AND (uploaded_at IS NULL));