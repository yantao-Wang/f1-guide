-- 本地开发示例数据（占位内容，仅用于本地联调，不代表正式内容）
-- 正式内容统一经管理后台（/admin）在线录入发布，不再走 SQL 流水线
-- 用法：make seed（幂等，可重复执行）

INSERT INTO drivers (slug, name, tagline, team_name, team_color, number, championships, country, story, personality, trivia, quote, featured, featured_race_year, featured_race_gp, jolpica_id)
VALUES ('zhou-guanyu', '周冠宇', '让中国国旗第一次出现在 F1 积分区的人', '凯迪拉克', '#00A1E0', 24, 0, '中国', '【占位】示例故事正文', '{沉稳,坚韧}', '{【占位】示例冷知识}', '【占位】示例金句', TRUE, 2022, '巴林大奖赛', 'zhou'),
       ('max-verstappen', '马克斯·维斯塔潘', '【占位】示例人设', '红牛', '#1E41FF', 1, 4, '荷兰', '【占位】示例故事正文', '{果断,激进}', '{}', '【占位】示例金句', TRUE, NULL, NULL, 'max_verstappen')
ON CONFLICT (slug) DO NOTHING;

INSERT INTO tracks (slug, name, country, tagline, type, first_grand_prix, length_km, laps, highlights, circuit_map_url)
VALUES ('shanghai', '上海国际赛车场', '中国', '上字形的速度迷宫', 'permanent', 2004, 5.451, 56, '{一号弯,长直道}', NULL)
ON CONFLICT (slug) DO NOTHING;

INSERT INTO moments (slug, title, year, grand_prix, type, background, why_classic, video_url, track_id)
VALUES ('2021-abu-dhabi', '2021 阿布扎比 · 最后一圈决出世界冠军', 2021, '阿布扎比大奖赛', 'championship', '【占位】背景速览', '【占位】经典原因', 'https://www.bilibili.com/video/BV-example',
        (SELECT id FROM tracks WHERE slug = 'shanghai'))
ON CONFLICT (slug) DO NOTHING;

INSERT INTO moment_drivers (moment_id, driver_id)
SELECT m.id, d.id FROM moments m, drivers d WHERE m.slug = '2021-abu-dhabi' AND d.slug = 'zhou-guanyu'
ON CONFLICT DO NOTHING;
