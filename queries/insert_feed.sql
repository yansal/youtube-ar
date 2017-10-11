insert into jobs(url, feed)
    values($1, $2)
    returning id;