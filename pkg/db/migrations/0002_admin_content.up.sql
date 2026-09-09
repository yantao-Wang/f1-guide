-- 内容管理后台支持（W8 提前交付，原二期）
-- 1) 车手表增加 Jolpica driverId：后台录入后，积分榜/赛程页可动态挂接站内故事链接
-- 2) 车手表增加照片路径：后台上传车手照片，前台详情页渲染官方风格卡片
-- 3) 赛道删除时名场面自动解绑（原 FK 默认 NO ACTION 会阻塞删除）

ALTER TABLE drivers ADD COLUMN jolpica_id TEXT UNIQUE;
COMMENT ON COLUMN drivers.jolpica_id IS '上游 Jolpica driverId（如 max_verstappen），用于积分榜映射站内 slug；为空表示不参与映射';

ALTER TABLE drivers ADD COLUMN image_url TEXT;
COMMENT ON COLUMN drivers.image_url IS '车手照片路径（后台 /uploads 上传），为空时前台降级为纯 CSS 车队色卡片';

ALTER TABLE moments DROP CONSTRAINT moments_track_id_fkey;
ALTER TABLE moments ADD CONSTRAINT moments_track_id_fkey
    FOREIGN KEY (track_id) REFERENCES tracks(id) ON DELETE SET NULL;
