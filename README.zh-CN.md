# CCUV 官方数据集索引

[English](README.md)

这是 `ccuv dataset` 用于浏览和安装受信任可执行数据集的官方签名索引。索引为每个 Release 资源记录不可变的 SHA-256 和平台信息；CCUV 会同时验证索引签名与下载归档。

## 当前支持的数据集

下表从已签名的 [`catalog/catalog.json`](catalog/catalog.json) 派生。catalog 是唯一事实来源；完整说明和配置指引仍保存在各数据集的 manifest 中。

<!-- datasets:start -->
| 数据集 | 简介 | 仓库 | 支持平台 |
| --- | --- | --- | --- |
| [微信读书 (WeRead)](https://github.com/Cookie-HOO/ccuv-dataset-weread) | 将微信读书的阅读时长和书籍活动可视化。 | Cookie-HOO/ccuv-dataset-weread | darwin/amd64, darwin/arm64, linux/amd64, windows/amd64 |
<!-- datasets:end -->

## 协议

CCUV 验证包含以下精确顶层成员的 envelope：

- Envelope schema：`ccuv.official-dataset-catalog-envelope/v1`
- Body schema：`ccuv.official-dataset-catalog/v1`
- 签名：base64 编码的原始 `Ed25519` 签名
- 被签名内容：`signed` 对象的 RFC 8785 JSON Canonicalization Scheme 表示

schema 位于 [schemas/](schemas/)。每个条目包含本地化的 `title`、用于浏览的一句话 `summary`，以及唯一的 `repository`；平台资源只记录不可变 Release 资产，不重复仓库信息。

## 维护

提交 `catalog/signed.json` 的变更后，必须重新签署整个 catalog，验证结果，并更新 README 中由 catalog 派生的表格。私钥不在仓库中；仅使用受保护的发布环境或维护者 Keychain。

```sh
go run ./cmd/ccuv-catalog sign \
  --body catalog/signed.json \
  --key-id ccuv-official-2026-01 \
  --keychain-account ccuv-official-2026-01 \
  --out catalog/catalog.json

go run ./cmd/ccuv-catalog verify \
  --catalog catalog/catalog.json \
  --public-key keys/ccuv-official-2026-01.pub
```
