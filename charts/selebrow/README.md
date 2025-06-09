# selebrow

![Version: 0.1.0](https://img.shields.io/badge/Version-0.1.0-informational?style=flat-square) ![Type: application](https://img.shields.io/badge/Type-application-informational?style=flat-square)

A Helm chart for standalone Selebrow deployment in Kubernetes

**Homepage:** <https://selebrow.github.io>

## Source Code

* <https://github.com/selebrow/selebrow>

## Values

### Selebrow service settings

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| selebrow.browserUri | string | `""` | Browsers catalog URI, leave empty to use fallback (remote) browsers URI |
| selebrow.namespace | string | `""` | namespace to create browser Pods, leave empty to match Selebrow deployment namespace) |
| selebrow.quota.limit | int | `0` | Browsers quota limit, set to positive value to limit number of concurrently running browsers |

### Browser template values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| templateValues.annotations | object | `{}` | Pod annotations |
| templateValues.browser.env | object | `{}` | Additional environment variables for browser container |
| templateValues.imagePullPolicy | string | `""` | Image pull policy (leave empty to apply default policy, see https://kubernetes.io/docs/concepts/containers/images/#imagepullpolicy-defaulting) |
| templateValues.imagePullSecrets | list | `[]` | Image pull secrets for the Pod |
| templateValues.labels | object | `{"app.kubernetes.io/created-by":"selebrow"}` | Static Pod labels |
| templateValues.priorityClassName | string | `""` | Priority class to set on the Pod (leave empty for default priority) |
| templateValues.scheduler | string | `""` | Custom Kubernetes scheduler to use (leave empty for default scheduler) |
| templateValues.searches | list | `[]` | Search domains list |

### Other Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| affinity | object | `{}` | Selebrow Pod affinity |
| extraEnv | object | `{}` | Extra environment variables for Selebrow container |
| fullnameOverride | string | `""` | Overrides full application name used in service/deployment and other Kubernetes resources' names |
| image.pullPolicy | string | `""` | Image pull policy (leave empty to apply default policy, see https://kubernetes.io/docs/concepts/containers/images/#imagepullpolicy-defaulting) |
| image.repository | string | `"ghcr.io/selebrow/selebrow"` | Repository image path without the tag |
| image.tag | string | `"latest"` | Overrides the image tag, whose default is the chart appVersion. |
| imagePullSecrets | list | `[]` | Image pull secrets |
| ingress.annotations | object | `{"nginx.ingress.kubernetes.io/proxy-read-timeout":"300"}` | Ingress [annotations](https://github.com/kubernetes/ingress-nginx/blob/main/docs/user-guide/nginx-configuration/annotations.md) |
| ingress.className | string | `""` | Ingress [class](https://kubernetes.io/docs/concepts/services-networking/ingress/#ingress-class) |
| ingress.enabled | bool | `false` | Enable ingress |
| ingress.hosts | list | `[{"host":"selebrow.local","paths":[{"path":"/","pathType":"ImplementationSpecific"}]}]` | Ingress Host and request routing rules |
| ingress.tls | list | `[]` | TLS secrets by hosts |
| nameOverride | string | `""` | Overrides short application name |
| nodeSelector | object | `{}` | Node selector for Selebrow Pod |
| podAnnotations | object | `{}` | Additional Selebrow Pod annotations |
| podLabels | object | `{}` | Additional Selebrow Pod labels |
| podSecurityContext | object | `{}` | Selebrow Pod [Security Context](https://kubernetes.io/docs/tasks/configure-pod-container/security-context/) |
| rbac.create | bool | `true` | Set to `true` to create role and role bindings for above service account |
| resources | object | `{}` | Selebrow container [resource settings](https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/) |
| securityContext | object | `{"capabilities":{"drop":["ALL"]},"readOnlyRootFilesystem":true,"runAsNonRoot":true,"runAsUser":65532}` | Selebrow container [Security Context](https://kubernetes.io/docs/tasks/configure-pod-container/security-context/) |
| service.port | int | `4444` | Service port |
| service.type | string | `"ClusterIP"` | Service [type](https://kubernetes.io/docs/concepts/services-networking/service/#publishing-services-service-types) to create |
| serviceAccount.annotations | object | `{}` | Annotations to add to the service account |
| serviceAccount.automount | bool | `true` | Automatically mount a ServiceAccount's API credentials? |
| serviceAccount.create | bool | `true` | Specifies whether a service account should be created |
| serviceAccount.name | string | `""` | The name of the service account to use. If not set and `create` is `true`, a name is generated using the fullname template |
| tolerations | list | `[]` | Tolerations for Selebrow Pod |
| volumeMounts | list | `[]` | Additional volumeMounts for Selebrow Pod |
| volumes | list | `[]` | Additional volumes for Selebrow Pod |

