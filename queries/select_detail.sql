select
    url, started_at, error, output, torlog
    from jobs
    where id = $1;