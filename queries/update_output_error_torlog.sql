update jobs
    set status = 'error', output = $1, error = $2, torlog = $3
    where id = $4;