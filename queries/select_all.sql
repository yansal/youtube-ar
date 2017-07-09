select
    id, url, started_at, downloaded_at, uploaded_at, output, error, retries, geoip->>'ip' as ip, geoip->>'country_name' as country
    from jobs
    order by started_at desc
    limit 100;