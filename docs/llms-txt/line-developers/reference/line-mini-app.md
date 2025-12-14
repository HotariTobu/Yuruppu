# LINE MINI App API Reference

## Service Messages

Service Messages enable verified MINI Apps to send notifications to users. The feature requires a service notification token and pre-registered templates.

### Issue Service Notification Token

**Endpoint:** `POST https://api.line.me/message/v3/notifier/token`

**Headers:**
- `Content-Type: application/json`
- `Authorization: Bearer {channel access token}`

**Request Body:**
- `liffAccessToken` (string, required): User access token from `liff.getAccessToken()`

**Response (200):**
```json
{
  "notificationToken": "string",
  "expiresIn": 31536000,
  "remainingCount": 5,
  "sessionId": "string"
}
```

**Token Features:**
- Expires 1 year after issuance
- Permits up to 5 message sends
- Renews with each use (if valid)
- One token per LIFF access token only

**Error Responses:**
- `400`: Malformed request or token rate-limited
- `401`: Invalid channel or LIFF access token
- `403`: Channel not authorized
- `500`: Server error

---

### Send Service Message

**Endpoint:** `POST https://api.line.me/message/v3/notifier/send?target=service`

**Headers:**
- `Content-Type: application/json`
- `Authorization: Bearer {channel access token}`

**Request Body:**
- `templateName` (string, required): Format `{name}_{BCP47-tag}` (max 30 chars)
- `params` (object, required): Template variables as key-value pairs
- `notificationToken` (string, required): Service notification token

**Supported Languages:** Arabic, Chinese (Simplified/Traditional), English, French, German, Indonesian, Italian, Japanese, Korean, Malay, Portuguese (Brazil/Portugal), Russian, Spanish, Thai, Turkish, Vietnamese

**Response (200):**
```json
{
  "notificationToken": "string",
  "expiresIn": 31536000,
  "remainingCount": 4,
  "sessionId": "string"
}
```

**Error Responses:**
- `400`: Invalid request or non-existent recipient
- `401`: Invalid tokens
- `403`: Unauthorized channel or template not found

---

## Common Profile Quick-fill

Quick-fill automates form completion using user profile data from LINE's Account Center. Requires verified MINI App status and prior approval.

### liff.$commonProfile.get()

Retrieves user's Common Profile with confirmation modal.

**Syntax:** `liff.$commonProfile.get(scopes, options)`

**Parameters:**
- `scopes` (array, required): Profile data types to retrieve
- `options.formatOptions` (object, optional): Data formatting rules

**formatOptions Properties:**
- `excludeEmojis` (boolean): Remove emoji characters (default: true)
- `excludeNonJp` (boolean): Reject 12+ digit phone numbers (default: true)
- `digitsOnly` (boolean): Remove non-numeric postal code characters (default: true)

**Returns:** `Promise<{ data: Partial<CommonProfile>, error: Partial<CommonProfileError> }>`

---

### liff.$commonProfile.getDummy()

Retrieves dummy Common Profile data for testing (10 cases available).

**Syntax:** `liff.$commonProfile.getDummy(scopes, options, caseId)`

**Parameters:**
- `scopes` (array, required): Profile data types
- `options` (object, optional): Formatting rules
- `caseId` (number, required): Dummy data ID (1-10)

**Returns:** `Promise<{ data: Partial<CommonProfile>, error: Partial<CommonProfileError> }>`

---

### liff.$commonProfile.fill()

Auto-populates form fields using obtained profile data via `data-liff-autocomplete` HTML attributes.

**Syntax:** `liff.$commonProfile.fill(profile)`

**Parameters:**
- `profile` (Partial<CommonProfile>, required): Profile data to populate

**Returns:** None

**Implementation Note:** "The value specified in the `data-liff-autocomplete` attribute of the form must match the scope of the profile information obtained"
