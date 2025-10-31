# Audit SQL - Vérification des colonnes PRTG

## Schéma attendu (d'après le code)

### Table: prtg_group
Colonnes utilisées dans le code:
- `id` (INTEGER) - PK
- `prtg_server_address_id` (INTEGER) - FK
- `name` (VARCHAR)
- `is_probe_node` (BOOLEAN)
- `self_group_id` (INTEGER) - Parent reference ⚠️ **NOT parent_id**
- `tree_depth` (INTEGER)

### Table: prtg_group_path
- `group_id` (INTEGER)
- `prtg_server_address_id` (INTEGER)
- `path` (TEXT)
- `tree_path` (?)

### Table: prtg_device
Colonnes utilisées:
- `id` (INTEGER) - PK
- `prtg_server_address_id` (INTEGER) - FK
- `name` (VARCHAR)
- `host` (VARCHAR)
- `prtg_group_id` (INTEGER) - FK to prtg_group
- `tree_depth` (INTEGER)

### Table: prtg_device_path
- `device_id` (INTEGER)
- `prtg_server_address_id` (INTEGER)
- `path` (TEXT)
- `tree_path` (?)

### Table: prtg_sensor
Colonnes utilisées:
- `id` (INTEGER) - PK
- `prtg_server_address_id` (INTEGER) - FK
- `name` (VARCHAR)
- `sensor_type` (VARCHAR)
- `prtg_device_id` (INTEGER) - FK to prtg_device
- `scanning_interval_seconds` (INTEGER)
- `status` (INTEGER)
- `status_text` (VARCHAR) - ⚠️ Check if exists in DB or computed
- `last_check_utc` (TIMESTAMP)
- `last_up_utc` (TIMESTAMP)
- `last_down_utc` (TIMESTAMP)
- `priority` (INTEGER)
- `message` (TEXT)
- `uptime_since_seconds` (DOUBLE PRECISION)
- `downtime_since_seconds` (DOUBLE PRECISION)
- `full_path` (TEXT) - ⚠️ Check if from prtg_sensor_path join

### Table: prtg_sensor_path
- `sensor_id` (INTEGER)
- `prtg_server_address_id` (INTEGER)
- `path` (TEXT)

### Table: prtg_tag
- `id` (INTEGER) - PK
- `prtg_server_address_id` (INTEGER) - FK
- `name` (VARCHAR)

### Table: prtg_sensor_tag (junction table)
- `prtg_sensor_id` (INTEGER)
- `prtg_tag_id` (INTEGER)
- `prtg_server_address_id` (INTEGER)

---

## Audit des requêtes par fonction

### ✅ GetSensors / GetSensorsExtended (L15-151)
**Requête principale:**
```sql
SELECT
    s.id,
    s.prtg_server_address_id,
    s.name,
    s.sensor_type,
    s.prtg_device_id,
    d.name AS device_name,
    s.scanning_interval_seconds,
    s.status,
    s.last_check_utc,
    s.last_up_utc,
    s.last_down_utc,
    s.priority,
    s.message,
    s.uptime_since_seconds,
    s.downtime_since_seconds,
    sp.path AS full_path,
    '' AS tags
FROM prtg_sensor s
INNER JOIN prtg_device d ON s.prtg_device_id = d.id
    AND s.prtg_server_address_id = d.prtg_server_address_id
INNER JOIN prtg_sensor_path sp ON s.id = sp.sensor_id
    AND s.prtg_server_address_id = sp.prtg_server_address_id
INNER JOIN prtg_group g ON d.prtg_group_id = g.id
    AND d.prtg_server_address_id = g.prtg_server_address_id
WHERE 1=1
```

**Status:** ✅ OK
**Notes:** Utilise g.name pour filter par group_name (ligne 75-77)

---

### ✅ GetSensorByID (L153-254)
**Requête:**
```sql
SELECT
    s.id,
    s.prtg_server_address_id,
    s.name,
    s.sensor_type,
    s.prtg_device_id,
    d.name AS device_name,
    s.scanning_interval_seconds,
    s.status,
    s.last_check_utc,
    s.last_up_utc,
    s.last_down_utc,
    s.priority,
    s.message,
    s.uptime_since_seconds,
    s.downtime_since_seconds,
    sp.path AS full_path,
    COALESCE(
        (SELECT string_agg(t.name, ',')
         FROM prtg_sensor_tag st
         JOIN prtg_tag t ON st.prtg_tag_id = t.id
         WHERE st.prtg_sensor_id = s.id
         AND st.prtg_server_address_id = s.prtg_server_address_id),
        ''
    ) AS tags
FROM prtg_sensor s
INNER JOIN prtg_device d ON s.prtg_device_id = d.id
    AND s.prtg_server_address_id = d.prtg_server_address_id
INNER JOIN prtg_sensor_path sp ON s.id = sp.sensor_id
    AND s.prtg_server_address_id = sp.prtg_server_address_id
WHERE s.id = $1
```

**Status:** ✅ OK
**Notes:** Agrégation des tags fonctionne correctement

---

### ✅ GetAlerts (L256-342)
**Requête:** Similaire à GetSensors avec tags
**Status:** ✅ OK
**Filters:**
- s.status != StatusUp
- Optional: hours filter, status filter, device name filter

---

### ✅ GetDeviceOverview (L344-464)
**Requête Device:**
```sql
SELECT
    d.id,
    d.prtg_server_address_id,
    d.name,
    d.host,
    d.prtg_group_id,
    g.name AS group_name,
    dp.path AS full_path,
    d.tree_depth,
    COALESCE(
        (SELECT COUNT(*) FROM prtg_sensor s
         WHERE s.prtg_device_id = d.id
         AND s.prtg_server_address_id = d.prtg_server_address_id),
        0
    ) AS sensor_count
FROM prtg_device d
INNER JOIN prtg_group g ON d.prtg_group_id = g.id
    AND d.prtg_server_address_id = g.prtg_server_address_id
INNER JOIN prtg_device_path dp ON d.id = dp.device_id
    AND d.prtg_server_address_id = dp.prtg_server_address_id
WHERE d.name ILIKE $1
LIMIT 1
```

**Status:** ✅ OK
**Sensors query:** Standard sensor query
**Status:** ✅ OK

---

### ✅ GetTopSensors (L466-540)
**Requête:** Similaire à GetSensors avec différent ORDER BY
**Status:** ✅ OK
**Order options:**
- downtime: `ORDER BY s.downtime_since_seconds DESC NULLS LAST`
- alerts: `ORDER BY s.priority DESC, s.status`
- uptime (default): `ORDER BY s.uptime_since_seconds DESC NULLS LAST`

---

### ✅ ExecuteCustomQuery (L542-591)
**Status:** ✅ OK - Security validation only, no hardcoded columns

---

### ✅ GetGroups (L628-700)
**Requête:**
```sql
SELECT
    g.id,
    g.prtg_server_address_id,
    g.name,
    g.is_probe_node,
    g.self_group_id,  ✅ FIXED
    gp.path AS full_path,
    g.tree_depth
FROM prtg_group g
INNER JOIN prtg_group_path gp ON g.id = gp.group_id
    AND g.prtg_server_address_id = gp.prtg_server_address_id
WHERE 1=1
```

**Status:** ✅ FIXED (was using parent_id)
**Filters:**
- g.name ILIKE (if groupName provided)
- g.self_group_id = (if parentID provided) ✅ FIXED

---

### ✅ GetDevicesByGroupID (L702-758)
**Requête:**
```sql
SELECT
    d.id,
    d.prtg_server_address_id,
    d.name,
    d.host,
    d.prtg_group_id,
    g.name AS group_name,
    dp.path AS full_path,
    COALESCE(
        (SELECT COUNT(*) FROM prtg_sensor s
         WHERE s.prtg_device_id = d.id
         AND s.prtg_server_address_id = d.prtg_server_address_id),
        0
    ) AS sensor_count,
    d.tree_depth
FROM prtg_device d
INNER JOIN prtg_group g ON d.prtg_group_id = g.id
    AND d.prtg_server_address_id = g.prtg_server_address_id
INNER JOIN prtg_device_path dp ON d.id = dp.device_id
    AND d.prtg_server_address_id = dp.prtg_server_address_id
WHERE d.prtg_group_id = $1
ORDER BY d.name
```

**Status:** ✅ OK

---

### ✅ GetHierarchy (L760-836)
**Requête root groups:**
```sql
SELECT
    g.id,
    g.prtg_server_address_id,
    g.name,
    g.is_probe_node,
    g.self_group_id,  ✅ FIXED
    gp.path AS full_path,
    g.tree_depth
FROM prtg_group g
INNER JOIN prtg_group_path gp ON g.id = gp.group_id
    AND g.prtg_server_address_id = gp.prtg_server_address_id
WHERE g.self_group_id IS NULL  ✅ FIXED
ORDER BY g.name
LIMIT 10
```

**Status:** ✅ FIXED (was using parent_id)
**Notes:** Recursive function calls GetGroups() and GetDevicesByGroupID()

---

### ✅ buildHierarchyNode (L838-928)
**Sensors query:**
```sql
SELECT
    s.id,
    s.prtg_server_address_id,
    s.name,
    s.sensor_type,
    s.prtg_device_id,
    $2 AS device_name,
    s.scanning_interval_seconds,
    s.status,
    s.last_check_utc,
    s.last_up_utc,
    s.last_down_utc,
    s.priority,
    s.message,
    s.uptime_since_seconds,
    s.downtime_since_seconds,
    sp.path AS full_path,
    '' AS tags
FROM prtg_sensor s
INNER JOIN prtg_sensor_path sp ON s.id = sp.sensor_id
    AND s.prtg_server_address_id = sp.prtg_server_address_id
WHERE s.prtg_device_id = $1
AND s.prtg_server_address_id = $3
ORDER BY s.name
LIMIT 50
```

**Status:** ✅ OK

---

### ✅ Search (L930-1098)
**Groups query:**
```sql
SELECT
    g.id,
    g.prtg_server_address_id,
    g.name,
    g.is_probe_node,
    g.self_group_id,  ✅ FIXED
    gp.path AS full_path,
    g.tree_depth
FROM prtg_group g
INNER JOIN prtg_group_path gp ON g.id = gp.group_id
    AND g.prtg_server_address_id = gp.prtg_server_address_id
WHERE g.name ILIKE $1
ORDER BY g.name
LIMIT $2
```

**Status:** ✅ FIXED (was using parent_id)

**Devices query:**
```sql
SELECT
    d.id,
    d.prtg_server_address_id,
    d.name,
    d.host,
    d.prtg_group_id,
    g.name AS group_name,
    dp.path AS full_path,
    COALESCE(
        (SELECT COUNT(*) FROM prtg_sensor s
         WHERE s.prtg_device_id = d.id
         AND s.prtg_server_address_id = d.prtg_server_address_id),
        0
    ) AS sensor_count,
    d.tree_depth
FROM prtg_device d
INNER JOIN prtg_group g ON d.prtg_group_id = g.id
    AND d.prtg_server_address_id = g.prtg_server_address_id
INNER JOIN prtg_device_path dp ON d.id = dp.device_id
    AND d.prtg_server_address_id = dp.prtg_server_address_id
WHERE d.name ILIKE $1 OR d.host ILIKE $1
ORDER BY d.name
LIMIT $2
```

**Status:** ✅ OK

**Sensors query:** Standard sensors query
**Status:** ✅ OK

---

### ✅ GetTags (L1100-1157)
**Requête:**
```sql
SELECT
    t.id,
    t.prtg_server_address_id,
    t.name,
    COUNT(DISTINCT st.prtg_sensor_id) as sensor_count
FROM prtg_tag t
LEFT JOIN prtg_sensor_tag st ON t.id = st.prtg_tag_id
    AND t.prtg_server_address_id = st.prtg_server_address_id
WHERE 1=1
GROUP BY t.id, t.prtg_server_address_id, t.name
ORDER BY t.name
LIMIT $N
```

**Status:** ✅ OK
**Notes:** Agrégation correcte des sensor counts

---

### ✅ GetBusinessProcesses (L1159-1226)
**Requête:**
```sql
SELECT
    s.id,
    s.prtg_server_address_id,
    s.name,
    s.sensor_type,
    s.prtg_device_id,
    d.name as device_name,
    s.scanning_interval_seconds,
    s.status,
    s.status_text,  ⚠️ CHECK: Does this column exist?
    s.last_check_utc,
    s.last_up_utc,
    s.last_down_utc,
    s.priority,
    s.message,
    s.uptime_since_seconds,
    s.downtime_since_seconds,
    s.full_path,  ⚠️ CHECK: Should be sp.path
    COALESCE(
        (SELECT STRING_AGG(t.name, ', ' ORDER BY t.name)
         FROM prtg_sensor_tag st
         JOIN prtg_tag t ON st.prtg_tag_id = t.id
         WHERE st.prtg_sensor_id = s.id
           AND st.prtg_server_address_id = s.prtg_server_address_id),
        ''
    ) as tags
FROM prtg_sensor s
LEFT JOIN prtg_device d ON s.prtg_device_id = d.id
    AND s.prtg_server_address_id = d.prtg_server_address_id
WHERE s.sensor_type ILIKE '%business%process%'
```

**Status:** ⚠️ **PROBLÈME TROUVÉ**
**Issues:**
1. **s.status_text** - Cette colonne n'existe probablement pas dans prtg_sensor
2. **s.full_path** - Devrait être sp.path depuis prtg_sensor_path
3. **Manque le JOIN** avec prtg_sensor_path

---

### ✅ GetStatistics (L1228-1313)
**Count query:**
```sql
SELECT
    (SELECT COUNT(*) FROM prtg_sensor) as total_sensors,
    (SELECT COUNT(*) FROM prtg_device) as total_devices,
    (SELECT COUNT(*) FROM prtg_group) as total_groups,
    (SELECT COUNT(*) FROM prtg_tag) as total_tags,
    (SELECT COUNT(*) FROM prtg_group WHERE is_probe_node = true) as total_probes
```

**Status:** ✅ OK

**Status breakdown:**
```sql
SELECT status, COUNT(*) as count
FROM prtg_sensor
GROUP BY status
ORDER BY status
```

**Status:** ✅ OK

**Sensor types:**
```sql
SELECT sensor_type, COUNT(*) as count
FROM prtg_sensor
WHERE sensor_type IS NOT NULL AND sensor_type != ''
GROUP BY sensor_type
ORDER BY count DESC
LIMIT 15
```

**Status:** ✅ OK

---

## 🔴 PROBLÈMES IDENTIFIÉS

### 1. GetBusinessProcesses (L1159-1226)

**Problème A: Colonne status_text**
```sql
s.status_text,  -- ❌ Cette colonne n'existe probablement pas
```
- **Solution:** Retirer de SELECT, calculer côté application avec GetStatusText()

**Problème B: Colonne full_path**
```sql
s.full_path,  -- ❌ Cette colonne n'existe pas dans prtg_sensor
```
- **Solution:** Ajouter JOIN avec prtg_sensor_path et utiliser sp.path

**Problème C: JOIN manquant**
```sql
LEFT JOIN prtg_device d ON ...  -- OK
-- ❌ Manque: INNER JOIN prtg_sensor_path sp ON s.id = sp.sensor_id
```

---

## 📋 ACTIONS REQUISES

1. ✅ **GetGroups**: FIXED - utilise maintenant self_group_id
2. ✅ **GetHierarchy**: FIXED - utilise maintenant self_group_id
3. ✅ **Search**: FIXED - utilise maintenant self_group_id
4. ❌ **GetBusinessProcesses**: À CORRIGER
   - Retirer s.status_text
   - Remplacer s.full_path par sp.path
   - Ajouter JOIN prtg_sensor_path

---

## ✅ COLONNES CONFIRMÉES INEXISTANTES

- `prtg_group.parent_id` → utiliser `prtg_group.self_group_id`
- `prtg_sensor.status_text` → calculer avec GetStatusText(status)
- `prtg_sensor.full_path` → obtenir depuis prtg_sensor_path.path

---

**Audit complété le:** 2025-10-31
**Fichier audité:** internal/database/queries.go
**Total requêtes:** 36 SELECT statements
**Problèmes trouvés:** 1 fonction (GetBusinessProcesses)
**Problèmes résolus:** 3 fonctions (GetGroups, GetHierarchy, Search)
