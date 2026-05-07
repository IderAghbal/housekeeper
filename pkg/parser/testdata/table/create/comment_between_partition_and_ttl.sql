CREATE TABLE `t` (
    `id` UInt64,
    `ts` DateTime
)
ENGINE = MergeTree()
ORDER BY `id`
PARTITION BY toYYYYMM(`ts`)
-- delete after a year
TTL `ts` + INTERVAL 1 YEAR;
