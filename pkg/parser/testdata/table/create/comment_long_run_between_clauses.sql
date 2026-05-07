CREATE TABLE `t` (
    `id` UInt64,
    `ts` DateTime
)
ENGINE = MergeTree()
ORDER BY `id`
PARTITION BY toYYYYMM(`ts`)
-- comment line 1
-- comment line 2
-- comment line 3
-- comment line 4
-- comment line 5
-- comment line 6
TTL `ts` + INTERVAL 1 YEAR
SETTINGS index_granularity = 8192;
