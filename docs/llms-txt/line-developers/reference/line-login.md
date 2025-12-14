# LINE Login v2.1 API Reference

## Common Specifications

### Rate Limits
The LINE Login API enforces rate limiting to prevent platform disruption. Specific thresholds are undisclosed. Applications sending excessive requests risk temporary restrictions.

### Status Codes
Standard HTTP status codes apply:
- **200 OK**: Successful request
- **400 Bad Request**: Invalid request parameters or JSON format
- **401 Unauthorized**: Authorization header error
- **403 Forbidden**: Insufficient permissions or plan authorization
- **413 Payload Too Large**: Request exceeds 2MB limit
- **429 Too Many Requests**: Rate limit exceeded
- **500 Internal Server Error**: Temporary server error

### Response Headers
All responses include `x-line-request-id` header containing a unique request identifier.

---

## OAuth Endpoints

### Issue Access Token
**POST** `https://api.line.me/oauth2/v2.1/token`

Exchanges an authorization code for access tokens, enabling apps to access user data including IDs, display names, profile images, and status messages.

**Request Headers:**
- `Content-Type: application/x-www-form-urlencoded`

**Request Body:**
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `grant_type` | String | Yes | Must be `authorization_code` |
| `code` | String | Yes | Authorization code from LINE Platform |
| `redirect_uri` | String | Yes | Must match authorization request value |
| `client_id` | String | Yes | Channel ID from LINE Developers Console |
| `client_secret` | String | Yes | Channel secret from Console |
| `code_verifier` | String | Optional | 43-128 character PKCE verification string |

**Response (200 OK):**
| Property | Type | Description |
|----------|------|-------------|
| `access_token` | String | Valid 30 days |
| `expires_in` | Number | Seconds until expiration |
| `id_token` | String | JWT with user info (if `openid` scope requested) |
| `refresh_token` | String | Valid 90 days; use for token refresh |
| `scope` | String | Granted permissions |
| `token_type` | String | Always `Bearer` |

---

### Verify Access Token Validity
**GET** `https://api.line.me/oauth2/v2.1/verify`

Confirms access token authenticity and status.

**Query Parameters:**
- `access_token` (Required): Token to verify

**Response (200 OK):**
| Property | Type | Description |
|----------|------|-------------|
| `scope` | String | Granted permissions |
| `client_id` | String | Channel ID that issued token |
| `expires_in` | Number | Seconds until expiration |

**Error Response (400):** Returned when token has expired.

---

### Refresh Access Token
**POST** `https://api.line.me/oauth2/v2.1/token`

Obtains new access token using refresh token (valid 90 days post-issuance).

**Request Headers:**
- `Content-Type: application/x-www-form-urlencoded`

**Request Body:**
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `grant_type` | String | Yes | Must be `refresh_token` |
| `refresh_token` | String | Yes | Refresh token (90-day validity) |
| `client_id` | String | Yes | Channel ID |
| `client_secret` | String | Conditional | Required for web-only app types |

**Response (200 OK):**
| Property | Type | Description |
|----------|------|-------------|
| `access_token` | String | New token (30-day validity) |
| `token_type` | String | `Bearer` |
| `refresh_token` | String | Same or new refresh token |
| `expires_in` | Number | Seconds until new token expires |
| `scope` | String | Granted permissions |

**Error Response (400):** Expired refresh token requires user re-authentication.

---

### Revoke Access Token
**POST** `https://api.line.me/oauth2/v2.1/revoke`

Invalidates a user's access token.

**Request Headers:**
- `Content-Type: application/x-www-form-urlencoded`

**Request Body:**
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `access_token` | String | Yes | Token to invalidate |
| `client_id` | String | Yes | Channel ID |
| `client_secret` | String | Conditional | Required for web-only app types |

**Response:** 200 OK with empty body

---

### Deauthorize App
**POST** `https://api.line.me/user/v1/deauthorize`

Revokes app permissions on behalf of user. Applicable to LINE Login, LIFF apps, and LINE MINI Apps.

**Request Headers:**
- `Authorization: Bearer {channel access token}`
  - Accepts v2.1 channel access tokens or stateless tokens

**Request Body:**
| Parameter | Type | Required |
|-----------|------|----------|
| `userAccessToken` | String | Yes |

**Response:** 204 No Content with empty body

**Error Response (400):** User already deauthorized or API already revoked access.

---

### Verify ID Token
**POST** `https://api.line.me/oauth2/v2.1/verify`

Validates ID token authenticity before using profile/email data.

**Request Headers:**
- `Content-Type: application/x-www-form-urlencoded`

**Request Body:**
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `id_token` | String | Yes | Token to verify |
| `client_id` | String | Yes | Expected channel ID |
| `nonce` | String | Optional | Expected nonce from auth request |
| `user_id` | String | Optional | Expected user ID |

**Response (200 OK) - ID Token Payload:**
| Property | Type | Description |
|----------|------|-------------|
| `iss` | String | Token issuer URL |
| `sub` | String | User ID |
| `aud` | String | Channel ID |
| `exp` | Number | UNIX expiration timestamp |
| `iat` | Number | UNIX issuance timestamp |
| `auth_time` | Number | User authentication time (if `max_age` specified) |
| `nonce` | String | Provided nonce value (if specified) |
| `amr` | Array | Auth methods: `pwd`, `lineautologin`, `lineqr`, `linesso`, `mfa` |
| `name` | String | Display name (if `profile` scope granted) |
| `picture` | String | Profile image URL (if `profile` scope granted) |
| `email` | String | Email address (if `email` scope granted) |

**Error Responses:**
- "Invalid IdToken": Malformed or invalid signature
- "Invalid IdToken Issuer": Generated outside LINE platform
- "IdToken expired": Token has expired
- "Invalid IdToken Audience": Channel ID mismatch
- "Invalid IdToken Nonce": Nonce value mismatch
- "Invalid IdToken Subject Identifier": User ID mismatch

---

### Get User Information
**GET/POST** `https://api.line.me/oauth2/v2.1/userinfo`

Retrieves user ID, display name, and profile image. Requires `openid` scope.

**Request Headers:**
- `Authorization: Bearer {access token}`

**Response:**
| Property | Type | Description |
|----------|------|-------------|
| `sub` | String | User ID |
| `name` | String | Display name (if `profile` scope granted) |
| `picture` | String | Profile image URL (if `profile` scope granted) |

---

## Profile Endpoints

### Get User Profile
**GET** `https://api.line.me/v2/profile`

Retrieves complete profile: user ID, display name, profile image, and status message. Requires `profile` scope.

**Request Headers:**
- `Authorization: Bearer {access token}`

**Response:**
| Property | Type | Description |
|----------|------|-------------|
| `userId` | String | User ID |
| `displayName` | String | Display name |
| `pictureUrl` | String | HTTPS profile image URL |
| `statusMessage` | String | User status (if set) |

**Profile Image Thumbnails:**
- `/large`: 200×200 pixels
- `/small`: 51×51 pixels

---

## Friendship Status Endpoint

### Get Friendship Status
**GET** `https://api.line.me/friendship/v1/status`

Determines if user has added the linked LINE Official Account as friend. Requires `profile` scope.

**Request Headers:**
- `Authorization: Bearer {access token}`

**Response:**
| Property | Type | Description |
|----------|------|-------------|
| `friendFlag` | Boolean | `true` if user added bot and not blocked; `false` otherwise |
