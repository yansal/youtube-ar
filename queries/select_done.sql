select
    id, url, uploaded_at, retries, geoip->>'ip' as ip, geoip->>'country_name' as country
    from jobs
    where status = 'done'
    order by started_at desc
    limit 25;