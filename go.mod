module github.com/benagricola/provider-cloudflare

go 1.13

require (
	github.com/cloudflare/cloudflare-go v0.16.0
	github.com/crossplane/crossplane-runtime v0.13.0
	github.com/crossplane/crossplane-tools v0.0.0-20210320162312-1baca298c527
	github.com/google/go-cmp v0.5.5
	github.com/pkg/errors v0.9.1
	gopkg.in/alecthomas/kingpin.v2 v2.2.6
	k8s.io/api v0.20.2
	k8s.io/apimachinery v0.21.0
	k8s.io/client-go v0.20.2
	k8s.io/utils v0.0.0-20210305010621-2afb4311ab10
	sigs.k8s.io/controller-runtime v0.8.3
	sigs.k8s.io/controller-tools v0.3.0
)
