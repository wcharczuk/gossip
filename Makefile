push: push-gossip push-data-plane push-metric-sink

push-gossip:
	@podman build --platform linux/arm64 . -t sf-microk8s.hawk-bluegill.ts.net:32000/gossip:latest -f Dockerfile
	@podman push sf-microk8s.hawk-bluegill.ts.net:32000/gossip:latest

push-data-plane:
	@podman build --platform linux/arm64 . -t sf-microk8s.hawk-bluegill.ts.net:32000/data-plane:latest -f Dockerfile.data_plane
	@podman push sf-microk8s.hawk-bluegill.ts.net:32000/data-plane:latest

push-metric-sink:
	@podman build --platform linux/arm64 . -t sf-microk8s.hawk-bluegill.ts.net:32000/metric-sink:latest -f Dockerfile.metric_sink
	@podman push sf-microk8s.hawk-bluegill.ts.net:32000/metric-sink:latest

apply: apply-gossip apply-data-plane apply-metric-sink

apply-gossip:
	@go run _k8s/gossip/main.go --name=service | kubectl -n gossip apply -f -
	@go run _k8s/gossip/main.go --name=deployment | kubectl -n gossip apply -f -

apply-data-plane:
	@go run _k8s/data-plane/main.go --name=service | kubectl -n gossip apply -f -
	@go run _k8s/data-plane/main.go --name=deployment | kubectl -n gossip apply -f -

apply-metric-sink:
	@go run _k8s/metric-sink/main.go --name=service | kubectl -n gossip apply -f -
	@go run _k8s/metric-sink/main.go --name=deployment | kubectl -n gossip apply -f -

deploy: deploy-gossip deploy-data-plane deploy-metric-sink

deploy-gossip:
	@kubectl -n gossip rollout restart deployment/gossip
	@kubectl -n gossip rollout status deployment/gossip -w

deploy-data-plane:
	@kubectl -n gossip rollout restart deployment/data-plane
	@kubectl -n gossip rollout status deployment/data-plane -w

deploy-metric-sink:
	@kubectl -n gossip rollout restart deployment/metric-sink
	@kubectl -n gossip rollout status deployment/metric-sink -w