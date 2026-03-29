# Phase 05 - MCP Integration

## 目标

增加 `MCPToolAdapter` 作为第二条 southbound 执行路线，验证 Control Layer 对不同执行通道仍保持同一套权限、审批、审计、证据语义。

## 范围

1. `MCPToolAdapter` 最小 SPI 实现。
2. capability check、route selection 与结果归一化。
3. 至少一个已存在动作的 MCP 路线接入。
4. `AuditEvent.selected_adapter`、`EvidencePack`、错误码映射对 MCP 路线保持一致。
5. 保留 `DBNativeAdapter` 作为基线对照通道。

## 禁止事项

1. 不把 MCP 当成 Control Layer 替代品。
2. 不让 Deep Agent 直接绕过 Control Layer 调 MCP tool。
3. 不为了适配 MCP tool 而修改核心状态机或审批语义。
4. 不顺手扩散到多动作、多 server、大量工具接入。

## 产物

1. `MCPToolAdapter` 最小实现。
2. route 配置与 capability 探测。
3. 同一动作的多通道路由能力或等价兼容验证。
4. MCP 兼容测试与使用说明。

## 验收标准

1. 同一 northbound 请求仍先经过 Control Layer，再路由到 MCP。
2. MCP 路线同样要求：
   - `AuthorizationDecision`
   - 审批
   - explicit execute
   - 审计
   - 证据
3. adapter 切换后，上层 skill / API 契约不变。
4. 路由结果、适配器类型、失败原因可在审计中查询。
5. MCP 路线不会削弱现有 `DBNativeAdapter` 已验证的控制门禁。

## 风险点

1. 企业内现成 MCP tool 的输入输出契约可能不稳定。
2. MCP tool 返回结果的可审计性和证据密度未必与 DB-native 一致。
3. route 规则设计不稳时，容易在不同 adapter 之间引入不可解释差异。

## 进入下一阶段条件

1. 至少一个 MCP 路线经过完整 control chain 验证。
2. DB-native 与 MCP 两条路径在核心治理语义上保持一致。
3. 团队确认 MCP 接入增加的是执行通道，而不是新的控制面。

## 推荐 branch 名

`phase/05-mcp-compat`

## 推荐 commit message 模式

1. `feat(mcp): add mcp tool adapter`
2. `refactor(router): support dbnative and mcp route selection`
3. `test(mcp): verify control-layer parity on mcp route`
4. `docs(mcp): add adapter capability and usage notes`

