CREATE TABLE `files` (
    `id`          UInt64,
    `search_text` String,
    `path`        String,
    INDEX `search_text_index` `search_text` TYPE text(`tokenizer` = `splitByNonAlpha`) GRANULARITY 100000000,
    INDEX `path_index` `path` TYPE text(`tokenizer` = splitByString(['/', '.']), `dictionary_block_size` = 512) GRANULARITY 1
)
ENGINE = MergeTree()
ORDER BY `id`;
