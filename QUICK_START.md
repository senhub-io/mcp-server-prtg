# 🚀 Guide de Démarrage Rapide - 5 Minutes

**Pour Windows** | Version v1.0.0

---

## 📥 ÉTAPE 1 : Installation (2 minutes)

### 1.1 Décompresser le ZIP

Décompressez `mcp-server-prtg_windows_amd64.zip` dans :
```
C:\Program Files\MCP-Server-PRTG\
```

### 1.2 Configurer les Variables d'Environnement

**Méthode rapide** :

1. `Win + R` → tapez `sysdm.cpl` → `Entrée`
2. Onglet **"Avancé"** → **"Variables d'environnement"**
3. Créez ces variables **Utilisateur** :

| Variable | Valeur à remplacer |
|----------|-------------------|
| `PRTG_DB_HOST` | `192.168.1.100` ← IP de votre serveur PostgreSQL |
| `PRTG_DB_PASSWORD` | `VotreMotDePasse` ← Mot de passe PostgreSQL |

Les autres variables ont des valeurs par défaut (optionnel) :
- `PRTG_DB_PORT` = `5432`
- `PRTG_DB_NAME` = `prtg_data_exporter`
- `PRTG_DB_USER` = `prtg_reader`
- `PRTG_DB_SSLMODE` = `disable`
- `LOG_LEVEL` = `info`

4. **OK** → **OK** → **OK**

---

## 🔧 ÉTAPE 2 : Configuration Claude Desktop (2 minutes)

### 2.1 Ouvrir le Fichier de Configuration

1. `Win + R` → tapez : `%APPDATA%\Claude` → `Entrée`

2. Ouvrez (ou créez) le fichier : `claude_desktop_config.json`

### 2.2 Copier-Coller cette Configuration

**Remplacez le contenu complet par** :

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

### 2.3 Personnaliser les Valeurs

Remplacez ces valeurs :
- `192.168.1.100` → IP de votre serveur PostgreSQL
- `VotreMotDePasseSecurise` → Votre vrai mot de passe PostgreSQL

**⚠️ IMPORTANT** :
- Utilisez des **double backslash** (`\\`) dans les chemins Windows
- Vérifiez qu'il n'y a **pas de virgule** après le dernier `}`

### 2.4 Enregistrer et Fermer

Enregistrez le fichier (`Ctrl + S`).

---

## ✅ ÉTAPE 3 : Tester (1 minute)

### 3.1 Redémarrer Claude Desktop

1. **Quitter complètement Claude Desktop** :
   - Clic droit sur l'icône dans la barre des tâches
   - "Quitter"

2. **Redémarrer Claude Desktop**

### 3.2 Vérifier la Connexion

Dans Claude Desktop, tapez :

```
Peux-tu utiliser le serveur PRTG pour me montrer les 5 premiers sensors ?
```

ou

```
Quels serveurs MCP sont disponibles ?
```

**Réponse attendue** :
```
Je détecte le serveur MCP "prtg" avec 6 outils disponibles.
Voici les 5 premiers sensors PRTG...
```

---

## 🎉 C'EST FINI !

Vous pouvez maintenant interroger PRTG en langage naturel via Claude.

---

## 💬 EXEMPLES DE QUESTIONS

### Monitoring

```
Y a-t-il des alertes actuellement dans PRTG ?
```

```
Montre-moi tous les sensors DOWN
```

```
Quels sont les 10 sensors avec le plus de downtime ?
```

### Recherche

```
Liste les sensors du device "web-server-prod"
```

```
Trouve tous les sensors de type PING
```

```
Quels sensors ont le tag "critical" ?
```

### Analyse

```
Donne-moi une vue d'ensemble du device "database-01"
```

```
Résume l'état des sensors dans les dernières 24h
```

```
Combien y a-t-il de sensors en warning ?
```

---

## 🔧 DÉPANNAGE RAPIDE

### ❌ "Le serveur PRTG n'est pas disponible"

**Solution 1** : Vérifier le chemin de l'executable
```powershell
Test-Path "C:\Program Files\MCP-Server-PRTG\mcp-server-prtg_windows_amd64.exe"
```
Doit retourner `True`

**Solution 2** : Vérifier le JSON
- https://jsonlint.com/ → Coller votre configuration
- Vérifier les `\\` dans les chemins
- Vérifier les virgules

**Solution 3** : Redémarrer Claude Desktop complètement
- Fermer toutes les fenêtres
- Quitter depuis la barre des tâches
- Relancer

### ❌ "Failed to connect to database"

**Solution 1** : Vérifier le mot de passe
```json
"PRTG_DB_PASSWORD": "VotreVraiMotDePasse"
```

**Solution 2** : Tester la connexion PostgreSQL
```powershell
psql -h 192.168.1.100 -p 5432 -U prtg_reader -d prtg_data_exporter
```

**Solution 3** : Vérifier l'IP du serveur
```json
"PRTG_DB_HOST": "192.168.1.100"  ← Remplacez par votre IP
```

### ❌ "Database password is required"

La variable `PRTG_DB_PASSWORD` n'est pas définie.

**Solution** : Vérifier dans `claude_desktop_config.json` :
```json
"env": {
  "PRTG_DB_PASSWORD": "VotreMotDePasse"  ← Doit être présent
}
```

---

## 📚 DOCUMENTATION COMPLÈTE

Pour plus de détails :
- **INSTALLATION_WINDOWS.md** : Guide complet d'installation
- **README.md** : Documentation générale
- **SECURITY.md** : Best practices de sécurité

---

## 🆘 BESOIN D'AIDE ?

### Logs Claude Desktop
```
%APPDATA%\Claude\logs\
```

### Test Manuel du Serveur
```powershell
cd "C:\Program Files\MCP-Server-PRTG"
.\mcp-server-prtg_windows_amd64.exe
```
Sortie attendue : `{"level":"INFO","msg":"database connection established"}`

### Variables d'Environnement
```powershell
Get-ChildItem Env: | Where-Object {$_.Name -like "PRTG_*"}
```

---

**Dernière mise à jour** : 24 octobre 2025 | **Version** : v1.0.0

🎊 **Bon monitoring avec PRTG et Claude !** 🎊
