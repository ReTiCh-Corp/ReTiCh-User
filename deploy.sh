#!/bin/bash
set -euo pipefail

# ============================================================
# Deploy ReTiCh User API to Azure Container Apps via ACR
# ============================================================
# Prérequis :
#   - Azure CLI installé et connecté (az login)
# ============================================================

# --- Configuration -------------------------------------------
RESOURCE_GROUP="rg-retich-v2"
LOCATION="francecentral"
ENVIRONMENT_NAME="retich-env"
APP_NAME="retich-user"
ACR_NAME="retichregistry"
IMAGE_NAME="user"
IMAGE_TAG="latest"
TARGET_PORT=8083

# --- Charger les variables depuis .env.prod -------------------
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ENV_FILE="$SCRIPT_DIR/.env.prod"

if [ -f "$ENV_FILE" ]; then
    set -a
    source "$ENV_FILE"
    set +a
else
    echo "❌ Fichier .env.prod introuvable à la racine du projet."
    echo "   Crée-le avec : DATABASE_URL=postgres://user:pass@host:5432/db?sslmode=require"
    exit 1
fi

: "${DATABASE_URL:?❌ DATABASE_URL non définie dans .env.prod}"

# --- Couleurs ------------------------------------------------
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log()  { echo -e "${GREEN}[✓]${NC} $1"; }
warn() { echo -e "${YELLOW}[!]${NC} $1"; }
err()  { echo -e "${RED}[✗]${NC} $1"; exit 1; }

# --- Vérifications -------------------------------------------
echo "==========================================="
echo "  ReTiCh User — Azure Deployment (ACR)"
echo "==========================================="
echo ""

command -v az >/dev/null 2>&1 || err "Azure CLI non installé. Installe-le : https://aka.ms/install-azure-cli"
az account show >/dev/null 2>&1 || err "Non connecté à Azure. Lance : az login"
ACCOUNT=$(az account show --query name -o tsv)
log "Connecté au compte Azure : $ACCOUNT"

# --- Étape 1 : Vérifier le Resource Group --------------------
echo ""
warn "Étape 1/5 — Vérification du Resource Group..."

if az group show --name "$RESOURCE_GROUP" >/dev/null 2>&1; then
    log "Resource Group '$RESOURCE_GROUP' existe."
else
    warn "Resource Group '$RESOURCE_GROUP' introuvable. Création..."
    az group create --name "$RESOURCE_GROUP" --location "$LOCATION" -o none
    log "Resource Group '$RESOURCE_GROUP' créé."
fi

# --- Étape 2 : Vérifier l'Azure Container Registry -----------
echo ""
warn "Étape 2/5 — Azure Container Registry..."

if az acr show --name "$ACR_NAME" --resource-group "$RESOURCE_GROUP" >/dev/null 2>&1; then
    log "ACR '$ACR_NAME' existe déjà."
else
    warn "Création de l'ACR '$ACR_NAME'..."
    az acr create \
        --name "$ACR_NAME" \
        --resource-group "$RESOURCE_GROUP" \
        --location "$LOCATION" \
        --sku Basic \
        --admin-enabled true \
        -o none
    log "ACR '$ACR_NAME' créé."
fi

ACR_LOGIN_SERVER=$(az acr show --name "$ACR_NAME" --resource-group "$RESOURCE_GROUP" --query loginServer -o tsv)
IMAGE="$ACR_LOGIN_SERVER/$IMAGE_NAME:$IMAGE_TAG"
log "Registry : $ACR_LOGIN_SERVER"

# --- Étape 3 : Build & push via ACR --------------------------
echo ""
warn "Étape 3/5 — Build et push de l'image via ACR..."

az acr build \
    --registry "$ACR_NAME" \
    --resource-group "$RESOURCE_GROUP" \
    --image "$IMAGE_NAME:$IMAGE_TAG" \
    --file Dockerfile \
    "$SCRIPT_DIR"
log "Image '$IMAGE' buildée et pushée sur ACR."

# --- Étape 4 : Créer le Container Apps Environment -----------
echo ""
warn "Étape 4/5 — Container Apps Environment..."

if az containerapp env show --name "$ENVIRONMENT_NAME" --resource-group "$RESOURCE_GROUP" >/dev/null 2>&1; then
    log "Environment '$ENVIRONMENT_NAME' existe déjà."
else
    warn "Création de l'environnement '$ENVIRONMENT_NAME'..."
    az containerapp env create \
        --name "$ENVIRONMENT_NAME" \
        --resource-group "$RESOURCE_GROUP" \
        --location "$LOCATION" \
        -o none
    log "Environment '$ENVIRONMENT_NAME' créé."
fi

# --- Étape 5 : Déployer la Container App ---------------------
echo ""
warn "Étape 5/5 — Déploiement de la Container App..."

# Récupérer les credentials ACR
ACR_USERNAME=$(az acr credential show --name "$ACR_NAME" --resource-group "$RESOURCE_GROUP" --query username -o tsv)
ACR_PASSWORD=$(az acr credential show --name "$ACR_NAME" --resource-group "$RESOURCE_GROUP" --query "passwords[0].value" -o tsv)

if az containerapp show --name "$APP_NAME" --resource-group "$RESOURCE_GROUP" >/dev/null 2>&1; then
    warn "Container App '$APP_NAME' existe. Mise à jour..."
    az containerapp update \
        --name "$APP_NAME" \
        --resource-group "$RESOURCE_GROUP" \
        --image "$IMAGE" \
        --set-env-vars \
            "DATABASE_URL=$DATABASE_URL" \
            "REDIS_URL=${REDIS_URL:-}" \
            "LOG_LEVEL=${LOG_LEVEL:-info}" \
            "PORT=$TARGET_PORT" \
        -o none
    log "Container App mise à jour."
else
    warn "Création de la Container App '$APP_NAME'..."
    az containerapp create \
        --name "$APP_NAME" \
        --resource-group "$RESOURCE_GROUP" \
        --environment "$ENVIRONMENT_NAME" \
        --image "$IMAGE" \
        --registry-server "$ACR_LOGIN_SERVER" \
        --registry-username "$ACR_USERNAME" \
        --registry-password "$ACR_PASSWORD" \
        --target-port "$TARGET_PORT" \
        --ingress external \
        --min-replicas 0 \
        --max-replicas 3 \
        --cpu 0.25 \
        --memory 0.5Gi \
        --env-vars \
            "DATABASE_URL=$DATABASE_URL" \
            "REDIS_URL=${REDIS_URL:-}" \
            "LOG_LEVEL=${LOG_LEVEL:-info}" \
            "PORT=$TARGET_PORT" \
        -o none
    log "Container App '$APP_NAME' créée."
fi

# --- Résultat -------------------------------------------------
echo ""
echo "==========================================="
FQDN=$(az containerapp show \
    --name "$APP_NAME" \
    --resource-group "$RESOURCE_GROUP" \
    --query properties.configuration.ingress.fqdn \
    -o tsv)

log "Déploiement terminé !"
echo ""
echo -e "  URL    : ${GREEN}https://$FQDN${NC}"
echo -e "  Health : ${GREEN}https://$FQDN/health${NC}"
echo -e "  Ready  : ${GREEN}https://$FQDN/ready${NC}"
echo ""
echo "==========================================="

# --- Test health endpoint ------------------------------------
echo ""
warn "Test du health endpoint..."
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "https://$FQDN/health" || true)

if [ "$HTTP_CODE" = "200" ]; then
    log "Health check OK (HTTP 200)"
else
    warn "Health check retourne HTTP $HTTP_CODE — l'app peut mettre quelques secondes à démarrer."
    warn "Réessaie : curl https://$FQDN/health"
fi
