SuperPlane Production-Only Flow

Canvas Node Mapping

- ci-success: receives webhook from GitHub Actions with sha and image references.
- pre-deploy-checks: run branch/image checks directly in Canvas command nodes.
- approval-gate: manual approval in SuperPlane UI.
- prod-deploy: run kubectl set image and rollout status in Canvas command nodes.
- health-check: call GET /health and verify status plus version.
- rollback-gate: run kubectl rollout undo commands on failure.
- notify: send Telegram or Slack from Canvas webhook/HTTP nodes.

Webhook payload fields

- event
- sha
- branch
- actor
- repo
- images.backend
- images.frontend
- images.ml
- images.web3

SuperPlane environment inputs

- SHA
- BRANCH
- DOCKERHUB_NAMESPACE or DOCKERHUB_USERNAME
- K8S_NAMESPACE (default chitsetu-prod)
- HEALTH_URL
- TELEGRAM_BOT_TOKEN and TELEGRAM_CHAT_ID (optional)
- SLACK_WEBHOOK_URL (optional)

Notes

- This is production only (no staging path).
- Kubernetes manifests are under k8s/prod.
- Health check validates status=ok and version=SHA when SHA is provided.
- Set backend APP_VERSION to SHA during deploy so /health returns deployed SHA.
