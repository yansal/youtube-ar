update jobs
    set uploaded_at = $1
    where id = $2;