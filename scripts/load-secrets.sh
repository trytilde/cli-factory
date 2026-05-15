#!/bin/bash
# Generate .env.secrets from secrets.yaml.
#
# secrets.yaml structure:
#   top-level scalar keys -> .env.secrets
#
# Example:
#   openai_api_key: sk-...
# becomes:
#   OPENAI_API_KEY="sk-..."

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
SECRETS_FILE="${1:-$ROOT_DIR/secrets.yaml}"
ENV_SECRETS="${2:-$ROOT_DIR/.env.secrets}"

if [[ ! -f "$SECRETS_FILE" ]]; then
    echo "Error: secrets.yaml not found at $SECRETS_FILE"
    echo "Run 'make sops-decrypt' or copy secrets.example.yaml to secrets.yaml first."
    exit 1
fi

if ! command -v yq &> /dev/null; then
    echo "Error: yq is not installed"
    echo "  brew install yq  # macOS"
    echo "  or: https://github.com/mikefarah/yq#install"
    exit 1
fi

{
    echo "# Generated from secrets.yaml -- DO NOT COMMIT"
    echo "# Generated at: $(date)"
    echo ""
    yq -o=json 'with_entries(select(.value | type == "!!str"))' "$SECRETS_FILE" \
      | python3 -c 'import json,sys
for k,v in json.load(sys.stdin).items():
    print(f"{k.upper()}=" + json.dumps(v))'
} > "$ENV_SECRETS"

echo "Written: $ENV_SECRETS"
