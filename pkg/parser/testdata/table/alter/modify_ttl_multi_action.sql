ALTER TABLE `analytics`.`events`
    MODIFY TTL `ts` + INTERVAL 90 DAY TO VOLUME 'cold', `ts` + INTERVAL 1 YEAR DELETE;
