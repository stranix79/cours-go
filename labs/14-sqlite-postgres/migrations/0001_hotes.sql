-- Migration 1 : la table des hôtes.
-- Écrite dans le SQL commun à SQLite et PostgreSQL : TEXT, TIMESTAMP, pas de
-- SERIAL ni d'AUTOINCREMENT (le nom d'hôte sert de clé primaire).
CREATE TABLE hotes (
    nom        TEXT PRIMARY KEY,
    ip         TEXT NOT NULL,
    role       TEXT NOT NULL DEFAULT '',
    dernier_vu TIMESTAMP              -- NULL tant que l'hôte n'a jamais répondu
);
