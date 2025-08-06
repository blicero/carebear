-- /home/krylon/go/src/github.com/blicero/carebear/database/uptime_recent.sql
-- Time-stamp: <2025-08-06 17:52:49 krylon>
-- created on 06. 08. 2025 by Benjamin Walkenhorst
-- (c) 2025 Benjamin Walkenhorst
-- Use at your own risk!

WITH recent AS (
    SELECT
        id,
        dev_id,
        timestamp,
        load1,
        load5,
        load15,
        ROW_NUMBER() OVER (PARTITION BY dev_id ORDER BY timestamp DESC) AS up_no
    FROM uptime
)

SELECT
    r.id,
    d.name, -- dev_id,
    datetime(r.timestamp, 'unixepoch') AS timestamp,
    r.load1,
    r.load5,
    r.load15
FROM recent r
INNER JOIN device d ON d.id = r.dev_id
WHERE up_no = 1
ORDER BY d.name
