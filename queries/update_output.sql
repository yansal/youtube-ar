update jobs
    set output = $1, downloaded_at = $2
    where id = $3;