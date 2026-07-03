# API Specification

Web Security Proxy の REST API 仕様（Phase 1〜3）。

Base URL: `http://localhost:8080`

## GET /api/transactions

通信履歴の一覧を JSON で返します。

### Query Parameters

| Name | Type | Default | Description |
|------|------|---------|-------------|
| `limit` | int | 100 | 最大取得件数 |

### Response `200 OK`

```json
[
  {
    "id": 1,
    "method": "GET",
    "url": "http://example.com/",
    "request_headers": "Host: example.com",
    "request_body": "",
    "status_code": 200,
    "response_headers": "Content-Type: text/html",
    "response_body": "<html></html>",
    "duration_ms": 42,
    "created_at": "2026-07-02T10:00:00Z"
  }
]
```

## GET /api/transactions/{id}

指定 ID の通信詳細を返します。

### Response `200 OK`

単一の `HTTPTransaction` オブジェクト。

### Response `404 Not Found`

```json
{
  "error": "not found"
}
```

## POST /api/repeater/send

編集したリクエストを再送信し、結果を履歴に保存します。

### Request Body

```json
{
  "method": "POST",
  "url": "http://example.com/api",
  "headers": "Content-Type: application/json\nAccept: */*",
  "body": "{\"name\":\"test\"}"
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `method` | string | yes | HTTP メソッド（CONNECT 不可） |
| `url` | string | yes | 送信先 URL |
| `headers` | string | no | `Key: Value` 形式（改行区切り） |
| `body` | string | no | リクエストボディ |

### Response `201 Created`

保存された `HTTPTransaction` オブジェクトを返します。

### Response `400 Bad Request`

バリデーションエラー、ヘッダー形式エラーなど。

## GET /api/transactions/{id}/diagnostics

指定 ID の通信に対するセキュリティ診断結果を返します。

### Response `200 OK`

```json
{
  "server": "nginx/1.18.0",
  "overall_risk": "LOW",
  "findings": [
    {
      "check_name": "security_headers",
      "title": "Missing CSP Header",
      "description": "Content-Security-Policy helps mitigate XSS and data injection attacks.",
      "risk_level": "LOW"
    }
  ],
  "cookies": [
    {
      "name": "session",
      "http_only": false,
      "secure": false,
      "same_site": "",
      "warnings": [
        {
          "check_name": "cookies",
          "title": "Cookie HttpOnly Missing",
          "risk_level": "HIGH"
        }
      ]
    }
  ]
}
```

### Response `404 Not Found`

通信が存在しない場合。

## POST /api/transactions/{id}/active-scan

対象通信の URL に対して能動スキャン（SQLi / XSS / robots.txt / sitemap.xml）を実行します。

### Response `200 OK`

`ActiveScanReport` オブジェクト（`overall_risk`, `findings`, `resources`, `passive` を含む）。

### Response `400 Bad Request`

CONNECT 通信や URL 不正など。

## Web UI

| Path | Description |
|------|-------------|
| `GET /` | 通信一覧（左）と空の詳細（右） |
| `GET /transactions/{id}` | 通信一覧 + 詳細 + Security 診断 + Repeater |

## Error Format

API エラーは以下の形式です。

```json
{
  "error": "message"
}
```
