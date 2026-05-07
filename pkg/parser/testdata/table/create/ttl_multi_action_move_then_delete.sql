CREATE TABLE `tiered_audit` (
    `id`             UInt64,
    `ts`             DateTime,
    `retention_days` UInt16
)
ENGINE = MergeTree()
ORDER BY `id`
TTL `ts` + INTERVAL 90 DAY TO VOLUME 'cold', `ts` + toIntervalDay(`retention_days`) DELETE;
