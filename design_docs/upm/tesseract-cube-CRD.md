# API Reference

## Packages
- [tesseract-cube.bsgchina.com/v1alpha1](#tesseract-cubebsgchinacomv1alpha1)


## tesseract-cube.bsgchina.com/v1alpha1

Package v1alpha1 contains API Schema definitions for the tesseract-cube v1alpha1 API group

### Resource Types
- [Unit](#unit)
- [UnitList](#unitlist)
- [UnitSet](#unitset)
- [UnitSetList](#unitsetlist)



#### Architecture







_Appears in:_
- [UnitSetSpec](#unitsetspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `units` _integer_ |  |  |  |
| `mode` _string_ | enum: single,clone,replication_async,replication_semi_sync |  |  |
| `role` _string_ |  |  |  |


#### ConfigSyncStatus







_Appears in:_
- [UnitStatus](#unitstatus)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `lastTransitionTime` _[Time](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#time-v1-meta)_ |  |  |  |


#### ContainerPort







_Appears in:_
- [UnitTemplate](#unittemplate)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `port` _integer_ |  |  |  |
| `name` _string_ |  |  |  |


#### ExternalSecretInfo







_Appears in:_
- [UnitSetSpec](#unitsetspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `organization` _string_ |  |  |  |
| `root_secret` _string_ |  |  |  |


#### ImageSyncStatus







_Appears in:_
- [UnitSetStatus](#unitsetstatus)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `lastTransitionTime` _[Time](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#time-v1-meta)_ |  |  |  |


#### ImageVersion



ImageVersion 镜像版本



_Appears in:_
- [UnitSetSpec](#unitsetspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `type` _string_ | 镜像类型<br />required: true<br />example: infini-gateway |  |  |
| `major` _integer_ | 主版本号<br />required: true<br />minimum: 0 |  |  |
| `minor` _integer_ | 小版本号<br />required: true<br />minimum: 0 |  |  |
| `patch` _integer_ | 小更新版本号<br />required: true<br />minimum: 0 |  |  |
| `build` _integer_ | 编译版本号<br />required: true<br />minimum: 0 |  |  |
| `arch` _string_ | 架构<br />required: true<br />enum: arm64,amd64 |  |  |


#### InitContainer







_Appears in:_
- [PodTemplate](#podtemplate)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ |  |  |  |
| `image` _string_ |  |  |  |
| `command` _string array_ |  |  |  |
| `security_context` _[SecurityContext](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#securitycontext-v1-core)_ |  |  |  |
| `service_ports` _[ServicePort](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#serviceport-v1-core) array_ |  |  |  |


#### MainContainer







_Appears in:_
- [PodTemplate](#podtemplate)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ |  |  |  |
| `image` _string_ |  |  |  |
| `command` _string array_ |  |  |  |
| `readiness_probe` _[Probe](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#probe-v1-core)_ |  |  |  |
| `liveness_probe` _[Probe](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#probe-v1-core)_ |  |  |  |
| `startup_probe` _[Probe](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#probe-v1-core)_ |  |  |  |
| `service_ports` _[ServicePort](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#serviceport-v1-core) array_ |  |  |  |




#### PvcCapacity







_Appears in:_
- [PvcInfo](#pvcinfo)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `storage` _[Quantity](#quantity)_ |  |  |  |


#### PvcInfo







_Appears in:_
- [UnitStatus](#unitstatus)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ |  |  |  |
| `volumeName` _string_ |  |  |  |
| `accessModes` _[PersistentVolumeAccessMode](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#persistentvolumeaccessmode-v1-core) array_ |  |  |  |
| `capacity` _[PvcCapacity](#pvccapacity)_ |  |  |  |
| `phase` _[PersistentVolumeClaimPhase](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#persistentvolumeclaimphase-v1-core)_ |  |  |  |


#### PvcSyncStatus







_Appears in:_
- [UnitSetStatus](#unitsetstatus)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `lastTransitionTime` _[Time](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#time-v1-meta)_ |  |  |  |




#### ResourceSyncStatus







_Appears in:_
- [UnitSetStatus](#unitsetstatus)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `lastTransitionTime` _[Time](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#time-v1-meta)_ |  |  |  |


#### ServiceInfo







_Appears in:_
- [UnitSpec](#unitspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `serviceType` _string_ |  |  |  |




#### Sidecar







_Appears in:_
- [PodTemplate](#podtemplate)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ |  |  |  |
| `image` _string_ |  |  |  |
| `command` _string array_ |  |  |  |
| `args` _string array_ |  |  |  |
| `resource` _[ResourceRequirements](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#resourcerequirements-v1-core)_ |  |  |  |
| `security_context` _[SecurityContext](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#securitycontext-v1-core)_ |  |  |  |
| `service_ports` _[ServicePort](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#serviceport-v1-core) array_ |  |  |  |
| `readiness_probe` _[Probe](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#probe-v1-core)_ |  |  |  |
| `liveness_probe` _[Probe](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#probe-v1-core)_ |  |  |  |
| `startup_probe` _[Probe](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#probe-v1-core)_ |  |  |  |


#### Unit



Unit is the Schema for the units API



_Appears in:_
- [UnitList](#unitlist)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `tesseract-cube.bsgchina.com/v1alpha1` | | |
| `kind` _string_ | `Unit` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[UnitSpec](#unitspec)_ |  |  |  |


#### UnitList



UnitList contains a list of Unit





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `tesseract-cube.bsgchina.com/v1alpha1` | | |
| `kind` _string_ | `UnitList` | | |
| `metadata` _[ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#listmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `items` _[Unit](#unit) array_ |  |  |  |


#### UnitMetadata







_Appears in:_
- [UnitTemplate](#unittemplate)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ |  |  |  |
| `labels` _object (keys:string, values:string)_ |  |  |  |
| `annotations` _object (keys:string, values:string)_ |  |  |  |


#### UnitPhase

_Underlying type:_ _string_

UnitPhase is a label for the condition of a pod at the current time.



_Appears in:_
- [UnitStatus](#unitstatus)

| Field | Description |
| `Pending` | UnitPending means the pod has been accepted by the system, but one or more of the containers<br />has not been started. This includes time before being bound to a node, as well as time spent<br />pulling images onto the host.<br /> |
| `Running` | UnitRunning means the pod has been bound to a node and all of the containers have been started.<br />At least one container is still running or is in the process of being restarted.<br /> |
| `Ready` | UnitReady means the pod Running and ready condition = true<br /> |
| `Succeeded` | UnitSucceeded means that all containers in the pod have voluntarily terminated<br />with a container exit code of 0, and the system is not going to restart any of these containers.<br /> |
| `Failed` | UnitFailed means that all containers in the pod have terminated, and at least one container has<br />terminated in a failure (exited with a non-zero exit code or was stopped by the system).<br /> |
| `Unknown` | UnitUnknown means that for some reason the state of the pod could not be obtained, typically due<br />to an error in communicating with the host of the pod.<br />Deprecated: It isn't being set since 2015 (74da3b14b0c0f658b3bb8d2def5094686d0e9095)<br /> |


#### UnitSet



UnitSet is the Schema for the unitsets API



_Appears in:_
- [UnitSetList](#unitsetlist)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `tesseract-cube.bsgchina.com/v1alpha1` | | |
| `kind` _string_ | `UnitSet` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[UnitSetSpec](#unitsetspec)_ |  |  |  |


#### UnitSetList



UnitSetList contains a list of UnitSet





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `tesseract-cube.bsgchina.com/v1alpha1` | | |
| `kind` _string_ | `UnitSetList` | | |
| `metadata` _[ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#listmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `items` _[UnitSet](#unitset) array_ |  |  |  |


#### UnitSetSpec



UnitSetSpec defines the desired state of UnitSet



_Appears in:_
- [UnitSet](#unitset)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `image` _[ImageVersion](#imageversion)_ |  |  |  |
| `architecture` _[Architecture](#architecture)_ |  |  |  |
| `sharedConfigName` _string_ | shared config configmap name<br />如果非空，先检查是否存在该cm，如果没有则报错<br />则使用该configmap作为shared config |  |  |
| `volumeClaimTemplates` _[PersistentVolumeClaim](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#persistentvolumeclaim-v1-core) array_ |  |  | Schemaless: {} <br /> |
| `template` _[UnitTemplate](#unittemplate)_ |  |  |  |
| `externalSecret` _[ExternalSecretInfo](#externalsecretinfo)_ |  |  |  |
| `rollingUpdate` _[RollingUpdateStatefulSetStrategy](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#rollingupdatestatefulsetstrategy-v1-apps)_ |  |  | Schemaless: {} <br /> |
| `strategy` _string_ | enum: Recreate,RollingUpdate |  |  |
| `serviceInfo` _[UnitsetServiceInfo](#unitsetserviceinfo)_ |  |  | Schemaless: {} <br /> |




#### UnitSpec



UnitSpec defines the desired state of Unit



_Appears in:_
- [Unit](#unit)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `mainContainerName` _string_ |  |  |  |
| `mainImageVersion` _string_ |  |  |  |
| `startup` _boolean_ |  |  |  |
| `unBindNode` _boolean_ |  |  |  |
| `volumes` _[Volume](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#volume-v1-core) array_ |  |  | Schemaless: {} <br /> |
| `volumeClaimTemplates` _[PersistentVolumeClaim](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#persistentvolumeclaim-v1-core) array_ |  |  | Schemaless: {} <br /> |
| `volumeMounts` _[VolumeMount](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#volumemount-v1-core) array_ |  |  | Schemaless: {} <br /> |
| `serviceInfo` _[ServiceInfo](#serviceinfo)_ |  |  |  |
| `template` _[PodTemplateSpec](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#podtemplatespec-v1-core)_ |  |  | Schemaless: {} <br /> |




#### UnitTemplate







_Appears in:_
- [UnitSetSpec](#unitsetspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `metadata` _[UnitMetadata](#unitmetadata)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `unbindNode` _boolean_ |  |  |  |
| `env` _[EnvVar](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#envvar-v1-core) array_ |  |  | Schemaless: {} <br /> |
| `resource` _[ResourceRequirements](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#resourcerequirements-v1-core)_ |  |  | Schemaless: {} <br /> |
| `volumes` _[Volume](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#volume-v1-core) array_ |  |  | Schemaless: {} <br /> |
| `volumeMounts` _[VolumeMount](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#volumemount-v1-core) array_ |  |  | Schemaless: {} <br /> |
| `serviceInfo` _[UnitsetServiceInfo](#unitsetserviceinfo)_ |  |  |  |
| `affinity` _[Affinity](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#affinity-v1-core)_ |  |  | Schemaless: {} <br /> |
| `topologySpreadConstraints` _[TopologySpreadConstraint](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.22/#topologyspreadconstraint-v1-core) array_ |  |  | Schemaless: {} <br /> |
| `ports` _[ContainerPort](#containerport) array_ |  |  |  |
| `shareProcessNamespace` _boolean_ | default: true |  |  |
| `serviceAccount` _string_ |  |  |  |


#### UnitsetServiceInfo







_Appears in:_
- [UnitSetSpec](#unitsetspec)
- [UnitTemplate](#unittemplate)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `serviceType` _string_ |  |  |  |
| `portNames` _string array_ |  |  |  |


