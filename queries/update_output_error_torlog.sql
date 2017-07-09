update jobs
    set output = $1, error = $2, torlog = $3
    where id = $4;