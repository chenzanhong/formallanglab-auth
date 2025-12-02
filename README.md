# FormalLangLab

FormalLangLab 是一个基于 Web 的交互式形式语言与自动机学习系统，旨在帮助计算机科学专业的学生更直观、高效地理解文法、正则表达式、自动机（DFA/NFA）等核心概念。该项目集成了自动机可视化、文法合法性检查、DFA/NFA 构造与转换、字符串接受判断、LL(1)/LR(1) 分析过程演示等功能，并通过 AI 模块提供智能辅助学习功能。

《支持自动机可视化的形式语言学习系统的设计与实现》


# Auth 服务是一个轻量级的身份认证中心（Identity Provider）
1. 负责
- 用户账户的创建与凭证管理（注册、登录、登出）
- 安全令牌的签发与刷新（Access Token + Refresh Token）
- 敏感操作的二次验证协调（通过异步事件触发验证码发送）

2. 流程
用户登录 → 后端返回 Access Token + Refresh Token（Cookie）。
前端携带 Access Token 调用业务接口 → 成功。
Access Token 过期 → 业务接口返回 401。
前端静默调用 /gdesign/auth/refresh → 后端验证 Refresh Token，返回新的 Access Token 与新的 Refresh Token（更新 Cookie）。
前端用新 Access Token 重试失败请求 → 成功。
用户主动登出 → 调用 /gdesign/auth/logout，后端清除refresh token记录，前端清空本地状态。