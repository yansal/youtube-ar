select
    url, started_at, error, output, torlog, feed
    from jobs
    where id = $1;