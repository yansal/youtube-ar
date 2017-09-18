update jobs
    set status = 'error', output = $1, error = $2
    where id = $3;