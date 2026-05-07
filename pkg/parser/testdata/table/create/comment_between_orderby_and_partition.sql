CREATE TABLE `t` (
    `id` UInt64,
    `ts` DateTime
)
ENGINE = MergeTree()
ORDER BY `id`
-- partition by month so retention drops by partition
PARTITION BY toYYYYMM(`ts`);
