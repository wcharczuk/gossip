all: push apply

push:
	@podman build --platform linux/arm64 . -t sf-microk8s.hawk-bluegill.ts.net:32000/gossip:latest
	@podman push sf-microk8s.hawk-bluegill.ts.net:32000/gossip:latest

apply:
	@kubectl apply -f _k8s/00_namespace.yaml
	@kubectl -n gossip apply -f _k8s/01_service.yaml
	@kubectl -n gossip apply -f _k8s/02_deployment.yaml

deploy:
	@kubectl -n gossip rollout restart deployment/gossip
	@kubectl -n gossip rollout status deployment/gossip -w