CREATE TABLE `t` (
    `id` UInt64,
    `ts` DateTime
)
ENGINE = MergeTree()
ORDER BY `id`
TTL `ts` + INTERVAL 1 YEAR
-- index every 8192 rows
SETTINGS index_granularity = 8192;
