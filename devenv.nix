{ pkgs, inputs, ... }:

let
  pkgs-unstable = import inputs.nixpkgs-unstable { system = pkgs.stdenv.system; };
  registryPort = "5000";
  registryName = "kind-registry";
  registryContainerName = "local-registry";
  clusterName = "dev-local";
  kustomizePath = "./infra/kube-local/competition-api-dev-local";
  kustomizeNamespace = "competition-api-dev-local";
in
{
  packages = [
    pkgs.sops
    pkgs.fluxcd
    pkgs.kustomize
    pkgs.kubelogin
    pkgs.kubectl
    pkgs.gnupg
    pkgs.git
    pkgs.gnutar
    pkgs.parallel
    pkgs.kind
    pkgs.kubernetes-helm
    pkgs.skopeo
    pkgs.jq
    pkgs.azure-cli

    pkgs.nixd
    pkgs.watch

    pkgs-unstable.k9s
    pkgs-unstable.mockgen
    pkgs-unstable.goreleaser
    pkgs-unstable.golangci-lint
    pkgs-unstable.go-swag
  ];

  languages.go.enable = true;
  languages.nix.enable = true;

  scripts.setupbetteralign.exec = ''
    go install github.com/dkorunic/betteralign/cmd/betteralign@latest
  '';

  scripts.competitionapi_golangcilint_fmt_apply.exec = ''
    golangci-lint fmt ./competition-api/...
  '';

  scripts.competitionapi_betteralign_apply.exec = ''
    betteralign -test_files -apply ./competition-api/...
  '';

  scripts.lint_go.exec = ''
    exit_code=0
    for mod in $(go list -m -u -json | jq -sr '.[].Dir'); do
      pushd "$mod" > /dev/null

      pwd

      if [[ -n "$(golangci-lint fmt --diff ./...)" ]]; then
        echo "ERROR: golangci-lint fmt made changes. Please fix and commit."
        exit_code=1
      fi

      if [[ -n "$(betteralign -test_files ./...)" ]]; then
        echo "ERROR: betteralign made changes. Please fix and commit."
        exit_code=1
      fi

      golangci-lint run --allow-parallel-runners ./...
      if [[ $? -ne 0 ]]; then
        exit_code=1
      fi

      popd > /dev/null
    done

    exit $exit_code
  '';

  enterShell = ''
    go version
    setupbetteralign
  '';

  git-hooks.hooks = {
    prettier = {
      enable = true;
    };
    shfmt = {
      enable = true;
    };
    golines = {
      enable = true;
    };
    nixpkgs-fmt = {
      enable = true;
    };
  };

  processes = {
    # Local OCI registry
    registry.exec = ''
      set -eu

      if podman container exists kind-registry; then
        echo "△ Removing old kind-registry container"
        podman rm -f ${registryContainerName}
      fi

      if ! podman network exists kind; then 
        echo "▶ starting local registry network"
        podman network create kind
      fi
        
      echo "▶ starting local registry on :${registryPort}"
      podman run -d --name ${registryContainerName} \
        --network kind \
        --network-alias ${registryName} \
        -p 127.0.0.1:5000:${registryPort} \
        registry:2

      cleanup() {
        echo "▼ clean up registry on exit"
        podman rm -f ${registryContainerName}
        podman network rm -f kind
      }
      trap cleanup EXIT INT TERM

      # set registry to insecure
      mkdir -p ~/.config/containers/registries.conf.d
      touch ~/.config/containers/registries.conf.d/kind-dev.conf
      cat > ~/.config/containers/registries.conf.d/kind-dev.conf << EOF
      [[registry]]
      location = "kind-registry:5000"
      insecure = true
      EOF

      echo "Registry up..."
      podman logs -f ${registryContainerName}
    '';

    # kind cluster
    localdevcluster.exec = ''
      set -euo pipefail
      CONFIG=$(mktemp)

      mkdir -p ./.kind/volumes

      # dynamically generate kind.yaml so it works on Mac and Linux
      cat > "$CONFIG" <<EOF
      kind: Cluster
      apiVersion: kind.x-k8s.io/v1alpha4
      name: ${clusterName}

      networking:
        apiServerAddress: "127.0.0.1"
      containerdConfigPatches:
      - |
        [plugins."io.containerd.grpc.v1.cri".registry.mirrors."localhost:5000"]
          endpoint = ["http://kind-registry:5000"]
        [plugins."io.containerd.grpc.v1.cri".registry.mirrors."kind-registry:5000"]
          endpoint = ["http://kind-registry:5000"]

      nodes:
        - role: control-plane
          image: kindest/node:v1.33.1
          extraPortMappings:
            - containerPort: 30080   # competition-api
              hostPort: 30080
              protocol: TCP
            - containerPort: 30081   # postgres
              hostPort: 30081
              protocol: TCP
            - containerPort: 30082   # redis
              hostPort: 30082
              protocol: TCP
            - containerPort: 30083   # azure blob
              hostPort: 30083
              protocol: TCP

        - role: worker
          image: kindest/node:v1.33.1
          labels:
              aixcc.tech/api-workload: devservices
      EOF

      # wait until registry is present to start to avoid first-time startup race
      while ! $(podman container exists ${registryContainerName})
      do
        echo "Waiting on ${registryContainerName} startup"
        sleep 1
      done
     
      # create cluster only if it’s missing
      if ! kind get clusters | grep -q "${clusterName}"; then
        echo "▶ creating kind cluster ${clusterName}"
        if uname -a | grep "NixOS"; then
          systemd-run --scope --user kind create cluster --name ${clusterName} --config "$CONFIG"
        else
          kind create cluster --name ${clusterName} --config "$CONFIG"
        fi
      fi

      cleanup() {
        echo "▼ deleting kind cluster"
        kind delete cluster --name ${clusterName} || true
        rm -rf .kind/
      }
      trap cleanup EXIT INT TERM

      # ensure registry container can be resolved from the nodes
      podman network connect kind ${registryContainerName} || true

      # keep the shell alive until devenv stops the process
      echo "✔ cluster ready – waiting…"
      podman logs -f ${clusterName}-worker
    '';
  };

  tasks."local:alias-registry".exec = ''
    REGISTRY_DIR="/etc/containerd/certs.d/localhost:${registryPort}"
    for node in $(kind get nodes); do
      docker exec "$node" mkdir -p "$REGISTRY_DIR"
      cat <<EOF | docker exec -i "$node" cp /dev/stdin "$REGISTRY_DIR/hosts.toml"
    [host."http://localhost:5000"]
    EOF
    done
    REGISTRY_DIR_2="/etc/containerd/certs.d/${registryName}:${registryPort}"
    for node in $(kind get nodes); do
      docker exec "$node" mkdir -p "$REGISTRY_DIR_2"
      cat <<EOF2 | docker exec -i "$node" cp /dev/stdin "$REGISTRY_DIR_2/hosts.toml"
    [host."http://${registryName}:${registryPort}"]
    EOF2
    done
  '';

  tasks."local:build-push".exec = ''
    set -euo pipefail

    make podman-build IMAGE_REPOSITORY=localhost:${registryPort}
    make podman-push PODMAN_PUSH_ARGS="--tls-verify=false" IMAGE_REPOSITORY=localhost:${registryPort}
  '';

  # 5) Apply Kustomize overlay that references the fresh image
  tasks."local:apply".exec = ''
    devenv tasks run local:alias-registry
    devenv tasks run local:build-push
    if [ $(kubectl get namespaces ${kustomizeNamespace}) ]; then
      kubectl replace -k ${kustomizePath} --force    # rebuild everything in case something not tracked is changed like a configMap
    else
      kubectl apply -k ${kustomizePath}
    fi
  '';
}
