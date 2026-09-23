-- Migration 2 : un index pour les listes filtrées par rôle.
-- Une migration = un fichier numéroté, appliqué une seule fois, jamais modifié
-- après coup : pour changer le schéma, on ajoute un 0003_....sql.
CREATE INDEX idx_hotes_role ON hotes (role);
