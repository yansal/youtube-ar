begin;

create table urls (
    id serial primary key,
    url text not null,
    created_at timestamp with time zone not null default now(),
    updated_at timestamp with time zone not null default now(),
    logs text[],
    status text not null default 'pending',
    error text,
    file text,
    retries int
);

create function urls_update() returns trigger as $urls_update$
    begin
        NEW.updated_at := current_timestamp;
        return NEW;
    end;
$urls_update$ language plpgsql;

create trigger urls_update before update on urls
    for each row execute procedure urls_update();

create table youtube_videos (
    id serial primary key,
    youtube_id text not null unique,
    created_at timestamp with time zone not null default now()
);

commit;