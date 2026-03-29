# Phase 04 - Codex Plan

## 文档作用

由 Codex 在进入 Phase 04 编码前填写，明确 Deep Agent 接入的边界、映射方式和验证方案。

## 当前 phase 目标

让 Deep Agent 通过 Skill / HTTP 安全接入 Control Layer，但不下放执行控制权。

## 必须回应事项

1. auto-chain execute 使用的 principal 策略是什么：
   - 若使用 user principal，`ExecutePolicy` 如何显式放行：
   - 若使用代理身份，是否明确为 `control_executor` 或等价受控 service principal：
2. 如何证明 Agent 仍只是 northbound client，而不是事实上的控制器：
3. 哪些验证能证明 prod 路径不会被 Agent 偷跑 execute：

## 本轮计划

- 当前分支：
- 本轮目标：
- 本轮范围：
- 明确不做：
- 预计修改路径：
- 计划验证命令：
- 请求 Claude review 的重点：
