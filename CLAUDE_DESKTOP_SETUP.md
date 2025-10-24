# 🖥️ Configuration Claude Desktop - MCP Server PRTG

## Vue d'ensemble

Claude Desktop peut se connecter à votre serveur MCP PRTG de **deux façons**:

1. **Mode Local (stdio)** ❌ Plus supporté dans v2.0
2. **Mode Réseau (SSE)** ✅ Nouveau dans v2.0

## Architecture SSE

```
Claude Desktop ←→ Client MCP Local ←→ HTTPS/SSE ←→ MCP Server PRTG ←→ PostgreSQL
   (Interface)      (Proxy/Wrapper)    (Auth Bearer)      (Serveur)        (Données)
```

---

## 🚀 Option 1: Serveur Local (Recommandé pour démarrer)

### Étape 1: Préparer le Serveur

```bash
# 1. Créer le répertoire de config
mkdir -p ~/mcp-servers/prtg
cd ~/mcp-servers/prtg

# 2. Copier le binaire
cp /path/to/build/mcp-server-prtg ./mcp-server-prtg
chmod +x ./mcp-server-prtg

# 3. Créer la configuration
cat > config.yaml <<EOF
config_version: 1

server:
  api_key: "$(uuidgen | tr '[:upper:]' '[:lower:]')"  # Génère un UUID
  bind_address: "127.0.0.1"
  port: 8443
  enable_tls: false  # Désactiver TLS en local
  read_timeout: 10
  write_timeout: 10

database:
  host: "localhost"
  port: 5432
  name: "prtg_data_exporter"
  user: "prtg_reader"
  password: "VOTRE_MOT_DE_PASSE"
  sslmode: "disable"

logging:
  level: "info"
  file: "./logs/mcp-server-prtg.log"
  max_size_mb: 10
  max_backups: 5
  max_age_days: 30
  compress: true
EOF

# 4. Noter l'API key générée
API_KEY=$(grep 'api_key:' config.yaml | awk '{print $2}' | tr -d '"')
echo "Votre API Key: $API_KEY"

# 5. Tester le serveur
./mcp-server-prtg run --config config.yaml
```

### Étape 2: Configuration Claude Desktop

**Emplacement du fichier config:**
- **macOS**: `~/Library/Application Support/Claude/claude_desktop_config.json`
- **Windows**: `%APPDATA%\Claude\claude_desktop_config.json`
- **Linux**: `~/.config/Claude/claude_desktop_config.json`

**Contenu (méthode stdio - NE FONCTIONNE PAS DIRECTEMENT):**

⚠️ **PROBLÈME**: Claude Desktop utilise **stdio** mais notre serveur utilise **SSE**!

**SOLUTION**: Créer un **wrapper script** qui lance le serveur en arrière-plan.

---

## 🔧 Option 2: Wrapper Script (Solution de contournement)

### macOS/Linux: `mcp-prtg-wrapper.sh`

```bash
#!/bin/bash
# Wrapper pour MCP Server PRTG - Lance le serveur SSE
# Fichier: ~/mcp-servers/prtg/mcp-prtg-wrapper.sh

CONFIG_DIR="$HOME/mcp-servers/prtg"
PID_FILE="$CONFIG_DIR/server.pid"
LOG_FILE="$CONFIG_DIR/wrapper.log"

# Fonction pour démarrer le serveur
start_server() {
    cd "$CONFIG_DIR"

    # Vérifier si déjà démarré
    if [ -f "$PID_FILE" ] && kill -0 $(cat "$PID_FILE") 2>/dev/null; then
        echo "Server already running (PID: $(cat $PID_FILE))" >&2
        return 0
    fi

    # Démarrer le serveur en arrière-plan
    ./mcp-server-prtg run --config config.yaml >> "$LOG_FILE" 2>&1 &
    echo $! > "$PID_FILE"

    # Attendre que le serveur soit prêt
    sleep 2

    # Vérifier que le serveur est up
    if ! kill -0 $(cat "$PID_FILE") 2>/dev/null; then
        echo "Failed to start server" >&2
        exit 1
    fi

    echo "Server started (PID: $(cat $PID_FILE))" >&2
}

# Fonction pour arrêter le serveur
stop_server() {
    if [ -f "$PID_FILE" ]; then
        PID=$(cat "$PID_FILE")
        if kill -0 $PID 2>/dev/null; then
            kill $PID
            rm "$PID_FILE"
            echo "Server stopped" >&2
        fi
    fi
}

# Trap pour arrêter le serveur à la sortie
trap stop_server EXIT

# Démarrer le serveur
start_server

# Mode interactif: lire depuis stdin et faire des appels HTTP
API_KEY=$(grep 'api_key:' "$CONFIG_DIR/config.yaml" | awk '{print $2}' | tr -d '"')
BASE_URL="http://127.0.0.1:8443"

# Boucle de lecture stdin → HTTP → stdout
while IFS= read -r line; do
    # Envoyer la requête JSON-RPC au serveur SSE
    # NOTE: Cette partie nécessite une implémentation plus complexe
    # pour convertir stdio ← → SSE

    echo "$line" | curl -s -X POST \
        -H "Authorization: Bearer $API_KEY" \
        -H "Content-Type: application/json" \
        -d @- \
        "$BASE_URL/message"
done
```

**PROBLÈME**: Cette approche est **complexe** car il faut gérer la conversion stdio ↔ SSE.

---

## 🎯 Option 3: Client MCP SSE (Recommandé)

**La VRAIE solution**: Utiliser un **client MCP compatible SSE** ou attendre que Claude Desktop supporte SSE.

### Vérifier le support SSE de Claude Desktop

```bash
# Vérifier la version de Claude Desktop
# Si version >= 1.x avec support SSE transport, alors:
```

**Configuration avec SSE transport:**

```json
{
  "mcpServers": {
    "prtg": {
      "transport": "sse",
      "url": "http://127.0.0.1:8443/sse",
      "headers": {
        "Authorization": "Bearer votre-api-key-ici"
      }
    }
  }
}
```

---

## 📋 Configuration Complète (Si SSE supporté)

**Fichier**: `~/Library/Application Support/Claude/claude_desktop_config.json`

```json
{
  "mcpServers": {
    "prtg": {
      "transport": "sse",
      "url": "http://127.0.0.1:8443/sse",
      "headers": {
        "Authorization": "Bearer a1b2c3d4-e5f6-4789-a012-3456789abcde"
      },
      "timeout": 30000,
      "reconnect": true
    }
  }
}
```

### Remplacer l'API Key

```bash
# Récupérer votre API key
cd ~/mcp-servers/prtg
API_KEY=$(grep 'api_key:' config.yaml | awk '{print $2}' | tr -d '"')
echo "Votre API Key: $API_KEY"

# Mettre à jour claude_desktop_config.json avec cette clé
```

---

## 🧪 Tester la Connexion

### 1. Vérifier que le serveur tourne

```bash
# Health check
curl http://127.0.0.1:8443/health

# Status (avec auth)
API_KEY="votre-api-key"
curl -H "Authorization: Bearer $API_KEY" http://127.0.0.1:8443/status
```

### 2. Tester SSE endpoint

```bash
# Connexion SSE
curl -N -H "Authorization: Bearer $API_KEY" http://127.0.0.1:8443/sse
```

**Réponse attendue:**
```
event: endpoint
data: http://127.0.0.1:18443/message?sessionId=<uuid>
```

### 3. Redémarrer Claude Desktop

```bash
# macOS
killall Claude
open -a Claude

# Vérifier les logs
tail -f ~/Library/Logs/Claude/mcp*.log
```

---

## 🔍 Dépannage

### Claude Desktop ne voit pas le serveur

**Vérifications:**
1. Le serveur est-il démarré?
   ```bash
   ps aux | grep mcp-server-prtg
   curl http://127.0.0.1:8443/health
   ```

2. La config Claude Desktop est-elle valide?
   ```bash
   cat ~/Library/Application\ Support/Claude/claude_desktop_config.json | jq .
   ```

3. Logs du serveur:
   ```bash
   tail -f ~/mcp-servers/prtg/logs/mcp-server-prtg.log
   ```

### Erreur "Unauthorized"

```bash
# Vérifier que l'API key correspond
grep 'api_key:' ~/mcp-servers/prtg/config.yaml
grep 'Bearer' ~/Library/Application\ Support/Claude/claude_desktop_config.json
```

### Le serveur ne démarre pas

```bash
# Vérifier PostgreSQL
psql -h localhost -p 5432 -U prtg_reader -d prtg_data_exporter

# Lancer en mode verbose
cd ~/mcp-servers/prtg
./mcp-server-prtg run --config config.yaml --verbose
```

---

## 🌐 Configuration Serveur Distant

### Avec TLS (Production)

**Serveur distant**: `mcp.votredomaine.com`

```json
{
  "mcpServers": {
    "prtg": {
      "transport": "sse",
      "url": "https://mcp.votredomaine.com:8443/sse",
      "headers": {
        "Authorization": "Bearer votre-api-key-production"
      },
      "tls": {
        "verify": true
      }
    }
  }
}
```

**Configuration serveur:**
```yaml
server:
  api_key: "production-key-uuid-ici"
  bind_address: "0.0.0.0"  # Écouter sur toutes les interfaces
  port: 8443
  enable_tls: true
  cert_file: "/etc/ssl/certs/mcp.votredomaine.com.crt"
  key_file: "/etc/ssl/private/mcp.votredomaine.com.key"
```

---

## 📚 Références

- **MCP Protocol**: https://github.com/anthropics/mcp
- **SSE Transport**: https://developer.mozilla.org/en-US/docs/Web/API/Server-sent_events
- **Bearer Token**: RFC 6750

---

## ⚠️ Note Importante

**Claude Desktop (version actuelle) supporte principalement stdio.**

Pour utiliser SSE, vous devez soit:
1. ✅ Attendre une mise à jour de Claude Desktop avec support SSE
2. ✅ Utiliser un client MCP custom qui supporte SSE
3. ⚠️ Créer un wrapper stdio → SSE (complexe)

**Recommandation**: Utilisez le serveur avec des **clients HTTP standards** en attendant le support SSE natif dans Claude Desktop.

---

**Auteur:** Matthieu Noirbusson <matthieu.noirbusson@sensorfactory.eu>
**Version:** v2.0.0-alpha
**Date:** 24 octobre 2025
