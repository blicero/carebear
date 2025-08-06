-- /home/krylon/go/src/github.com/blicero/carebear/database/update_recent.sql
-- Time-stamp: <2025-08-06 17:46:27 krylon>
-- created on 06. 08. 2025 by Benjamin Walkenhorst
-- (c) 2025 Benjamin Walkenhorst
-- Use at your own risk!

WITH recent AS (
    SELECT
        id,
        dev_id,
        timestamp,
        updates,
        ROW_NUMBER() OVER (PARTITION BY dev_id ORDER BY timestamp DESC) AS update_no
    FROM updates
)

SELECT
    id,
    dev_id,
    timestamp,
    updates
FROM recent WHERE update_no = 1
ORDER BY timestamp DESC;
