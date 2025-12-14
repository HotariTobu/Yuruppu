# LINE Social Plugins: Share Button Documentation

## Overview

The Share button from LINE Social Plugins enables websites to add sharing functionality. "You can easily create and add the Share button from LINE Social Plugins to the website of your choice."

For native app integration (iOS/Android), the documentation recommends using the "Share with" screen via the LINE URL Scheme.

## Implementation Methods

### Official LINE Icons

The default approach requires selecting a language, specifying a webpage URL, and choosing a button design from LY Corporation's provided options.

### Custom Icons

Custom icons are available after reviewing the LINE Social Plugins usage guidelines. The implementation follows this format:

**URL Structure:**
```
https://social-plugins.line.me/lineit/share?url=[ENCODED_URL]&text=[TEXT]
```

**Example:**
```
https://social-plugins.line.me/lineit/share?url=https%3A%2F%2Fline.me%2Fen&text=text
```

**Activation:**
```javascript
<script type="text/javascript">LineIt.loadButton();</script>
```

## Share Count API

### Request
```
GET https://api.line.me/social-plugin/metrics?url=[URL]
```

### Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| url | String | Yes | "The URL to get the share count for" |

### Example Request
```bash
curl -X GET 'https://api.line.me/social-plugin/metrics?url=https://line.me/en'
```

### Response
```json
{
  "share": "4173"
}
```

### Status Codes

- **200**: Request succeeded
- **400**: Invalid parameters or values
- **500**: Internal server error
