insert into jobs(url, retries)
    values($1, $2)
    returning id;