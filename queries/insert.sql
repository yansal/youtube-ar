insert into jobs(url)
    values($1)
    returning id;