update jobs
    set output = $1, error = $2
    where id = $3;