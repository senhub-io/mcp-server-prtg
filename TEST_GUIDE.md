# 🧪 Guide de Test - MCP Server PRTG v2.0

## Prérequis

1. **PostgreSQL** avec la base `prtg_data_exporter`
2. **Build** du serveur: `make build`
3. **Configuration** de test prête

## 🚀 Test 1: Démarrage du Serveur (Mode Console)

### Avec Configuration YAML

```bash
# Créer la config de test
cat > config.test.yaml <<EOF
config_version: 1

agent:
  key: "test-api-key-12345678-1234-1234-1234-123456789abc"
  generated: false

server:
  bind_address: "127.0.0.1"
  port: 8443
  enable_tls: false
  read_timeout: 10
  write_timeout: 10

database:
  host: "localhost"
  port: 5432
  name: "prtg_data_exporter"
  user: "prtg_reader"
  password: "VotreMotDePasse"
  sslmode: "disable"

logging:
  level: "debug"
  file: "./logs/mcp-server-prtg-test.log"
  max_size_mb: 10
  max_backups: 2
  max_age_days: 7
  compress: false
EOF

# Lancer le serveur
./build/mcp-server-prtg run --config config.test.yaml
```

### Avec Arguments CLI

```bash
./build/mcp-server-prtg run \
  --port 8443 \
  --bind 127.0.0.1 \
  --db-host localhost \
  --db-password "VotreMotDePasse" \
  --log-level debug \
  --verbose
```

### Sortie Attendue

```
{"level":"info","module":"agent","time":"2025-10-24T18:00:00+02:00","message":"Initializing MCP Server PRTG Agent"}
{"level":"info","module":"configuration","time":"2025-10-24T18:00:00+02:00","message":"Configuration loaded successfully"}
{"level":"info","module":"database","time":"2025-10-24T18:00:00+02:00","message":"database connection established"}
{"level":"info","module":"agent","time":"2025-10-24T18:00:00+02:00","message":"MCP tools registered","tools_count":6}
{"level":"info","module":"server","time":"2025-10-24T18:00:00+02:00","message":"Starting MCP Server with SSE transport (v2)"}
{"level":"info","module":"server","time":"2025-10-24T18:00:00+02:00","message":"Starting internal SSE server","address":"127.0.0.1:18443"}
{"level":"info","module":"server","time":"2025-10-24T18:00:00+02:00","message":"Starting HTTP proxy server"}
{"level":"info","module":"server","time":"2025-10-24T18:00:00+02:00","message":"
╔════════════════════════════════════════════════════════════════╗
║         MCP Server PRTG - SSE Transport Started (v2)           ║
╠════════════════════════════════════════════════════════════════╣
║  Protocol:     HTTP                                            ║
║  External:     127.0.0.1:8443                                  ║
║  Internal:     127.0.0.1:18443                                 ║
║  Base URL:     http://127.0.0.1:8443                           ║
║  MCP Tools:    6 tools registered                              ║
║  Endpoints:                                                     ║
║    - GET  /sse      (SSE stream - authenticated)               ║
║    - POST /message  (RPC messages - authenticated)             ║
║    - GET  /health   (Health check - public)                    ║
║    - GET  /status   (Server status - authenticated)            ║
╠════════════════════════════════════════════════════════════════╣
║  Authentication:                                                ║
║    - Header: X-API-Key                                          ║
║    - Query:  ?api_key=...                                       ║
║  API Key: test...9abc                                           ║
╚════════════════════════════════════════════════════════════════╝
"}
```

## 🔍 Test 2: Vérification des Endpoints

### Test Health Check (Sans Auth)

```bash
curl http://127.0.0.1:8443/health
```

**Réponse attendue:**
```json
{"status":"healthy","timestamp":"2025-10-24T18:00:00+02:00"}
```

### Test Status (Avec Auth)

```bash
# Sans auth (doit échouer)
curl http://127.0.0.1:8443/status

# Avec auth (doit réussir)
curl -H "X-API-Key: test-api-key-12345678-1234-1234-1234-123456789abc" \
     http://127.0.0.1:8443/status
```

**Réponse attendue (avec auth):**
```json
{
  "status":"running",
  "version":"v2.0.0-alpha",
  "transport":"sse",
  "tls_enabled":false,
  "base_url":"http://127.0.0.1:8443",
  "mcp_tools":6,
  "timestamp":"2025-10-24T18:00:00+02:00"
}
```

### Test SSE Endpoint

```bash
# Test connexion SSE (doit rester ouvert et recevoir des events)
curl -N -H "X-API-Key: test-api-key-12345678-1234-1234-1234-123456789abc" \
     http://127.0.0.1:8443/sse
```

**Réponse attendue:**
```
event: endpoint
data: http://127.0.0.1:18443/message?sessionId=<uuid>

(stream reste ouvert)
```

## 🧪 Test 3: Test avec Client MCP

### Créer un Client MCP Simple (Python)

```python
# test_mcp_client.py
import requests
import json
from sseclient import SSEClient  # pip install sseclient-py

API_KEY = "test-api-key-12345678-1234-1234-1234-123456789abc"
BASE_URL = "http://127.0.0.1:8443"

def test_sse_connection():
    headers = {"X-API-Key": API_KEY}

    # Connect to SSE endpoint
    response = requests.get(
        f"{BASE_URL}/sse",
        headers=headers,
        stream=True
    )

    print(f"Status: {response.status_code}")
    print("SSE Events:")

    client = SSEClient(response)
    for event in client.events():
        print(f"  Event: {event.event}")
        print(f"  Data: {event.data}")

        # Get message endpoint from first event
        if event.event == "endpoint":
            message_url = event.data
            print(f"\n✅ Message URL: {message_url}")
            return message_url

    return None

def test_mcp_rpc(message_url):
    headers = {
        "X-API-Key": API_KEY,
        "Content-Type": "application/json"
    }

    # Call prtg_get_sensors tool
    rpc_request = {
        "jsonrpc": "2.0",
        "id": 1,
        "method": "tools/call",
        "params": {
            "name": "prtg_get_sensors",
            "arguments": {
                "limit": 5
            }
        }
    }

    response = requests.post(
        message_url,
        headers=headers,
        json=rpc_request
    )

    print(f"\n📡 RPC Response ({response.status_code}):")
    print(json.dumps(response.json(), indent=2))

if __name__ == "__main__":
    print("🧪 Testing MCP Server PRTG v2.0...\n")

    # Test SSE connection
    message_url = test_sse_connection()

    if message_url:
        # Test RPC call
        test_mcp_rpc(message_url)
    else:
        print("❌ Failed to get message URL")
```

### Exécuter le Test

```bash
pip install requests sseclient-py
python test_mcp_client.py
```

## 🔧 Test 4: Installation comme Service

### Linux (systemd)

```bash
# Installation
sudo ./build/mcp-server-prtg install \
  --config /etc/mcp-server-prtg/config.yaml \
  --working-dir /opt/mcp-server-prtg

# Vérification du service
systemctl status mcp-server-prtg

# Logs
journalctl -u mcp-server-prtg -f

# Démarrage
sudo systemctl start mcp-server-prtg

# Test endpoints
curl http://localhost:8443/health
```

### macOS (launchd)

```bash
# Installation
sudo ./build/mcp-server-prtg install --config ./config.yaml

# Liste des services
launchctl list | grep mcp-server-prtg

# Logs
tail -f ./logs/mcp-server-prtg.log

# Démarrage
sudo launchctl start mcp-server-prtg
```

### Windows (Service)

```powershell
# Installation (PowerShell Admin)
.\build\mcp-server-prtg_windows_amd64.exe install --config C:\mcp-server-prtg\config.yaml

# Vérification
Get-Service mcp-server-prtg

# Démarrage
Start-Service mcp-server-prtg

# Logs
Get-Content C:\mcp-server-prtg\logs\mcp-server-prtg.log -Wait
```

## 📊 Test 5: Vérification des Logs

### Structure des Logs

```bash
# Logs structurés JSON
tail -f logs/mcp-server-prtg-test.log | jq .
```

**Exemple de log:**
```json
{
  "level": "info",
  "module": "database",
  "time": "2025-10-24T18:00:00+02:00",
  "message": "executing query",
  "query": "SELECT id, name FROM prtg_sensor LIMIT $1",
  "args": [5]
}
```

### Vérification du Masquage

Les logs ne doivent **jamais** afficher:
- ❌ Mots de passe en clair
- ❌ API keys complètes
- ❌ Tokens

Exemple de masquage correct:
```json
{
  "password": "te***rd",
  "api_key": "test...9abc",
  "db_connection": "postgres://user:pa***rd@host"
}
```

## 🐛 Dépannage

### Erreur: "Failed to connect to database"

```bash
# Vérifier PostgreSQL
psql -h localhost -p 5432 -U prtg_reader -d prtg_data_exporter

# Vérifier la config
cat config.test.yaml | grep -A 6 "database:"

# Logs détaillés
./build/mcp-server-prtg run --verbose --debug-modules database
```

### Erreur: "Address already in use"

```bash
# Trouver le processus
lsof -i :8443

# Changer le port
./build/mcp-server-prtg run --port 8444
```

### Erreur: "Unauthorized"

```bash
# Vérifier l'API key dans la config
cat config.test.yaml | grep "key:"

# Test avec la bonne clé
curl -H "X-API-Key: $(grep 'key:' config.test.yaml | awk '{print $2}' | tr -d '"')" \
     http://127.0.0.1:8443/status
```

## ✅ Checklist de Test

- [ ] Build réussi sans erreurs
- [ ] Serveur démarre en mode console
- [ ] Endpoint `/health` répond
- [ ] Endpoint `/status` requiert auth
- [ ] Endpoint `/sse` établit une connexion
- [ ] Logs structurés JSON générés
- [ ] Secrets masqués dans les logs
- [ ] Configuration hot-reload fonctionne
- [ ] Service système installe correctement
- [ ] Service système démarre/arrête
- [ ] Client MCP peut se connecter
- [ ] Appels RPC fonctionnent

## 📝 Rapport de Test

```
Environnement: macOS 14.6 / Linux Ubuntu 22.04 / Windows 11
Version: v2.0.0-alpha
Date: 2025-10-24
Testeur: Matthieu Noirbusson

Résultats:
✅ Build: OK
✅ Démarrage: OK
✅ Endpoints: OK
✅ Authentication: OK
✅ SSE: OK
✅ Logs: OK
✅ Service: OK
⚠️  Client MCP: À tester avec base PostgreSQL réelle

Notes:
- Architecture proxy fonctionne correctement
- Authentification par API key opérationnelle
- Masquage des secrets effectif
- Service système s'installe sans erreur

Recommandations:
- Ajouter tests unitaires
- Créer tests d'intégration end-to-end
- Documenter schéma JSON-RPC
```

---

**Dernière mise à jour:** 24 octobre 2025
**Auteur:** Matthieu Noirbusson <matthieu.noirbusson@sensorfactory.eu>
