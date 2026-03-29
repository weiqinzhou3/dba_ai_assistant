# 动作 -> 执行入口映射表（面向 Assistant）

## 0. 使用说明

### 0.1 文档目的

本文件按“标准动作”组织，而不是按 controller/模块组织，目的是为 future DBA Assistant 提供统一动作字典的第一版 blueprint。

### 0.2 判定口径

- `代码事实`：仓库里能直接追到的 controller / service / downstream。
- `推荐执行方式`：面向未来 assistant 的设计建议，不代表当前系统已有。
- `待验证`：代码没有完整闭环，或闭环存在但不够显式。

### 0.3 统一状态理解

- `有 task/order` 指当前 UPM 是否已有对应运行时/工单对象。
- `审批痕迹` 只表示是否能在当前代码中看到 `OrderGroup.approve(...)` 这类显式状态痕迹。
- `风险等级建议` 面向 future assistant，而不是当前 UPM 内置等级。

---

## 1. `mysql.database.create`

- 标准动作名：`mysql.database.create`
- 业务语义说明：在指定 MySQL 服务组上创建数据库 schema。
- 在 upm-api-server 中的入口位置：
  - Controller: `MysqlDatabaseController.save(...)`
  - Path: `POST /serv_groups/{serv_group_id}/mysql/databases`
  - 文件：`upm-service/upm-service-mysql/upm-service-mysql-ms/src/main/java/io/syntropycloud/upm/service/mysql/ms/controller/MysqlDatabaseController.java`
- 关键调用链：
  - `MysqlDatabaseController.save(...)`
  - `MysqlDatabaseServiceImpl.save(servGroupId, MysqlDatabaseForm)`
  - `MysqlDatabaseCheck.checkSave(...)`
  - `FeatureService.buildJdbcTemplate(servGroupDO)`
  - `JdbcTemplate.execute("CREATE DATABASE IF NOT EXISTS ...")`
  - 文件：`upm-service/upm-service-mysql/upm-service-mysql-ms/src/main/java/io/syntropycloud/upm/service/mysql/ms/service/impl/MysqlDatabaseServiceImpl.java`
- 依赖的核心对象：
  - `ServGroupDO`
  - `MysqlDatabaseForm`
- 是否异步：否
- 是否有 task/order：否 / 否
- 是否有审批痕迹：否
- 是否有操作日志：是，`@OperateLog(action = ADD)`
- 最终执行入口猜测：`DB-native`
- 对未来 Assistant 的推荐执行方式：`DB-native adapter`
- 风险等级建议：`R1 低风险写`
- 是否必须审批：建议否；生产高敏库可按策略升级
- 是否必须产出证据包：建议否，记录 SQL 摘要即可
- 备注 / 待验证点：
  - 当前代码没有显式 SQL dry-run/precheck 阶段，future assistant 可补

---

## 2. `mysql.user.create`

- 标准动作名：`mysql.user.create`
- 业务语义说明：创建 MySQL 账号，并带初始资源限制和授权集。
- 在 upm-api-server 中的入口位置：
  - Controller: `MysqlUserController.save(...)`
  - Path: `POST /serv_groups/{serv_group_id}/mysql/users`
  - 文件：`upm-service/upm-service-mysql/upm-service-mysql-ms/src/main/java/io/syntropycloud/upm/service/mysql/ms/controller/MysqlUserController.java`
- 关键调用链：
  - `MysqlUserController.save(...)`
  - `MysqlUserServiceImpl.save(servGroupId, MysqlUserForm)`
  - `MysqlUserCheck.checkSave(...)`
  - `FeatureService.buildJdbcTemplate(servGroupDO)`
  - `CREATE USER ...`
  - 遍历 `MysqlUserGrantForm` 执行 `GRANT ...`
  - 异常补偿：`DROP USER IF EXISTS ...`
  - 文件：`upm-service/upm-service-mysql/upm-service-mysql-ms/src/main/java/io/syntropycloud/upm/service/mysql/ms/service/impl/MysqlUserServiceImpl.java`
- 依赖的核心对象：
  - `ServGroupDO`
  - `MysqlUserForm`
  - `MysqlUserGrantForm`
- 是否异步：否
- 是否有 task/order：否 / 否
- 是否有审批痕迹：否
- 是否有操作日志：是，`@OperateLog(action = ADD)`
- 最终执行入口猜测：`DB-native`
- 对未来 Assistant 的推荐执行方式：`DB-native adapter`
- 风险等级建议：`R2 中风险`
- 是否必须审批：建议按账号类型决定；普通业务账号可否，管理账号应是
- 是否必须产出证据包：建议是，至少保留授权摘要
- 备注 / 待验证点：
  - 当前请求把“建用户”和“授初始权限”放在一起；future assistant 建议拆成两个动作

---

## 3. `mysql.user.grant`

- 标准动作名：`mysql.user.grant`
- 业务语义说明：为已有 MySQL 用户追加或调整权限。
- 在 upm-api-server 中的入口位置：
  - 没有独立 controller
  - 当前能力合并在 `MysqlUserController.update(...)`
  - Path: `PUT /serv_groups/{serv_group_id}/mysql/users/{username}`
- 关键调用链：
  - `MysqlUserController.update(...)`
  - `MysqlUserServiceImpl.update(servGroupId, username, MysqlUserBaseForm)`
  - `MysqlUserCheck.checkUpdate(...)`
  - `ALTER USER ...`
  - `buildAddMysqlUserGrantForm(...)` -> `GRANT ...`
  - `buildRemoveMysqlUserGrantDTO(...)` -> `REVOKE ...`
- 依赖的核心对象：
  - `ServGroupDO`
  - `MysqlUserBaseForm`
  - `MysqlUserGrantDTO/Form`
- 是否异步：否
- 是否有 task/order：否 / 否
- 是否有审批痕迹：否
- 是否有操作日志：是，复用 `MysqlUserController.update(...)` 的操作日志
- 最终执行入口猜测：`DB-native`
- 对未来 Assistant 的推荐执行方式：`DB-native adapter`
- 风险等级建议：`R2 中风险`，若涉及全库/管理权限建议升 `R3`
- 是否必须审批：建议对高权限 grant 为是
- 是否必须产出证据包：是
- 备注 / 待验证点：
  - 当前系统没有显式 `grant` 一等动作，future action dictionary 需要补齐

---

## 4. `mysql.password.change`

- 标准动作名：`mysql.password.change`
- 业务语义说明：修改指定 MySQL 用户密码。
- 在 upm-api-server 中的入口位置：
  - Controller: `MysqlUserController.resetPassword(...)`
  - Path: `PUT /serv_groups/{serv_group_id}/mysql/users/{username}/password/reset`
- 关键调用链：
  - `MysqlUserController.resetPassword(...)`
  - `MysqlUserServiceImpl.resetPassword(servGroupId, username, ResetPasswordForm)`
  - `MysqlUserCheck.checkResetPassword(...)`
  - `FeatureService.buildJdbcTemplate(servGroupDO)`
  - `ALTER USER ... IDENTIFIED BY ...`
- 依赖的核心对象：
  - `ServGroupDO`
  - `ResetPasswordForm`
- 是否异步：否
- 是否有 task/order：否 / 否
- 是否有审批痕迹：否
- 是否有操作日志：是，`@OperateLog(action = RESET_PASSWORD)`
- 最终执行入口猜测：`DB-native`
- 对未来 Assistant 的推荐执行方式：`DB-native adapter`
- 风险等级建议：`R2 中风险`
- 是否必须审批：建议对生产账号为是
- 是否必须产出证据包：是
- 备注 / 待验证点：
  - future assistant 需避免把明文密码写入普通审计日志

---

## 5. `mysql.backup.create`

- 标准动作名：`mysql.backup.create`
- 业务语义说明：创建一次 MySQL 备份任务，并把产物登记到备份文件对象。
- 在 upm-api-server 中的入口位置：
  - 服务组级：`ServGroupController.backup(...)`
  - Path: `PUT /serv_groups/{serv_group_id}/backup`
  - 单 unit 级：`UnitController.backup(...)`
  - Path: `PUT /units/{unit_id}/backup`
- 关键调用链：
  - 服务组级：
    - `ServGroupController.backup(...)`
    - `ServGroupServiceImpl.backup(...)`
    - `buildBackupTask(servGroupDO, ..., MysqlBackupForm)`
    - `saveTaskAndExecutor(taskDO)`
  - 运行级：
    - `ServGroupServiceImpl.executeSubtask(...)`
    - `SubtaskExecuteService.backup(...)`
    - `KubeResourceService.createBackupGrpcCall(...)`
    - `backupFileDAO.save(MysqlBackupFileDO)`
    - `KubeResourceCoreService.pollingSubTaskGrpcCallStatus(...)`
    - 回填 `MysqlBackupFileDO.status/size/msg`
- 依赖的核心对象：
  - `ServGroupDO`
  - `UnitDO`
  - `MysqlBackupForm` / `MysqlUnitBackupForm`
  - `MysqlBackupFileDO`
  - `StorageS3DO`
- 是否异步：是
- 是否有 task/order：有 task / 无 order
- 是否有审批痕迹：无显式审批
- 是否有操作日志：是
- 最终执行入口猜测：`GrpcCall + S3`
- 对未来 Assistant 的推荐执行方式：`GrpcCall adapter` 或 `MCP tool -> 受控 backup tool`
- 风险等级建议：`R1 低风险写`，若在主库执行或带锁备份建议升 `R2`
- 是否必须审批：建议否；生产主库/全量备份窗口外可要求审批
- 是否必须产出证据包：是
- 备注 / 待验证点：
  - 当前代码支持定时策略：`MysqlBackupStrategyTask`
  - future assistant 应把“即时备份”和“策略备份”都纳入统一动作族

---

## 6. `mysql.restore.create`

- 标准动作名：`mysql.restore.create`
- 业务语义说明：从备份文件恢复到指定 unit。
- 在 upm-api-server 中的入口位置：
  - 没有显式 `restore` controller
  - 当前借道：`UnitController.recover(...)`
  - Path: `PUT /units/{unit_id}/recover`
- 关键调用链：
  - `UnitController.recover(...)`
  - `UnitServiceImpl.recover(id, MysqlRecoverForm)`
  - `execute(id, RECOVER, paramJson)`
  - `buildRecoverTask(...)`
  - 当 `MysqlRecoverForm.backupFileId` 存在时：
    - `buildRestoreSubtaskDOs(...)`
    - 子任务序列：`ISOLATE? -> STOP -> RESTORE -> REMOVE_POD -> START -> GTID_PURGED -> RECOVER?`
  - `SubtaskExecuteService.restore(...)`
  - `KubeResourceService.createRestoreGrpcCall(...)`
  - `KubeResourceCoreService.pollingSubTaskGrpcCallStatus(...)`
- 依赖的核心对象：
  - `UnitDO`
  - `MysqlRecoverForm`
  - `MysqlBackupFileDO`
- 是否异步：是
- 是否有 task/order：有 task / 无 order
- 是否有审批痕迹：无显式审批
- 是否有操作日志：是，记录在 `UnitController.recover(...)`
- 最终执行入口猜测：`GrpcCall + Pod lifecycle + replication CRD`
- 对未来 Assistant 的推荐执行方式：`GrpcCall adapter + K8s adapter`
- 风险等级建议：`R3 高风险`
- 是否必须审批：是
- 是否必须产出证据包：是
- 备注 / 待验证点：
  - 当前系统里 `restore` 不是一等动作，而是 `recover` 的一种分支；future assistant 应拆成显式动作

---

## 7. `mysql.service.create`

- 标准动作名：`mysql.service.create`
- 业务语义说明：创建新的 MySQL 服务组，可带 ProxySQL。
- 在 upm-api-server 中的入口位置：
  - `OrderGroupController.save(...)` -> `POST /order_groups`
  - `OrderGroupController.approve(...)` -> `PUT /order_groups/{order_group_id}/approval`
  - `OrderGroupController.execute(...)` -> `PUT /order_groups/{order_group_id}/execute`
- 关键调用链：
  - `OrderGroupServiceImpl.save(...)`
  - `BaseService.buildOrderGroupCoreForm(OrderGroupForm)`
  - `OrderGroupCoreService.save(...)`
  - `OrderGroupCoreService.approve(...)`
  - `OrderGroupCoreService.execute(...)`
  - `ServGroupCoreService.create(orderGroupDO, isDelete)`
  - `TaskSchedulerCoreService.saveTaskAndExecutor(taskDO)`
  - `ServGroupServiceImpl.executeSubtask(...)`
  - 下发到 `KubeResourceCoreService` / `SubtaskExecuteService`
- 依赖的核心对象：
  - `OrderGroupDO`
  - `OrderDO`
  - `ServGroupDO`
  - `ServDO`
  - `UnitDO`
  - `StorageClassDO`
  - `Software`
- 是否异步：是
- 是否有 task/order：有 task / 有 order
- 是否有审批痕迹：有
- 是否有操作日志：是
- 最终执行入口猜测：`K8s/CRD + replication/proxysql sync`
- 对未来 Assistant 的推荐执行方式：`CRD adapter + K8s adapter`
- 风险等级建议：`R3 高风险`
- 是否必须审批：是
- 是否必须产出证据包：是
- 备注 / 待验证点：
  - 当前创建动作没有独立 REST 入口，而是通过工单型动作完成；future assistant 很适合保留这套模式

---

## 8. `mysql.service.scale`

- 标准动作名：`mysql.service.scale`
- 业务语义说明：变更 MySQL 服务的计算、存储或副本数。
- 在 upm-api-server 中的入口位置：
  - `PUT /serv_groups/{serv_group_id}/computer_scaling`
  - `PUT /serv_groups/{serv_group_id}/storage_expansion`
  - `PUT /serv_groups/{serv_group_id}/unit_expansion`
  - Controller: `ServGroupController`
- 关键调用链：
  - `ServGroupController.*Scaling(...)`
  - MySQL `ServGroupServiceImpl` 进入 `ServGroupCoreService`
  - 固化 `OrderGroupDO / OrderDO`
  - `OrderGroupCoreService.execute(...)`
  - `ServGroupCoreService.computerScaling/storageExpansion/unitExpansion(...)`
  - `TaskSchedulerCoreService.saveTaskAndExecutor(...)`
  - `KubeResourceCoreService.updateUnitSetResource/updateUnitSetStorage/updateUnitSetUnits(...)`
  - MySQL 副本变化再由 `SubtaskExecuteService.updateMysqlReplication(...)` 同步
- 依赖的核心对象：
  - `ServGroupDO`
  - `OrderGroupDO`
  - `OrderDO`
  - `ScaleDO`
- 是否异步：是
- 是否有 task/order：有 / 有
- 是否有审批痕迹：有
- 是否有操作日志：是
- 最终执行入口猜测：`K8s/CRD + replication CRD`
- 对未来 Assistant 的推荐执行方式：`CRD adapter + K8s adapter`
- 风险等级建议：`R2 中风险`
- 是否必须审批：建议是，至少生产环境为是
- 是否必须产出证据包：是
- 备注 / 待验证点：
  - current UPM 把 scale 拆成三条接口；future assistant 建议统一成一个动作并增加 `dimension`

---

## 9. `mysql.service.start`

- 标准动作名：`mysql.service.start`
- 业务语义说明：启动整个 MySQL 服务组。
- 在 upm-api-server 中的入口位置：
  - `ServGroupController.start(...)`
  - Path: `PUT /serv_groups/{serv_group_id}/start`
- 关键调用链：
  - `ServGroupController.start(...)`
  - `ServGroupService.start(...)` / `ServGroupCoreService.start(...)`
  - `buildTask(servGroupDO, START, ...)`
  - `saveTaskAndExecutor(taskDO)`
  - `SubtaskExecuteCoreService.startUnitSet(...)`
  - `KubeResourceCoreService.startUnitSet(...)`
  - `k8sUnitCrdService.start(...)`
- 依赖的核心对象：
  - `ServGroupDO`
  - `ServDO`
  - `TaskDO`
- 是否异步：是
- 是否有 task/order：有 task / 无 order
- 是否有审批痕迹：无显式审批
- 是否有操作日志：是
- 最终执行入口猜测：`K8s Unit/UnitSet CRD`
- 对未来 Assistant 的推荐执行方式：`CRD adapter` 或 `K8s adapter`
- 风险等级建议：`R2 中风险`
- 是否必须审批：建议生产环境为是
- 是否必须产出证据包：建议是
- 备注 / 待验证点：
  - start/stop 更像运行态控制动作，不一定需要工单，但需要强审计

---

## 10. `mysql.service.stop`

- 标准动作名：`mysql.service.stop`
- 业务语义说明：停止整个 MySQL 服务组。
- 在 upm-api-server 中的入口位置：
  - `ServGroupController.stop(...)`
  - Path: `PUT /serv_groups/{serv_group_id}/stop`
- 关键调用链：
  - `ServGroupController.stop(...)`
  - `ServGroupCoreService.stop(...)`
  - `buildTask(servGroupDO, STOP, ...)`
  - `saveTaskAndExecutor(taskDO)`
  - `SubtaskExecuteCoreService.stopUnitSet(...)`
  - `KubeResourceCoreService.stopUnitSet(...)`
  - `k8sUnitCrdService.stop(...)`
- 依赖的核心对象：
  - `ServGroupDO`
  - `ServDO`
  - `TaskDO`
- 是否异步：是
- 是否有 task/order：有 task / 无 order
- 是否有审批痕迹：无显式审批
- 是否有操作日志：是
- 最终执行入口猜测：`K8s Unit/UnitSet CRD`
- 对未来 Assistant 的推荐执行方式：`CRD adapter` 或 `K8s adapter`
- 风险等级建议：`R3 高风险`
- 是否必须审批：是
- 是否必须产出证据包：是
- 备注 / 待验证点：
  - 强烈建议 future assistant 对 stop 类动作强制 human-in-the-loop

---

## 11. `mysql.unit.rebuild`

- 标准动作名：`mysql.unit.rebuild`
- 业务语义说明：重建指定 unit，必要时先 isolate。
- 在 upm-api-server 中的入口位置：
  - `UnitController.rebuild(...)`
  - Path: `PUT /units/{unit_id}/rebuild`
- 关键调用链：
  - `UnitController.rebuild(...)`
  - `UnitServiceImpl.rebuild(id, UnitRebuildForm)`
  - `UnitCoreService.rebuild(...)`
  - `UnitServiceImpl.buildRebuildTask(...)`
  - 当满足条件时先加 `ISOLATE` 子任务，再加 `REBUILD`
  - `saveTaskAndExecutor(taskDO)`
  - `SubtaskExecuteCoreService.rebuildUnit(...)`
  - 依赖 `KubeResourceCoreService.rebuildUnit(...)`
- 依赖的核心对象：
  - `UnitDO`
  - `UnitRebuildForm`
  - `ServDO`
  - `ServGroupDO`
- 是否异步：是
- 是否有 task/order：有 task / 无 order
- 是否有审批痕迹：无显式审批
- 是否有操作日志：是
- 最终执行入口猜测：`K8s Unit/Pod lifecycle + replication CRD`
- 对未来 Assistant 的推荐执行方式：`CRD adapter + K8s adapter`
- 风险等级建议：`R2 中风险`，若主实例/唯一实例则升 `R3`
- 是否必须审批：建议按拓扑角色决定
- 是否必须产出证据包：是
- 备注 / 待验证点：
  - 当前逻辑已体现“拓扑感知动作规划”，这是很值得保留的设计

---

## 12. `mysql.parameter.change`

- 标准动作名：`mysql.parameter.change`
- 业务语义说明：修改服务参数，若参数支持动态更新则直接热生效，否则写入配置等待重启。
- 在 upm-api-server 中的入口位置：
  - `ServController.updateParamCfg(...)`
  - Path: `PUT /servs/{serv_id}/cfgs`
- 关键调用链：
  - `ServController.updateParamCfg(...)`
  - `ServServiceImpl.updateParamCfg(...)`
  - `ServCheck.checkUpdateParamCfg(...)`
  - `ServCoreService.updateParamCfgCore(servDO, paramCfgCoreForm)`
  - `KubeResourceCoreService.updateParamCfg(...)`
  - 对每个 unit 更新 `ConfigMap`
  - 若 `KeySet.dynamic == true`：
    - `setParamVariable(...)`
    - 构造 `GrpcCall`
    - 轮询执行结果
- 依赖的核心对象：
  - `ServDO`
  - `ParamCfgForm`
  - `KeySet`
  - `ConfigMap`
- 是否异步：当前 API 侧表现为同步
- 是否有 task/order：否 / 否
- 是否有审批痕迹：否
- 是否有操作日志：是，`@OperateLog(action = MODIFY_CFG)`
- 最终执行入口猜测：`ConfigMap + 可选 GrpcCall`
- 对未来 Assistant 的推荐执行方式：`K8s adapter + GrpcCall adapter`
- 风险等级建议：`R2 中风险`
- 是否必须审批：建议按参数类别决定；影响复制/日志/持久化的参数应是
- 是否必须产出证据包：是
- 备注 / 待验证点：
  - future assistant 应把参数变更分为：
    - `dynamic`
    - `restart-required`
    - `unsafe`

---

## 13. `resource.cluster.register`

- 标准动作名：`resource.cluster.register`
- 业务语义说明：注册一个 K8s 集群进入控制面，并完成基础初始化。
- 在 upm-api-server 中的入口位置：
  - `ClusterController.save(...)`
  - Path: `POST /clusters`
- 关键调用链：
  - `ClusterController.save(...)`
  - `ClusterServiceImpl.save(ClusterForm)`
  - `ClusterCheck.checkSave(...)`
  - `clusterDAO.save(clusterDO)`
  - `k8sRedisUtils.save(clusterDO.getRelateName(), clusterConfig)`
  - 遍历已有 project：
    - `k8sProjectService.create(...)`
    - `k8sSecretService.create(...)`
- 依赖的核心对象：
  - `ClusterDO`
  - `ProjectDO`
  - `ProjectCertificatesDO`
- 是否异步：否
- 是否有 task/order：否 / 否
- 是否有审批痕迹：否
- 是否有操作日志：是，`@OperateLog(action = REGISTER)`
- 最终执行入口猜测：`内部持久化 + K8s API`
- 对未来 Assistant 的推荐执行方式：`K8s adapter`
- 风险等级建议：`R3 高风险`
- 是否必须审批：是
- 是否必须产出证据包：是
- 备注 / 待验证点：
  - cluster register 影响全局资源面，应由平台管理员动作触发，而不是普通 DBA 自助执行

---

## 14. `resource.storageclass.register`

- 标准动作名：`resource.storageclass.register`
- 业务语义说明：把 K8s 集群里的 storage class 纳入平台受控目录。
- 在 upm-api-server 中的入口位置：
  - `StorageClassController.save(...)`
  - Path: `POST /storageclasses`
- 关键调用链：
  - `StorageClassController.save(...)`
  - `StorageClassServiceImpl.save(StorageClassForm)`
  - `StorageClassCheck.checkSave(...)`
  - `storageClassDAO.save(storageClassDO)`
  - `k8sStorageClassService.register(clusterRelateName, relateName)`
- 依赖的核心对象：
  - `StorageClassDO`
  - `ClusterDO`
- 是否异步：否
- 是否有 task/order：否 / 否
- 是否有审批痕迹：否
- 是否有操作日志：是，`@OperateLog(action = REGISTER)`
- 最终执行入口猜测：`内部持久化 + K8s API`
- 对未来 Assistant 的推荐执行方式：`K8s adapter`
- 风险等级建议：`R2 中风险`
- 是否必须审批：建议是
- 是否必须产出证据包：建议是
- 备注 / 待验证点：
  - 当前动作更像“纳管”，不是“创建 storage class”

---

## 15. `resource.hostgroup.register`

- 标准动作名：`resource.hostgroup.register`
- 业务语义说明：注册主机组/节点组。
- 在 upm-api-server 中的入口位置：
  - 当前最接近对象不是 `hostgroup`，而是 `node group`
  - `NodeGroupController.save(...)`
  - Path: `POST /node_grps`
  - 文件：`upm-resource/upm-resource-ms/src/main/java/io/syntropycloud/upm/resource/ms/controller/NodeGroupController.java`
- 关键调用链：
  - `NodeGroupController.save(...)`
  - `NodeGroupServiceImpl.save(NodeGroupForm)`
  - `NodeGroupCheck.checkSave(...)`
  - `nodeGroupDAO.save(nodeGroupDO)`
- 依赖的核心对象：
  - `NodeGroupDO`
  - `ZoneDO`
  - `ClusterDO`
- 是否异步：否
- 是否有 task/order：否 / 否
- 是否有审批痕迹：否
- 是否有操作日志：是，`@OperateLog(action = ADD)`
- 最终执行入口猜测：`内部持久化`
- 对未来 Assistant 的推荐执行方式：`Control Layer internal action`
- 风险等级建议：`R1 低风险写`
- 是否必须审批：建议否
- 是否必须产出证据包：建议否
- 备注 / 待验证点：
  - 如果用户真正指的是“注册主机”，当前对应的是 `NodeController.save(...)`
  - future assistant 需要统一 `hostgroup` / `nodegroup` 词汇

---

## 16. 总结性观察

### 16.1 当前系统里的动作分三类

1. 直接同步动作  
   典型：`mysql.database.create`、`mysql.user.create`

2. 任务型动作  
   典型：`mysql.backup.create`、`mysql.restore.create`、`mysql.unit.rebuild`

3. 工单 + 任务型动作  
   典型：`mysql.service.create`、`mysql.service.scale`

### 16.2 对 future assistant 的直接启发

- 统一动作字典必须比 UPM 当前 controller 更抽象。
- 同一个标准动作最终可以路由到不同执行器：
  - `MCP tool`
  - `CRD adapter`
  - `GrpcCall adapter`
  - `DB-native adapter`
  - `K8s adapter`
- 关键不是“能不能调工具”，而是：
  - 何时需要工单
  - 何时需要 task
  - 何时必须审批
  - 何时必须证据包

### 16.3 推荐的 future action contract

每个标准动作至少应固化以下元数据：

- `action_name`
- `resource_selector_schema`
- `parameter_schema`
- `default_execution_route`
- `risk_level`
- `approval_required`
- `evidence_required`
- `idempotency_strategy`
- `rollback_strategy`

而这正是当前 `upm-api-server` 已经给出雏形、但尚未显式产品化的一层。
