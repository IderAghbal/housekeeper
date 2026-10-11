CREATE TABLE `test_projections` (
    `id`        UInt64,
    `name`      String,
    `timestamp` DateTime,
    PROJECTION `by_time` (SELECT *
ORDER BY `timestamp`)
)
ENGINE = MergeTree()
ORDER BY `id`;
