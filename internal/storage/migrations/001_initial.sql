CREATE TABLE trainers (
    id INTEGER PRIMARY KEY,
    display_name TEXT NOT NULL,
    normalized_name TEXT NOT NULL UNIQUE,
    created_at_ms INTEGER NOT NULL
);

CREATE TABLE app_state (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    active_trainer_id INTEGER REFERENCES trainers(id) ON DELETE RESTRICT
);
INSERT INTO app_state (id, active_trainer_id) VALUES (1, NULL);

CREATE TABLE trainer_progress (
    trainer_id INTEGER PRIMARY KEY REFERENCES trainers(id) ON DELETE RESTRICT,
    xp_total INTEGER NOT NULL DEFAULT 0 CHECK (xp_total >= 0)
);

CREATE TABLE encounters (
    id INTEGER PRIMARY KEY,
    trainer_id INTEGER NOT NULL REFERENCES trainers(id) ON DELETE RESTRICT,
    species_id INTEGER NOT NULL CHECK (species_id > 0),
    form_id TEXT NOT NULL,
    gender_key TEXT NOT NULL CHECK (gender_key IN ('default', 'male', 'female')),
    palette TEXT NOT NULL CHECK (palette IN ('regular', 'shiny')),
    encountered_at_ms INTEGER NOT NULL,
    dataset_id TEXT NOT NULL,
    species_name_snapshot TEXT NOT NULL,
    form_name_snapshot TEXT NOT NULL,
    type1_snapshot TEXT NOT NULL,
    type2_snapshot TEXT,
    regional_snapshot INTEGER NOT NULL CHECK (regional_snapshot IN (0, 1)),
    transformation_snapshot INTEGER NOT NULL CHECK (transformation_snapshot IN (0, 1)),
    first_species INTEGER NOT NULL CHECK (first_species IN (0, 1)),
    first_variant INTEGER NOT NULL CHECK (first_variant IN (0, 1)),
    xp_awarded INTEGER NOT NULL CHECK (xp_awarded >= 0)
);

CREATE INDEX encounters_trainer_recent
    ON encounters (trainer_id, encountered_at_ms DESC, id DESC);
CREATE INDEX encounters_trainer_species
    ON encounters (trainer_id, species_id);

CREATE TABLE species_discoveries (
    trainer_id INTEGER NOT NULL REFERENCES trainers(id) ON DELETE RESTRICT,
    species_id INTEGER NOT NULL CHECK (species_id > 0),
    first_seen_ms INTEGER NOT NULL,
    last_seen_ms INTEGER NOT NULL,
    encounter_count INTEGER NOT NULL CHECK (encounter_count > 0),
    PRIMARY KEY (trainer_id, species_id)
);

CREATE TABLE variant_discoveries (
    trainer_id INTEGER NOT NULL,
    species_id INTEGER NOT NULL,
    form_id TEXT NOT NULL,
    gender_key TEXT NOT NULL CHECK (gender_key IN ('default', 'male', 'female')),
    palette TEXT NOT NULL CHECK (palette IN ('regular', 'shiny')),
    first_seen_ms INTEGER NOT NULL,
    last_seen_ms INTEGER NOT NULL,
    encounter_count INTEGER NOT NULL CHECK (encounter_count > 0),
    PRIMARY KEY (trainer_id, species_id, form_id, gender_key, palette),
    FOREIGN KEY (trainer_id, species_id)
        REFERENCES species_discoveries(trainer_id, species_id) ON DELETE RESTRICT
);

CREATE TABLE achievement_unlocks (
    trainer_id INTEGER NOT NULL REFERENCES trainers(id) ON DELETE RESTRICT,
    achievement_id TEXT NOT NULL,
    unlocked_at_ms INTEGER NOT NULL,
    dataset_id TEXT NOT NULL,
    target_at_unlock INTEGER,
    PRIMARY KEY (trainer_id, achievement_id)
);
