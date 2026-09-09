-- 回滚 0002：恢复原外键行为，删除后台新增列

ALTER TABLE moments DROP CONSTRAINT moments_track_id_fkey;
ALTER TABLE moments ADD CONSTRAINT moments_track_id_fkey
    FOREIGN KEY (track_id) REFERENCES tracks(id);

ALTER TABLE drivers DROP COLUMN image_url;
ALTER TABLE drivers DROP COLUMN jolpica_id;
