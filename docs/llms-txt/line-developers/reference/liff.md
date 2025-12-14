# LIFF v2 API Reference

## Common Specifications

### Operating Environment
LIFF v2 supports iOS, Android, and web browsers. Feature availability depends on whether the app runs in a LIFF browser or external browser. "LIFF apps are not compatible with OpenChat."

### Error Handling
Errors return `LiffError` objects with `code`, `message`, and `cause` properties. Error identification should reference both code and message rather than message text alone, as messages may change.

## LIFF SDK Properties

### liff.id
Holds the LIFF app ID passed to `liff.init()`. Value is `null` until initialization completes.

### liff.ready
A `Promise` that resolves when `liff.init()` completes for the first time. "You can use `liff.ready` even before the initialization of the LIFF app by `liff.init()` has finished."

## Initialization

### liff.init(config, successCallback, errorCallback)
Initializes the LIFF app and obtains access/ID tokens.

**Config parameters:**
- `liffId` (String, required): LIFF app ID
- `withLoginOnExternalBrowser` (Boolean, optional): Auto-execute login in external browsers (default: `false`)

**Return value:** `Promise` object

**Important notes:**
- Execute at endpoint URL or lower-level paths only
- Process URL changes after Promise resolves
- Don't send primary redirect URLs to analytics tools
- Execute once for both primary and secondary redirect URLs

## Getting Environment Information

### liff.getOS()
Returns device OS: `"ios"`, `"android"`, or `"web"`. Usable before initialization.

### liff.getAppLanguage()
Returns LINE app language setting (RFC 5646 format). Requires v2.24.0+. Usable before initialization.

### liff.getLanguage() [Deprecated]
Returns `navigator.language` value. Use `getAppLanguage()` instead.

### liff.getVersion()
Returns LIFF SDK version string. Usable before initialization.

### liff.getLineVersion()
Returns LINE app version in LIFF browser, `null` in external browsers. Usable before initialization.

### liff.getContext()
Returns context object with: `type` (utou/group/room/external/none), `userId`, `liffId`, `viewType` (compact/tall/full), `endpointUrl`, `accessTokenHash`, `availability`, `scope`, `menuColorSetting`, `miniAppId`, `miniDomainAllowed`, `permanentLinkPattern`.

### liff.isInClient()
Returns `true` if running in LIFF browser, `false` otherwise. Usable before initialization.

### liff.isLoggedIn()
Returns `true` if user is logged in, `false` otherwise.

### liff.isApiAvailable(apiName)
Checks if specified API is available. Accepts: `shareTargetPicker`, `createShortcutOnHomeScreen`, `multipleLiffTransition`. Returns boolean.

## Authentication

### liff.login(loginConfig)
Performs login in external browser or LINE's in-app browser.

**Parameters:**
- `loginConfig.redirectUri` (String, optional): URL after login (default: endpoint URL)

**Note:** "If the URL specified in `redirectUri` doesn't start with the URL specified in **Endpoint URL**, the login process fails."

### liff.logout()
Logs out the user.

### liff.getAccessToken()
Returns current user's access token string. Token valid for 12 hours; may revoke when closing LIFF app.

### liff.getIDToken()
Returns ID token (JWT) containing user data. Requires `openid` scope.

### liff.getDecodedIDToken()
Returns ID token payload with user info (displayName, pictureUrl, email, etc.). Requires `openid` scope. "You can only get the main profile information. You can't get the user's subprofile."

### liff.permission.getGrantedAll()
Returns array of scopes user granted: `profile`, `chat_message.write`, `openid`, `email`.

### liff.permission.query(permission)
Checks if user granted specific permission. Returns object with `state`: `granted`/`prompt`/`unavailable`.

### liff.permission.requestAll()
Displays verification screen for LINE MINI Apps. Requires Channel consent simplification enabled. Returns `Promise`.

## Profile

### liff.getProfile()
Returns user profile object: `userId`, `displayName`, `pictureUrl` (optional), `statusMessage` (optional). Requires `profile` scope.

### liff.getFriendship()
Returns `friendFlag` boolean indicating if user added linked LINE Official Account as friend. Requires `profile` scope.

## Window

### liff.openWindow(params)
Opens URL in LINE's in-app browser or external browser.

**Parameters:**
- `url` (String, required): Full URL
- `external` (Boolean, optional): Open externally (default: `false`)

### liff.closeWindow()
Closes LIFF app. Usable before initialization (v2.4.0+). "Not guaranteed to work in external browser."

## Message

### liff.sendMessages(messages)
Sends up to 5 messages to current chat. Requires `chat_message.write` scope and full-size LIFF browser.

**Supported message types:** text, sticker, image, video, audio, location, template, flex (URI actions only).

**Return value:** `Promise` resolved on success, rejected with `LiffError` on failure.

### liff.shareTargetPicker(messages, options)
Displays target picker for group/friend selection and sends up to 5 messages.

**Parameters:**
- `messages` (Array): Message objects
- `options.isMultiple` (Boolean, optional): Allow multiple recipients (default: `true`)

**Return value:** `Promise` with `{status: "success"}` on send, resolved with no value if cancelled, rejected on error.

## Camera

### liff.scanCodeV2()
Launches 2D code reader. Works on iOS 14.3+, all Android versions, and WebRTC-supporting browsers.

**Return value:** `Promise` resolving to `{value: "scanned string"}`

### liff.scanCode() [Deprecated]
Legacy 2D code reader. Use `scanCodeV2()` instead. Only works on Android LIFF browser.

## Permanent Link

### liff.permanentLink.createUrlBy(url)
Gets permanent link for specified URL. URL must start with endpoint URL.

**Return value:** `Promise` resolving to permanent link string (format: `https://liff.line.me/{liffId}/{path}?{query}#{fragment}`)

### liff.permanentLink.createUrl()
Gets permanent link for current page. May deprecate; use `createUrlBy()` instead.

**Return value:** Permanent link string; throws `LiffError` if URL doesn't match endpoint URL.

### liff.permanentLink.setExtraQueryParam(extraString)
Adds query parameters to current page's permanent link. Overwrites previous parameters. Pass `""` to delete.

## LIFF Plugin

### liff.use(module, option)
Activates LIFF API module or plugin.

**Parameters:**
- `module` (Object, required): LIFF API or plugin (must be instantiated if class)
- `option` (Any, optional): Value passed to plugin's `install()` method

**Return value:** `liff` object
