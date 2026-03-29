# Phase 01 - Codex Plan

## 文档作用

由 Codex 在进入 Phase 01 编码前填写，明确本 phase 的范围、计划顺序、验证方式和禁止事项自检。

## 当前 phase 目标

固化 skeleton、interfaces、guardrails，不进入真实 `mysql.database.create` 执行。

## 已有产物盘点

在开始本 phase 前，必须先盘点当前仓库已有代码与测试，并回答：

- 已存在的 domain objects：
- 已存在的 application interfaces / services：
- 已存在的 northbound API skeleton：
- 已存在的 adapter SPI / `DBNativeAdapter` skeleton：
- 已存在的测试：
- 与 Phase 01 验收标准已对齐的项：
- 仍需补齐的 gap：

## 必须回应事项

1. DryRun 在 Adapter SPI skeleton 中的落点是什么：
2. `AdapterDryRunResult` 是否已存在，若不存在如何补：
3. Skill contract structs 是否完整覆盖：
   - `request_mysql_database_create`
   - `execute_assistant_order`

## 本轮计划

- 当前分支：
- 本轮目标：
- 本轮范围：
- 明确不做：
- 预计修改路径：
- 计划验证命令：
- 请求 Claude review 的重点：
