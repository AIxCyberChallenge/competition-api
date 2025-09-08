# Competition API

The Competition API is a web API designed to enable the AIXCC Finals Competition by:

- Packaging challenge repositories as tarballs and delivering them to competitors to begin challenges,
- Receive submissions from challengers,
- Test those submissions with a built-in job pipeline,
- And run ad-hoc tests for scoring and review using previously-generated artifacts.

To get started, make sure devenv is installed (see below) and then follow the instructions in the Local Cluster section

## Environment Setup

### [devenv](https://devenv.sh)

This repository requires `devenv >= 1.4.0`.

#### Installation

1. Follow the guide specfic to your platform. **If using apple silicon** do not use the pre-release version like they suggest.
   It may also be necessary to set this environment variable during the install: `NIX_FIRST_BUILD_UID=331`.
2. Before attempting to use ensure you restart your shell
3. Execute `devenv shell` to enter the dev environment

Optionally, users can install the package `direnv` and execute `direnv allow .` inside the competition-api root directory.
This will configure devenv shell to automatically execute as soon as you enter the directory. This is just a convenience, not required.

If, upon your first execution of `devenv shell` you encounter an error relating to the devenv user not being able to
make system changes related to cachix, follow the recommended output option `a)` and write your user as a trusted user
alongside root in `/etc/nix/nix.conf`. This option simply enables your user to make changes to the nix package manager's
cache configuration.

#### Updating

1. `nix-env --install --attr devenv -f https://github.com/NixOS/nixpkgs/tarball/nixpkgs-unstable`

### Local Cluster

Requirements:

- devenv and nix as above
- podman
- At least 10GB of available space for image files on your `/home` dir

Devenv has been configured with processes to bootstrap a kind cluster to run the Competition API from a kustomize deployment locally.
To start these processes, run `devenv up` from the root competition-api directory. This will take over the terminal
with the devenv processes UI, from which you can monitor the state of the processes. If you'd rather background them, add
`-d` to `devenv up` to start them in the background. Press `ctrl-C` to exit a windowed process manager, or invoke
`devenv processes down` if you ran them in the background. It takes about a minute to bootstrap the kind cluster. After you
see log output from the worker node, you may proceed to building and applying your kubernetes configuration below.

#### Build and Apply New Images

First, move your `competition-api/infra/kube-local/competition-api-dev-local/sops/config.example.yaml` to `config.yaml`
in the same directory. Modify values as-necessary. Specifically update the IP address under `azure/storage_account/containers/url` to an ip address routable to the CRS and the containers in the cluster (local IP, tailscale, etc). It is also valid to use a real azure storage account if that is easier.

To build and apply new container images to run the Competition API in your local kind cluster, run `devenv tasks run local:apply`.
This command builds the images, pushes them to a local repository, loads them onto the kind nodes, and then applies
the kustomize for the local version of the deployment (competition-api/infra/kube-local). It might take a few minutes
the first time it is run.
It will also detect if the same label is used and transparently perform a kubectl replace to update the image regardless
of tag.
The Competition API service can be contacted at `http://localhost:30080` and can be tested by performing a test job from tools/test_local_job.sh

Note that if you are monitoring the status of the kind cluster as it comes up, with a tool like k9s or something similar,
when the cluster first starts it will show bad statuses for a few pods. As everything initializes those will stabilize, typically in
about a minute.

#### Notable Differences When Running locally

Since the Postgres instance exists within the kind cluster, when it's torn down its database goes with it, so the database
doesn't persist between usages

## GitHub App Setup

[Create the App](https://github.com/settings/apps/new) here. After done setting it up install it on the relevant repositories.

It should be installed on the oss fuzz repository (if non public) and any of the challenge repositories.

### Webhook URL

The webhook URL needs to be publicly accessible. Use a tool like `ngrok` to make that happen. Example: `ngrok http 30080`

### Permissions

- Read and write to issues
- Read and write to pull requests
- Read only, code scanning alerts
- Read only, contents

### Events

- Codescanning alert
- Pull Request
- Release
