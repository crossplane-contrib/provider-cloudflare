module github.com/benagricola/provider-cloudflare

go 1.13

require (
	github.com/cloudflare/cloudflare-go v0.17.0
	github.com/crossplane/crossplane-runtime v0.13.0
	github.com/crossplane/crossplane-tools v0.0.0-20210320162312-1baca298c527
	github.com/google/go-cmp v0.5.6
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.10.0
	gopkg.in/alecthomas/kingpin.v2 v2.2.6
	k8s.io/api v0.20.2
	k8s.io/apimachinery v0.21.1
	k8s.io/client-go v0.20.2
	k8s.io/utils v0.0.0-20210527160623-6fdb442a123b
	sigs.k8s.io/controller-runtime v0.8.3
	sigs.k8s.io/controller-tools v0.5.0
)
