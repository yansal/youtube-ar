begin;

create type job_status as enum ('running', 'done', 'error');
create table jobs (
    id serial primary key,
    url text not null,
    status job_status default 'running' not null,
    started_at timestamp with time zone default now() not null,
    downloaded_at timestamp with time zone,
    uploaded_at timestamp with time zone,
    output text,
    error text,
    retries integer default 0 not null,
    geoip jsonb,
    torlog text,
    feed xml,
    constraint status_running check (status <> 'running' or (error is null and uploaded_at is null)),
    constraint status_done check (status <> 'done' or (error is null and output is not null and uploaded_at is not null)),
    constraint status_error check (status <> 'error' or (error is not null and uploaded_at is null))
);

-- forbid more than one running job per url
create unique index jobs_url_status_running_idx on jobs (url) where status = 'running';

-- notify worker after insert
create function
  notify_jobs()
  returns trigger
  as $$
  begin
    notify jobs;
    return null;
  end;
$$ language plpgsql;

create trigger notify_jobs after insert
  on jobs
  execute procedure notify_jobs();

create table oauth2_token ( token jsonb not null );

create table youtube_likes (
    id char(11) primary key,
    created_at timestamp with time zone default now() not null,
    check (length(id) >= 11)
);

create function insert_into_jobs_after_insert_into_youtube_likes()
  returns trigger
  as $$
  begin
    insert into jobs(url) values('https://www.youtube.com/watch?v=' || new.id);
    return null;
  end;
$$ language plpgsql;

create trigger insert_into_jobs_after_insert_into_youtube_likes after insert
  on youtube_likes
  for each row execute procedure insert_into_jobs_after_insert_into_youtube_likes();


commit;