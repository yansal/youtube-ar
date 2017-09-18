update jobs
    set status = 'done', uploaded_at = $1
    where id = $2;