CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TYPE user_role AS ENUM ('user', 'admin');

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    nickname TEXT NOT NULL UNIQUE CHECK (LENGTH(nickname) BETWEEN 4 and 20),
    password_hash TEXT NOT NULL,
    role user_role NOT NULL DEFAULT 'user',
    xp INT NOT NULL DEFAULT 0 CHECK (xp >= 0),
    created_at TIMESTAMP NOT NULL DEFAULT now()
);

CREATE INDEX idx_users_created_at ON users(created_at);
CREATE INDEX idx_users_admins ON users(role) WHERE role = 'admin';  -- частичный индекс
CREATE INDEX idx_users_xp ON users(xp DESC);

CREATE TABLE rooms (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL DEFAULT 'custom_room' CHECK (LENGTH(name) BETWEEN 4 AND 30),
    owner_id UUID NOT NULL REFERENCES users(id),
    max_players INT NOT NULL DEFAULT 4,
    active_players INT NOT NULL DEFAULT 0 CHECK (active_players >= 0 AND active_players <= max_players),
    duration_minutes INT NOT NULL DEFAULT 5,
    gold_per_kill INT NOT NULL DEFAULT 10,
    fund_modifier DOUBLE PRECISION NOT NULL DEFAULT 1.0,
    created_at TIMESTAMP NOT NULL DEFAULT now()
);

CREATE INDEX idx_rooms_duration_minutes ON rooms(duration_minutes);  -- ← ДОБАВИЛ ;
CREATE INDEX idx_rooms_created_at ON rooms(created_at);              -- ← ДОБАВИЛ ;
CREATE INDEX idx_rooms_owner_id ON rooms(owner_id);
CREATE INDEX idx_rooms_game_params ON rooms(max_players, active_players, duration_minutes);

CREATE TYPE battle_status AS ENUM ('active', 'finished', 'cancelled');

CREATE TABLE battles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    room_id UUID NOT NULL REFERENCES rooms(id) ON DELETE CASCADE,
    started_at TIMESTAMP NOT NULL DEFAULT now(),
    ends_at TIMESTAMP NOT NULL CHECK (ends_at > started_at),
    finished_at TIMESTAMP NULL CHECK (finished_at >= started_at),
    status battle_status NOT NULL DEFAULT 'active',
    fund BIGINT NOT NULL DEFAULT 0 CHECK (fund >= 0)
);

CREATE INDEX idx_battles_room_id ON battles(room_id);
CREATE INDEX idx_battles_status ON battles(status);
CREATE INDEX idx_battles_ends_at ON battles(ends_at);
CREATE INDEX idx_battles_time_range ON battles(started_at, ends_at);

CREATE TABLE battle_players (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    battle_id UUID NOT NULL REFERENCES battles(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id),
    kills INT NOT NULL DEFAULT 0 CHECK (kills >= 0),
    deaths INT NOT NULL DEFAULT 0 CHECK (deaths >= 0),
    damage_dealt INT NOT NULL DEFAULT 0 CHECK (damage_dealt >= 0),
    damage_taken INT NOT NULL DEFAULT 0 CHECK (damage_taken >= 0),
    score INT NOT NULL DEFAULT 0,
    reward BIGINT NOT NULL DEFAULT 0 CHECK (reward >= 0),
    xp INT NOT NULL DEFAULT 0 CHECK (xp >= 0),
    joined_at TIMESTAMP NOT NULL DEFAULT now(),
    left_at TIMESTAMP NULL,
    UNIQUE(battle_id, user_id)
);

CREATE INDEX idx_battle_players_battle_id ON battle_players(battle_id);
CREATE INDEX idx_battle_players_user_id ON battle_players(user_id);
CREATE INDEX idx_battle_players_score ON battle_players(score DESC);
CREATE INDEX idx_battle_players_kills ON battle_players(kills DESC);

CREATE TABLE user_stats (
    user_id UUID PRIMARY KEY REFERENCES users(id),
    total_kills BIGINT NOT NULL DEFAULT 0,
    total_deaths BIGINT NOT NULL DEFAULT 0,
    total_damage_dealt BIGINT NOT NULL DEFAULT 0,
    total_damage_taken BIGINT NOT NULL DEFAULT 0,
    total_score BIGINT NOT NULL DEFAULT 0,
    total_reward BIGINT NOT NULL DEFAULT 0,
    total_xp BIGINT NOT NULL DEFAULT 0,
    battles_played INT NOT NULL DEFAULT 0,
    battles_completed INT NOT NULL DEFAULT 0,
    battles_won INT NOT NULL DEFAULT 0,
    avg_kills_per_battle DECIMAL(5,2) NOT NULL DEFAULT 0,
    avg_deaths_per_battle DECIMAL(5,2) NOT NULL DEFAULT 0,
    avg_damage_per_battle DECIMAL(8,2) NOT NULL DEFAULT 0,
    kd_ratio DECIMAL(5,2) NOT NULL DEFAULT 0,
    avg_battle_duration_minutes DECIMAL(5,2) NOT NULL DEFAULT 0,
    max_kills_in_battle INT NOT NULL DEFAULT 0,
    max_score_in_battle INT NOT NULL DEFAULT 0,
    max_reward_in_battle BIGINT NOT NULL DEFAULT 0,
    last_updated TIMESTAMP NOT NULL DEFAULT now()
);

-- Таблица связей игроков и комнат
CREATE TABLE IF NOT EXISTS room_players (
    room_id UUID NOT NULL REFERENCES rooms(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    joined_at TIMESTAMP NOT NULL DEFAULT NOW(),
    PRIMARY KEY (room_id, user_id)
);
CREATE INDEX IF NOT EXISTS idx_room_players_user_id ON room_players(user_id);
CREATE INDEX IF NOT EXISTS idx_room_players_joined_at ON room_players(joined_at);

--todo триггер потом либо перенесу либо хабь. хуй, посмотрим
-- CREATE OR REPLACE FUNCTION update_user_stats()
-- RETURNS TRIGGER AS $$
-- BEGIN
--     INSERT INTO user_stats (
--         user_id, total_kills, total_deaths, total_damage_dealt, 
--         total_damage_taken, total_score, total_reward, total_xp, battles_played
--     ) VALUES (
--         NEW.user_id, NEW.kills, NEW.deaths, NEW.damage_dealt,
--         NEW.damage_taken, NEW.score, NEW.reward, NEW.xp, 1
--     )
--     ON CONFLICT (user_id) 
--     DO UPDATE SET
--         total_kills = user_stats.total_kills + NEW.kills,
--         total_deaths = user_stats.total_deaths + NEW.deaths,
--         total_damage_dealt = user_stats.total_damage_dealt + NEW.damage_dealt,
--         total_damage_taken = user_stats.total_damage_taken + NEW.damage_taken,
--         total_score = user_stats.total_score + NEW.score,
--         total_reward = user_stats.total_reward + NEW.reward,
--         total_xp = user_stats.total_xp + NEW.xp,
--         battles_played = user_stats.battles_played + 1,
--         --battles_completed = user_stats.battles_completed + CASE WHEN NEW.left_at IS NOT NULL THEN 1 ELSE 0 END,
--         avg_kills_per_battle = (user_stats.total_kills + NEW.kills)::DECIMAL / (user_stats.battles_played + 1),
--         avg_deaths_per_battle = (user_stats.total_deaths + NEW.deaths)::DECIMAL / (user_stats.battles_played + 1),
--         avg_damage_per_battle = (user_stats.total_damage_dealt + NEW.damage_dealt)::DECIMAL / (user_stats.battles_played + 1),
--         kd_ratio = CASE 
--             WHEN user_stats.total_deaths + NEW.deaths = 0 THEN (user_stats.total_kills + NEW.kills)::DECIMAL 
--             ELSE (user_stats.total_kills + NEW.kills)::DECIMAL / (user_stats.total_deaths + NEW.deaths) 
--         END,
--         max_kills_in_battle = GREATEST(user_stats.max_kills_in_battle, NEW.kills),
--         max_score_in_battle = GREATEST(user_stats.max_score_in_battle, NEW.score),
--         max_reward_in_battle = GREATEST(user_stats.max_reward_in_battle, NEW.reward),
--         last_updated = now();
--     RETURN NEW;
-- END;
-- $$ LANGUAGE plpgsql;

-- CREATE TRIGGER battle_player_stats
--     AFTER INSERT ON battle_players
--     FOR EACH ROW
--     EXECUTE FUNCTION update_user_stats();