CREATE TABLE `staged_logs` (
    `id` UInt64,
    `ts` DateTime
)
ENGINE = MergeTree()
ORDER BY `id`
TTL `ts` + INTERVAL 7 DAY RECOMPRESS CODEC(ZSTD(3)), `ts` + INTERVAL 30 DAY TO VOLUME 'cold', `ts` + INTERVAL 1 YEAR DELETE;
