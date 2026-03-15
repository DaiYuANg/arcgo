Redis / Valkey 统一访问框架设计文档
1. 项目概述

本项目旨在构建一个面向 Redis 与 Valkey 的统一访问框架，为 Go 应用提供：

强类型对象访问能力

类似 Spring Data Redis Hash 的对象映射体验

Repository 风格的数据访问接口

常见高级能力（Pub/Sub、Stream、JSON、Search、Lua）的统一访问入口

可扩展、模块化的能力架构

该框架定位为：

面向 Redis / Valkey 的对象访问与能力统一框架，而非通用缓存后端抽象层。

2. 设计动机

Redis 与 Valkey 在协议语义、核心数据结构与使用模式上具有高度一致性。
在现代 Go 服务架构中，Redis / Valkey 常承担：

缓存存储

对象存储

消息传递

事件流

查询与搜索

分布式协调

然而，Go 生态中缺乏一个：

强类型

泛型优先

模块化

面向对象模型

能力统一

同时支持 Redis / Valkey

的访问框架。

本项目即为解决该问题而设计。

3. 设计目标与非目标
   3.1 核心目标

提供统一上层抽象，屏蔽 Redis 与 Valkey 驱动差异。

提供类似 Spring Data Redis Hash 的对象映射能力。

支持基于 struct tag 的元信息声明：

主键字段

字段映射

忽略字段

TTL 字段

索引字段

提供 Repository 风格 API 与便捷查询能力。

支持多种对象存储模型：

Hash 模型

JSON 文档模型

提供统一访问入口以支持以下高级能力：

Pub/Sub

Stream

JSON

Search

Lua / Function

采用模块化架构，使不同能力域可独立扩展。

采用泛型优先设计，为业务层提供强类型开发体验。

3.2 非目标

不兼容 Memcache、数据库、对象存储等异构后端。

不实现传统关系型 ORM 能力，例如：

Join

关系导航

延迟加载

多表事务建模

不追求完整复刻 Spring Data 全家桶功能。

不实现重量级自动机制，例如：

自动实体扫描

自动索引迁移

自动 schema 管理

不强行将所有高级能力整合进单一 Client 或单一 Repository 接口。

初期不覆盖复杂查询优化与分布式一致性语义。

4. 核心设计原则
   4.1 泛型优先

面向业务的 API 应具备强类型体验。

Repository、消息访问、流处理等应采用泛型建模。

泛型用于提升开发体验与编译期安全性。

4.2 底层能力抽象稳定

核心能力接口应保持简洁。

避免与具体客户端强耦合。

避免将对象模型泄漏到底层命令抽象层。

4.3 模块化能力域

不同能力域应拆分为独立模块：

Object Mapping

Pub/Sub

Stream

JSON

Search

Script

模块之间通过核心抽象层协作，而非直接耦合。

4.4 对象映射优先于命令封装

框架重点是对象访问体验，而非命令透传封装。

避免退化为“Redis 客户端包装器”。

4.5 Redis / Valkey 语义优先

框架只针对 Redis / Valkey 能力模型设计。

不为兼容其他后端牺牲语义完整性。

5. 总体架构

框架采用五层结构：

Core Client Abstraction Layer

Object Mapping Layer

Repository Layer

Feature Modules Layer

Adapter Layer

6. Core Client Abstraction Layer

该层提供最基础的后端能力抽象。

目标：

屏蔽 Redis / Valkey 驱动差异

提供稳定能力接口

不涉及业务模型与对象类型

能力域包括：

KV 操作

Hash 操作

Pipeline

Lock

Pub/Sub 原始能力

Stream 原始能力

Script 执行

JSON 原始操作

Search 原始操作

该层为整个框架的稳定基础。

7. Object Mapping Layer

该层负责将 Go 对象与 Redis / Valkey 存储模型进行映射。

职责包括：

struct tag 解析

元信息缓存

主键字段识别

TTL 字段识别

索引字段识别

Key 构建策略

对象序列化与反序列化策略

Hash 编解码

JSON 编解码

该层是：

框架最核心的领域模型层。

8. Repository Layer

该层提供面向对象的数据访问接口。

设计目标：

提供类似 Spring Data Hash 的访问体验

提供统一对象生命周期操作：

Save

FindByID

Delete

Exists

提供批量操作能力

提供索引字段查询入口

提供搜索能力集成入口

Repository 分为不同类型：

Hash Repository

JSON Repository

Indexed Repository

Search Repository

Repository 仅负责对象访问，不直接暴露底层命令语义。

9. Feature Modules Layer

该层负责组织 Redis / Valkey 的高级能力。

每个能力域应独立模块化设计。

9.1 Pub/Sub 模块

职责：

消息发布

消息订阅

Typed Message 支持

消息编解码策略

该模块不与 Repository 强耦合。

9.2 Stream 模块

职责：

Stream 写入

Consumer Group 支持

Typed Message 映射

Stream Offset 管理

消息确认语义

Stream 属于事件流模型，应独立于对象存储模型。

9.3 JSON 模块

职责：

JSON 文档存储

JSON 局部更新

JSON Path 操作

文档对象映射

JSON 与 Hash 模型语义不同，应独立实现。

9.4 Search 模块

职责：

文本搜索

数值过滤

标签过滤

排序

分页

聚合

Search 模块应作为：

对 Hash / JSON 模型的增强查询层。

9.5 Script 模块

职责：

Lua 执行

Function 调用

原子操作封装

批量逻辑执行

Script 模块不直接绑定对象模型。

10. Adapter Layer

该层负责具体客户端适配。

目标：

隔离 Redis 与 Valkey 客户端差异

提供统一能力接口实现

支持未来替换驱动

建议提供：

Redis Adapter

Valkey Adapter

Adapter 层不得向上暴露具体客户端类型。

11. 对象建模与索引策略

框架支持基于字段的轻量索引机制。

设计原则：

索引为可选增强能力

不强制所有字段建立索引

索引策略应简单透明

索引维护由 Repository 控制

Search 模块可作为复杂查询增强手段。

12. 查询模型设计

框架支持三类查询方式：

主键查询

索引字段查询

Search 查询

Repository 不承担复杂查询 DSL 的职责。
复杂查询应通过 Search 模块实现。

13. Key 设计原则

Key 构建应统一管理。

原则：

不允许业务层手动拼接 Key

Key 应由 Repository 或 KeyBuilder 生成

Key 格式稳定可预测

支持多租户或命名空间扩展

14. 序列化策略

框架支持可配置序列化器。

目标：

支持 JSON、MsgPack、Proto 等编码方式

Hash 与 JSON 编解码策略分离

Typed Codec 支持

序列化层属于 Object Mapping 子系统。

15. 错误模型

框架应定义统一错误语义：

Not Found

Invalid Model

Missing Primary Key

Serialization Error

Backend Error

Adapter 层负责转换底层错误。

16. 可观测性与扩展性

未来应支持：

Metrics Hook

Tracing Hook

Logging Hook

Retry Policy

Circuit Breaker

Capability Detection

这些能力应通过扩展机制提供。

17. 演进路线
    Phase 1

Core 抽象

Hash Mapping

基础 Repository

Redis Adapter

Phase 2

Valkey Adapter

JSON 模块

Pub/Sub 模块

Phase 3

Stream 模块

Script 模块

轻量索引机制

Phase 4

Search 模块

可观测性扩展

高级查询能力

18. 最终定位

本框架定位为：

面向 Redis / Valkey 的强类型对象访问与能力统一框架。

其核心价值在于：

提供现代 Go 风格的数据访问体验

构建统一能力模型

提升复杂系统中 Redis / Valkey 的工程可维护性

为后续高级能力扩展提供稳定基础