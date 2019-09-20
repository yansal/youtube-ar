begin;

create table urls (
    id serial primary key,
    url text not null,
    created_at timestamp with time zone not null default now(),
    updated_at timestamp with time zone not null default now(),
    deleted_at timestamp with time zone,
    logs text[],
    status text not null default 'pending',
    error text,
    file text,
    retries int,
    oembed jsonb,
    tsv tsvector
);

create function urls_update() returns trigger as $urls_update$
    begin
        NEW.updated_at := current_timestamp;
        return NEW;
    end;
$urls_update$ language plpgsql;

create trigger urls_update before update on urls
    for each row execute procedure urls_update();

create function urls_update_tsv() returns trigger as $urls_update_tsv$
    begin
        NEW.tsv := to_tsvector(coalesce(new.oembed->>'title', '')) ||
            to_tsvector(coalesce(new.oembed->>'author_name', ''));
        return NEW;
    end
$urls_update_tsv$ LANGUAGE plpgsql;

create trigger urls_update_tsv before insert or update on urls
    for each row execute procedure urls_update_tsv();

create table youtube_videos (
    id serial primary key,
    youtube_id text not null unique,
    created_at timestamp with time zone not null default now()
);

commit;