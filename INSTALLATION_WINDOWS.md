# Installation et Configuration - Windows

**Version:** v1.0.0
**Plateforme:** Windows 10/11 (64-bit)
**Taille:** 6.4 MB (executable) / 2.7 MB (ZIP)

---

## 📦 INSTALLATION RAPIDE (5 MINUTES)

### Étape 1 : Télécharger le Serveur MCP

1. **Télécharger le fichier ZIP** : `mcp-server-prtg_windows_amd64.zip`

2. **Décompresser** dans un dossier de votre choix :
   ```
   C:\Program Files\MCP-Server-PRTG\
   ```

3. **Contenu extrait** :
   ```
   C:\Program Files\MCP-Server-PRTG\
   └── mcp-server-prtg_windows_amd64.exe
   ```

### Étape 2 : Configuration de l'Environnement

#### Option A : Variables d'Environnement (Recommandé)

1. **Ouvrir les Variables d'Environnement** :
   - Appuyez sur `Win + R`
   - Tapez : `sysdm.cpl` et validez
   - Onglet **"Paramètres système avancés"**
   - Cliquez sur **"Variables d'environnement"**

2. **Créer les Variables Utilisateur** (cliquez sur **"Nouvelle..."** dans la section "Variables utilisateur") :

   | Variable | Valeur | Exemple |
   |----------|--------|---------|
   | `PRTG_DB_HOST` | Adresse de votre serveur PostgreSQL | `192.168.1.100` ou `prtg-db.entreprise.local` |
   | `PRTG_DB_PORT` | Port PostgreSQL | `5432` |
   | `PRTG_DB_NAME` | Nom de la base de données | `prtg_data_exporter` |
   | `PRTG_DB_USER` | Utilisateur PostgreSQL | `prtg_reader` |
   | `PRTG_DB_PASSWORD` | Mot de passe (**OBLIGATOIRE**) | `VotreMotDePasseSecurise` |
   | `PRTG_DB_SSLMODE` | Mode SSL (disable/require) | `disable` ou `require` |
   | `LOG_LEVEL` | Niveau de log (debug/info/warn/error) | `info` |

3. **Valider** : Cliquez sur **"OK"** sur toutes les fenêtres

4. **Redémarrer CMD/PowerShell** pour prendre en compte les nouvelles variables

#### Option B : Fichier de Configuration (Alternative)

1. **Créer le dossier de configuration** :
   ```powershell
   mkdir "C:\Program Files\MCP-Server-PRTG\configs"
   ```

2. **Créer le fichier** `C:\Program Files\MCP-Server-PRTG\configs\config.yaml` :
   ```yaml
   database:
     host: 192.168.1.100        # Adresse PostgreSQL
     port: 5432
     name: prtg_data_exporter
     user: prtg_reader
     password: VotreMotDePasseSecurise
     sslmode: disable           # ou 'require' pour SSL

   log:
     level: info                # debug, info, warn, error
   ```

### Étape 3 : Test de l'Installation

1. **Ouvrir PowerShell** :
   ```powershell
   cd "C:\Program Files\MCP-Server-PRTG"
   ```

2. **Tester la connexion** :
   ```powershell
   .\mcp-server-prtg_windows_amd64.exe
   ```

3. **Sortie attendue** :
   ```json
   {"time":"2025-10-24T16:40:00+02:00","level":"INFO","msg":"database connection established"}
   {"time":"2025-10-24T16:40:00+02:00","level":"INFO","msg":"MCP server initialized","server_name":"prtg-server","version":"1.0.0","tools_count":6}
   {"time":"2025-10-24T16:40:00+02:00","level":"INFO","msg":"starting PRTG MCP server"}
   ```

4. **Arrêter le serveur** : `Ctrl + C`

---

## 🔧 CONFIGURATION CLAUDE DESKTOP

### Étape 1 : Localiser le Fichier de Configuration

Le fichier de configuration Claude Desktop se trouve à :

```
%APPDATA%\Claude\claude_desktop_config.json
```

Chemin complet typique :
```
C:\Users\VotreNom\AppData\Roaming\Claude\claude_desktop_config.json
```

### Étape 2 : Ouvrir le Fichier de Configuration

1. **Ouvrir l'Explorateur Windows** : `Win + E`

2. **Aller dans la barre d'adresse** et taper :
   ```
   %APPDATA%\Claude
   ```

3. **Ouvrir le fichier** `claude_desktop_config.json` avec **Notepad** ou **VS Code**

   > **Note:** Si le fichier n'existe pas, créez-le !

### Étape 3 : Ajouter la Configuration MCP

#### Configuration Complète (Copier-Coller)

```json
{
  "mcpServers": {
    "prtg": {
      "command": "C:\\Program Files\\MCP-Server-PRTG\\mcp-server-prtg_windows_amd64.exe",
      "env": {
        "PRTG_DB_HOST": "192.168.1.100",
        "PRTG_DB_PORT": "5432",
        "PRTG_DB_NAME": "prtg_data_exporter",
        "PRTG_DB_USER": "prtg_reader",
        "PRTG_DB_PASSWORD": "VotreMotDePasseSecurise",
        "PRTG_DB_SSLMODE": "disable",
        "LOG_LEVEL": "info"
      }
    }
  }
}
```

#### Configuration avec Fichier YAML (Alternative)

Si vous utilisez un fichier `config.yaml` :

```json
{
  "mcpServers": {
    "prtg": {
      "command": "C:\\Program Files\\MCP-Server-PRTG\\mcp-server-prtg_windows_amd64.exe",
      "args": ["-config", "C:\\Program Files\\MCP-Server-PRTG\\configs\\config.yaml"],
      "env": {
        "LOG_LEVEL": "info"
      }
    }
  }
}
```

#### Configuration avec Plusieurs Serveurs MCP

Si vous avez déjà d'autres serveurs MCP configurés :

```json
{
  "mcpServers": {
    "prtg": {
      "command": "C:\\Program Files\\MCP-Server-PRTG\\mcp-server-prtg_windows_amd64.exe",
      "env": {
        "PRTG_DB_HOST": "192.168.1.100",
        "PRTG_DB_PORT": "5432",
        "PRTG_DB_NAME": "prtg_data_exporter",
        "PRTG_DB_USER": "prtg_reader",
        "PRTG_DB_PASSWORD": "VotreMotDePasseSecurise",
        "PRTG_DB_SSLMODE": "disable",
        "LOG_LEVEL": "info"
      }
    },
    "filesystem": {
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-filesystem", "C:\\Users"]
    },
    "github": {
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-github"],
      "env": {
        "GITHUB_PERSONAL_ACCESS_TOKEN": "votre_token"
      }
    }
  }
}
```

### Étape 4 : Vérifier la Configuration

#### Points Importants

1. **Backslash doublés** : Utilisez `\\` dans les chemins Windows
   - ✅ Correct : `"C:\\Program Files\\MCP-Server-PRTG\\mcp-server-prtg_windows_amd64.exe"`
   - ❌ Incorrect : `"C:\Program Files\MCP-Server-PRTG\mcp-server-prtg_windows_amd64.exe"`

2. **JSON valide** : Vérifiez les virgules et accolades
   - Pas de virgule après le dernier élément d'un objet
   - Guillemets doubles `"` uniquement (pas de guillemets simples `'`)

3. **Mot de passe** : Remplacez `VotreMotDePasseSecurise` par votre vrai mot de passe

4. **Adresse PostgreSQL** : Remplacez `192.168.1.100` par l'IP/hostname de votre serveur

#### Validation JSON

**Outil en ligne** : https://jsonlint.com/
1. Copiez votre configuration
2. Cliquez sur "Validate JSON"
3. Corrigez les erreurs éventuelles

### Étape 5 : Redémarrer Claude Desktop

1. **Fermer complètement Claude Desktop** :
   - Clic droit sur l'icône dans la barre des tâches
   - "Quitter" ou "Exit"

2. **Redémarrer Claude Desktop**

3. **Vérifier la connexion** : Dans Claude, vous devriez voir une icône 🔌 ou un indicateur de serveur MCP connecté

---

## ✅ VERIFICATION DE L'INSTALLATION

### Test 1 : Vérifier que Claude détecte le serveur

1. **Ouvrir Claude Desktop**

2. **Nouveau chat**, taper :
   ```
   Peux-tu me lister les serveurs MCP disponibles ?
   ```

3. **Réponse attendue** :
   ```
   Je détecte les serveurs MCP suivants :
   - prtg : Serveur PRTG avec 6 outils disponibles
   ```

### Test 2 : Tester une requête PRTG

Dans Claude Desktop, essayez :

```
Utilise le serveur PRTG pour me montrer les 5 premiers sensors disponibles
```

ou

```
Quels sont les sensors actuellement en alerte dans PRTG ?
```

ou

```
Donne-moi une vue d'ensemble du device "nom-de-votre-device"
```

### Test 3 : Vérifier les outils disponibles

```
Quels sont les outils PRTG disponibles ?
```

**Réponse attendue** :
```
Le serveur PRTG expose 6 outils :

1. prtg_get_sensors - Rechercher des sensors avec filtres
2. prtg_get_sensor_status - Status détaillé d'un sensor
3. prtg_get_alerts - Sensors en alerte
4. prtg_device_overview - Vue d'ensemble d'un device
5. prtg_top_sensors - Top sensors par métrique
6. prtg_query_sql - Requêtes SQL personnalisées
```

---

## 🔍 DÉPANNAGE

### Problème 1 : "Le serveur PRTG n'est pas disponible"

**Solutions** :

1. **Vérifier le fichier de configuration** :
   ```powershell
   notepad "%APPDATA%\Claude\claude_desktop_config.json"
   ```

2. **Vérifier le chemin de l'executable** :
   ```powershell
   Test-Path "C:\Program Files\MCP-Server-PRTG\mcp-server-prtg_windows_amd64.exe"
   ```
   Doit retourner `True`

3. **Tester l'executable manuellement** :
   ```powershell
   cd "C:\Program Files\MCP-Server-PRTG"
   $env:PRTG_DB_PASSWORD="VotreMotDePasse"
   .\mcp-server-prtg_windows_amd64.exe
   ```

4. **Vérifier les logs Claude** :
   ```
   %APPDATA%\Claude\logs\
   ```

### Problème 2 : "Failed to ping database"

**Causes possibles** :

1. **PostgreSQL n'est pas accessible** :
   ```powershell
   # Tester la connexion avec psql
   psql -h 192.168.1.100 -p 5432 -U prtg_reader -d prtg_data_exporter
   ```

2. **Firewall bloque la connexion** :
   - Vérifier que le port 5432 est ouvert
   - Windows Defender : Autoriser la connexion sortante

3. **Mauvais mot de passe** :
   - Vérifier `PRTG_DB_PASSWORD`
   - Tester avec psql

4. **SSL requis mais désactivé** :
   - Changer `PRTG_DB_SSLMODE` à `require`

### Problème 3 : "Database password is required"

**Solution** :

La variable `PRTG_DB_PASSWORD` n'est pas définie. Vérifiez :

1. **Variables d'environnement système** :
   ```powershell
   [System.Environment]::GetEnvironmentVariable("PRTG_DB_PASSWORD", "User")
   ```

2. **Configuration Claude Desktop** :
   Vérifiez que `PRTG_DB_PASSWORD` est bien dans la section `env`

3. **Fichier config.yaml** :
   Vérifiez le champ `password` dans `database:`

### Problème 4 : JSON invalide

**Erreur** : Claude Desktop ne démarre pas ou ne charge pas la configuration

**Solution** :

1. **Valider le JSON** :
   - https://jsonlint.com/
   - Vérifier les backslash : `\\` au lieu de `\`
   - Vérifier les virgules (pas de virgule finale)

2. **Configuration minimale de test** :
   ```json
   {
     "mcpServers": {
       "prtg": {
         "command": "C:\\Program Files\\MCP-Server-PRTG\\mcp-server-prtg_windows_amd64.exe",
         "env": {
           "PRTG_DB_PASSWORD": "test"
         }
       }
     }
   }
   ```

### Problème 5 : "Access Denied" lors de l'extraction

**Solution** :

1. **Extraire dans votre profil utilisateur** :
   ```
   C:\Users\VotreNom\MCP-Server-PRTG\
   ```

2. **Ou exécuter en tant qu'Administrateur** :
   - Clic droit sur l'executable
   - "Exécuter en tant qu'administrateur"

---

## 📋 CHECKLIST DE CONFIGURATION

Avant de contacter le support, vérifiez :

- [ ] Le fichier `mcp-server-prtg_windows_amd64.exe` existe
- [ ] La variable `PRTG_DB_PASSWORD` est définie
- [ ] L'adresse PostgreSQL est correcte (`PRTG_DB_HOST`)
- [ ] Le port PostgreSQL est correct (`PRTG_DB_PORT` = 5432)
- [ ] Le nom de la base est correct (`PRTG_DB_NAME`)
- [ ] Le fichier `claude_desktop_config.json` est valide (jsonlint.com)
- [ ] Les chemins utilisent des backslash doublés (`\\`)
- [ ] Claude Desktop a été redémarré après modification
- [ ] PostgreSQL est accessible (test avec psql)
- [ ] Le firewall autorise la connexion PostgreSQL
- [ ] Le mot de passe est correct (test manuel)

---

## 🔐 SÉCURITÉ - BONNES PRATIQUES

### 1. Créer un Utilisateur PostgreSQL Read-Only

**Sur votre serveur PostgreSQL** :

```sql
-- Créer l'utilisateur
CREATE USER prtg_reader WITH PASSWORD 'MotDePasseSecurise';

-- Donner accès à la base
GRANT CONNECT ON DATABASE prtg_data_exporter TO prtg_reader;
GRANT USAGE ON SCHEMA public TO prtg_reader;

-- Donner uniquement les droits de lecture
GRANT SELECT ON ALL TABLES IN SCHEMA public TO prtg_reader;

-- Pour les futures tables
ALTER DEFAULT PRIVILEGES IN SCHEMA public
GRANT SELECT ON TABLES TO prtg_reader;
```

### 2. Activer SSL/TLS

Si PostgreSQL supporte SSL :

```json
"env": {
  "PRTG_DB_SSLMODE": "require"
}
```

Niveaux de SSL disponibles :
- `disable` : Pas de SSL (par défaut)
- `require` : SSL obligatoire
- `verify-ca` : SSL + vérification certificat CA
- `verify-full` : SSL + vérification complète

### 3. Protéger le Fichier de Configuration

1. **Permissions** :
   - Clic droit sur `claude_desktop_config.json`
   - Propriétés → Sécurité
   - Limiter l'accès à votre compte uniquement

2. **Ne jamais commiter** :
   - Ne pas partager le fichier avec le mot de passe
   - Utiliser un gestionnaire de secrets en entreprise

### 4. Logging en Production

En production, utilisez `LOG_LEVEL=info` ou `warn` :

```json
"env": {
  "LOG_LEVEL": "info"
}
```

En développement uniquement : `LOG_LEVEL=debug`

---

## 🚀 UTILISATION AVEC CLAUDE

### Exemples de Questions

Une fois configuré, vous pouvez poser à Claude :

#### Recherche de Sensors

```
Montre-moi tous les sensors PING qui sont actuellement DOWN
```

```
Liste les sensors du device "web-server-prod"
```

```
Quels sont les sensors avec le tag "critical" ?
```

#### Alertes et Monitoring

```
Y a-t-il des alertes dans les dernières 24 heures ?
```

```
Quels sont les 10 sensors avec le plus de downtime ?
```

```
Donne-moi un résumé des sensors en warning
```

#### Vue d'Ensemble

```
Donne-moi une vue complète du device "database-server-01"
```

```
Combien de sensors sont actuellement en alerte ?
```

#### Requêtes Avancées

```
Utilise une requête SQL pour me montrer les sensors qui n'ont pas été vérifiés depuis plus de 2 heures
```

```
Combien y a-t-il de sensors de type "http" dans PRTG ?
```

---

## 📞 SUPPORT ET RESSOURCES

### Documentation

- **README.md** : Guide général du projet
- **SECURITY.md** : Best practices de sécurité
- **PROJECT_SUMMARY.md** : Détails techniques

### Logs

En cas de problème, joindre les logs :

**Logs serveur MCP** : Dans la sortie PowerShell quand vous exécutez manuellement

**Logs Claude Desktop** :
```
%APPDATA%\Claude\logs\
```

### Commandes Utiles

**Vérifier les variables d'environnement** :
```powershell
Get-ChildItem Env: | Where-Object {$_.Name -like "PRTG_*"}
```

**Tester la connexion PostgreSQL** :
```powershell
psql -h 192.168.1.100 -p 5432 -U prtg_reader -d prtg_data_exporter
```

**Afficher la configuration Claude** :
```powershell
Get-Content "$env:APPDATA\Claude\claude_desktop_config.json"
```

---

## 📝 NOTES DE VERSION

**Version 1.0.0** - 24 octobre 2025

- ✅ 6 outils MCP disponibles
- ✅ Support Windows 10/11 64-bit
- ✅ Configuration flexible (env vars + YAML)
- ✅ Sécurité SQL renforcée (0 injection)
- ✅ Logging structuré avec niveaux configurables

---

**🎉 Vous êtes prêt à utiliser PRTG avec Claude Desktop ! 🎉**

Pour toute question, consultez le README.md ou contactez le support.
