bouncer: Kubernetes Authentication Webhook for Platform9-managed Kubernetes Clusters
---

# Usage

* auth-ttl
* unauth-ttl

# Performance Tuning

* cache-size
* bcrypt cost

# Known Issues

* While talking HTTPS to Keystone is optional, server verification is mandatory
  when HTTPS is used. In addition, the only CAs found at
  [hard-coded paths](https://raw.githubusercontent.com/golang/go/release-branch.go1.6/src/crypto/x509/root_linux.go)
  in the go stdlib are loaded.
