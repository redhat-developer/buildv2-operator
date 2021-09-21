# Configure Shipwright with Gatekeeper

[Gatekeeper](https://github.com/open-policy-agent/gatekeeper) is a customizable admission webhook for Kubernetes, which allows you to configure [policy](https://www.openpolicyagent.org/docs/latest/policy-language/) over what resources can be created in the cluster. Gatekeeper is very generic, but in particular we can use it to add policy to Shipwright `Build`s.

## Only allow builds of allow-listed sources

An organization may want to limit builds to trusted sources, such as a github organization or an internal git repository. This is an example of how something like that could be configured.

(If you are using `kind` to test, you can use [hack/install-gatekeeper.sh](https://github.com/shipwright-io/build/blob/main/hack/install-tekton.sh) to install gatekeeper)

### Configure Gatekeeper to watch for `Build` resources

Either Create or append to your gatekeeper-system config. This configmap specifies all resources that Gatekeeper will monitor.

```yaml
apiVersion: config.gatekeeper.sh/v1alpha1
kind: Config
metadata:
  name: config
  namespace: "gatekeeper-system"
spec:
  sync:
    syncOnly:
      - group: "shipwright.io"
        version: "v1alpha1"
        kind: "Build"
```

### Create the ShipwrightAllowlist Constraint Template

This constraint template will let us create our [`ShipwrightAllowlist`](#create-your-shipwrightallowlist-constraint); in the paremeters to the `ShipwrightAllowlist` we will specify the allowed sources, and this constraint template will apply the logic.

```yaml
# Reference: https://github.com/open-policy-agent/gatekeeper/blob/master/demo/agilebank/templates/k8sallowedrepos_template.yaml
apiVersion: templates.gatekeeper.sh/v1beta1
kind: ConstraintTemplate
metadata:
  name: shipwrightallowlist
spec:
  crd:
    spec:
      names:
        kind: ShipwrightAllowlist
  targets:
    - target: admission.k8s.gatekeeper.sh
      rego: |
        package shipwrightallowlist

        violation[{"msg": msg}] {
          input.review.object.kind == "Build"
          repo_url := input.review.object.spec.source.url
          repo = strings.replace_n({
            "https://": "",
            "http://": "",
            "git://": "",
            "ssh://": "",
          }, repo_url)

          # is the repo in the allowlist?
          allowlist := [
            good | source = input.parameters.allowedsources[_];
            good = startswith(repo, source)
          ]
          not any(allowlist)

          msg := sprintf("The Build repo has not been pre-approved: %v. Allowed sources are: %v", [repo, input.parameters.allowedsources])
        }
```

### Create your ShipwrightAllowlist constraint

```yaml
apiVersion: constraints.gatekeeper.sh/v1beta1
kind: ShipwrightAllowlist
metadata:
  name: shipwrightallowlist
spec:
  match:
    kinds:
      - apiGroups: ["shipwright.io"]
        kinds: ["Build"]
  parameters:
    # Remember to terminate the sources with a `/` at the end.
    # Don't include the protocol. I.e.
    # GOOD: "github.com/shipwright-io/"
    # BAD:  "https://github.com/shipwright-io/"
    allowedsources:
      - "github.com/shipwright-io/"
```


### Test it out

This `Build` will be created, as is under the `github.com/shipwright-io/` organization, so it is allowed.

```yaml
# sample-go-build.yaml
apiVersion: shipwright.io/v1alpha1
kind: Build
metadata:
  name: buildah-golang-build
spec:
  source:
    url: https://github.com/shipwright-io/sample-go
    contextDir: docker-build
  strategy:
    name: buildah
    kind: ClusterBuildStrategy
  dockerfile: Dockerfile
  output:
    image: image-registry.openshift-image-registry.svc:5000/build-examples/taxi-app
```

Create it 

```sh
$ kubectl apply -f sample-go-build.yaml
build.shipwright.io/buildah-golang-build created
```


However the build below will not be created, as it belongs to `https://github.com/docker-library/`

```yaml
# hello-world-build.yaml
apiVersion: shipwright.io/v1alpha1
kind: Build
metadata:
  name: kaniko-hello-world-build
  annotations:
    build.shipwright.io/build-run-deletion: "true"
spec:
  source:
    url: https://github.com/docker-library/hello-world
    contextDir: .
  strategy:
    name: kaniko
    kind: ClusterBuildStrategy
  dockerfile: Dockerfile.build
  output:
    image: image-registry.openshift-image-registry.svc:5000/build-examples/hello-world
```

Attempting to create this will yield an error.

```sh
$ kubectl apply -f hello-world-build.yaml
Error from server ([shipwrightallowlist] The Build repo has not been pre-approved: github.com/docker-library/hello-world. Allowed sources are: ["github.com/shipwright-io/"]): error when creating "hello-world-build.yaml": admission webhook "validation.gatekeeper.sh" denied the request: [shipwrightallowlist] The Build repo has not been pre-approved: github.com/docker-library/hello-world. Allowed sources are: ["github.com/shipwright-io/"]
```
