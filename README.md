# provider-cloudflare

`provider-cloudflare` is a [Crossplane](https://crossplane.io/) Provider
that manages Cloudflare resources via their V4 API (`cloudflare-go`). It comes
with the following resources:

- A `Zone` resource type that manages Cloudflare Zones.


## Developing

Run against a Kubernetes cluster:

```console
make run
```

Install `latest` into Kubernetes cluster where Crossplane is installed:

```console
make install
```

Install local build into [Kind](https://kind.sigs.k8s.io/docs/user/quick-start/)
cluster where Crossplane is installed:

```console
make install-local
```

Build, push, and install:

```console
make all
```

Build image:

```console
make image
```

Push image:

```console
make push
```

Build binary:

```console
make build
```
