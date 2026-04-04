Production Kubernetes Manifests

Files

- namespace.yaml
- backend.yaml
- frontend.yaml
- ml.yaml
- web3.yaml
- ingress.yaml

Deploy order

1. kubectl apply -f k8s/prod/namespace.yaml
2. Create required secrets:
   - chitsetu-backend-env
   - chitsetu-frontend-env
   - chitsetu-ml-env
   - chitsetu-web3-env
3. kubectl apply -f k8s/prod/backend.yaml
4. kubectl apply -f k8s/prod/frontend.yaml
5. kubectl apply -f k8s/prod/ml.yaml
6. kubectl apply -f k8s/prod/web3.yaml
7. kubectl apply -f k8s/prod/ingress.yaml

Image updates from SuperPlane Canvas

- kubectl set image deployment/chitsetu-backend backend=DOCKERHUB_NAMESPACE/chitsetu-backend:SHA -n chitsetu-prod
- kubectl set image deployment/chitsetu-frontend frontend=DOCKERHUB_NAMESPACE/chitsetu-frontend:SHA -n chitsetu-prod
- kubectl set image deployment/chitsetu-ml ml=DOCKERHUB_NAMESPACE/chitsetu-ml:SHA -n chitsetu-prod
- kubectl set image deployment/chitsetu-web3 web3=DOCKERHUB_NAMESPACE/chitsetu-web3:SHA -n chitsetu-prod
- kubectl set env deployment/chitsetu-backend APP_VERSION=SHA -n chitsetu-prod

Health check target

- GET https://DOMAIN_NAME/health
  Expected keys: status, db, blockchain, version
