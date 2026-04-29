package main

// @title TeslaMateApi
// @version 1.0
// @description 基于 TeslaMate 数据库的 REST API，提供车辆、充电、行程、统计、图表、时间线和洞察数据。原 TeslaMateApi 兼容接口保持响应结构；扩展接口按当前项目最佳实践重新设计，可能包含破坏性变更。日期参数支持 RFC3339、带时区偏移、本地日期时间和纯日期；未带时区的本地日期使用环境变量配置的时区。
// @BasePath /api
// @schemes http https
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
//
// @tag.name Compatible API
// @tag.description 原 TeslaMateApi 兼容路由，尽量保持历史响应形态。部分历史处理器失败时仍可能返回 HTTP 200 和包含 error 字段的 JSON。
// @tag.name Extended API
// @tag.description 重新设计的扩展路由，按摘要、仪表盘、实时、时序、分布、洞察、时间线、地图等职责拆分。
// @tag.name System
// @tag.description 健康检查、连通性检查和文档入口。
