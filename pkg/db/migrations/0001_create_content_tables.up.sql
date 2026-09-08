-- 内容三表：车手 / 赛道 / 名场面（对应 api/openapi.yaml 组件模型）

CREATE TABLE drivers (
    id                  BIGSERIAL PRIMARY KEY,
    slug                TEXT NOT NULL UNIQUE,
    name                TEXT NOT NULL,
    tagline             TEXT NOT NULL,
    team_name           TEXT NOT NULL,
    team_color          TEXT NOT NULL,
    number              INT  NOT NULL,
    championships       INT  NOT NULL DEFAULT 0,
    country             TEXT NOT NULL,
    story               TEXT NOT NULL,
    personality         TEXT[] NOT NULL DEFAULT '{}',
    trivia              TEXT[] NOT NULL DEFAULT '{}',
    quote               TEXT NOT NULL,
    featured            BOOLEAN NOT NULL DEFAULT FALSE,
    featured_race_year  INT,
    featured_race_gp    TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE tracks (
    id               BIGSERIAL PRIMARY KEY,
    slug             TEXT NOT NULL UNIQUE,
    name             TEXT NOT NULL,
    country          TEXT NOT NULL,
    tagline          TEXT NOT NULL,
    type             TEXT NOT NULL CHECK (type IN ('permanent', 'street')),
    first_grand_prix INT  NOT NULL,
    length_km        NUMERIC(6,3) NOT NULL,
    laps             INT  NOT NULL,
    highlights       TEXT[] NOT NULL DEFAULT '{}',
    circuit_map_url  TEXT,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE moments (
    id          BIGSERIAL PRIMARY KEY,
    slug        TEXT NOT NULL UNIQUE,
    title       TEXT NOT NULL,
    year        INT  NOT NULL,
    grand_prix  TEXT NOT NULL,
    type        TEXT NOT NULL CHECK (type IN ('championship', 'overtake', 'safety', 'rain')),
    background  TEXT NOT NULL,
    why_classic TEXT NOT NULL,
    video_url   TEXT,
    track_id    BIGINT REFERENCES tracks(id),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- 名场面 ↔ 车手 多对多；名场面 ↔ 赛道 一对多（moments.track_id）
CREATE TABLE moment_drivers (
    moment_id BIGINT NOT NULL REFERENCES moments(id) ON DELETE CASCADE,
    driver_id BIGINT NOT NULL REFERENCES drivers(id) ON DELETE CASCADE,
    PRIMARY KEY (moment_id, driver_id)
);

-- 首页精选过滤
CREATE INDEX idx_drivers_featured ON drivers (featured) WHERE featured;
-- 名场面按年份倒序
CREATE INDEX idx_moments_year ON moments (year DESC);
-- 反查车手关联的名场面
CREATE INDEX idx_moment_drivers_driver ON moment_drivers (driver_id);
