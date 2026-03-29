# Phase 02 - Codex Plan

## 文档作用

由 Codex 在进入 Phase 02 编码前填写，明确最小控制链路的接线计划和验证路径。

## 当前 phase 目标

打通 request、authorization、order、approval、execute、audit/evidence 基础链路，但不做真实 MySQL 写操作。

## 必须回应事项

1. `RiskEngine.Evaluate()` 如何根据 asset environment 动态给出 `R1/R2`：
2. `Plan Revalidate` 在本阶段是否会调用 `Adapter.DryRun()`，如果会，调用位置是什么；如果不会，stub 如何保留：
3. `PLAN_STALE` 路径的 `EvidencePack` 如何落地，尤其是 `task_id = null` 和 stale reason：

## 本轮计划

- 当前分支：
- 本轮目标：
- 本轮范围：
- 明确不做：
- 预计修改路径：
- 计划验证命令：
- 请求 Claude review 的重点：
