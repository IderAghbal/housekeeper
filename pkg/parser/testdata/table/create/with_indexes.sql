CREATE TABLE `search_logs` (
    `id`            UInt64,
    `query`         String,
    `user_id`       UInt64,
    `timestamp`     DateTime,
    `category`      LowCardinality(String),
    `response_time` Float32,
    INDEX `query_bloom` `query` TYPE bloom_filter GRANULARITY 1,
    INDEX `user_minmax` `user_id` TYPE minmax GRANULARITY 2,
    INDEX `category_set` `category` TYPE set(1000) GRANULARITY 1
)
ENGINE = MergeTree()
ORDER BY (`timestamp`, `user_id`)
PARTITION BY toYYYYMM(`timestamp`);
