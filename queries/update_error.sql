update jobs
    set status = 'error', error = $1
    where id = $2;