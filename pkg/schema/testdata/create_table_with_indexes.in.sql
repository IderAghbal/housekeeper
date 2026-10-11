-- Current state: no table exists
;
-- Target state: a table with skip indexes, a full-text index, a constraint and a projection, which the CREATE keeps
CREATE TABLE files (
    id UInt64,
    owner_id UInt64,
    path String,
    body String,
    INDEX path_bloom path TYPE bloom_filter GRANULARITY 4,
    INDEX body_text body TYPE text(tokenizer = splitByNonAlpha) GRANULARITY 100000000,
    CONSTRAINT owned CHECK owner_id > 0,
    PROJECTION by_owner (SELECT owner_id, count() GROUP BY owner_id)
) ENGINE = MergeTree()
ORDER BY id;
