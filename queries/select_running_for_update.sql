select
    id, url, retries
    from jobs
    where status = 'running'
    order by started_at
    limit 1
    for update skip locked;