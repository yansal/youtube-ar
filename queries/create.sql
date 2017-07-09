create table if not exists jobs (
    id serial primary key,
    url text not null,
    started_at timestamp with time zone not null default current_timestamp,
    downloaded_at timestamp with time zone,
    uploaded_at timestamp with time zone,
    output text,
    error text,
    retries int not null default 0,
    geoip jsonb,
    torlog text
);

create unique index if not exists jobs_url_idx on jobs
    ( url )
    where error is null and uploaded_at is null;