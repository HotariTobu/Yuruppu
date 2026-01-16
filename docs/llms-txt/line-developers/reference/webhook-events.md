# LINE Messaging API - Webhook Events Reference

> Complete reference for LINE Messaging API webhook event structures, focusing on group/room events including Join, Member Joined/Left, and Source objects. Based on line-bot-sdk-go v8.18.0.

This document provides detailed information about webhook event structures that occur when bots interact with groups and rooms in LINE Messaging API.

## Table of Contents

- [Common Properties](#common-properties)
- [Source Objects](#source-objects)
- [Group/Room Events](#grouproom-events)
- [Follow/Unfollow Events](#followunfollow-events)
- [Message Events](#message-events)
- [Supporting Types](#supporting-types)
- [Webhook Parsing](#webhook-parsing)

## Common Properties

All webhook events share these common properties:

| Property | Type | Description |
|----------|------|-------------|
| `type` | string | Event type identifier |
| `source` | SourceInterface | Event source (UserSource, GroupSource, or RoomSource) |
| `timestamp` | int64 | Event timestamp in milliseconds since epoch |
| `mode` | EventMode | Event mode: "active" or "standby" |
| `webhookEventId` | string | Unique event identifier in ULID format |
| `deliveryContext` | DeliveryContext | Delivery context including redelivery flag |
| `replyToken` | string | Token for sending reply messages (not available for all events) |

### EventMode

```
"active"   - Normal event mode
"standby"  - Standby mode (bot in standby state)
```

### DeliveryContext

```json
{
  "isRedelivery": false
}
```

| Field | Type | Description |
|-------|------|-------------|
| `isRedelivery` | boolean | Whether this event is being redelivered |

## Source Objects

Source objects identify where the event originated. The `source` field uses one of these types:

### UserSource

Used when event originates from a 1-on-1 chat with a user.

```json
{
  "type": "user",
  "userId": "U1234567890abcdef1234567890abcdef"
}
```

**Go Structure:**
```go
type UserSource struct {
    Source
    UserId string `json:"userId"`  // Required
}
```

### GroupSource

Used when event originates from a group chat.

```json
{
  "type": "group",
  "groupId": "C1234567890abcdef1234567890abcdef",
  "userId": "U1234567890abcdef1234567890abcdef"
}
```

**Go Structure:**
```go
type GroupSource struct {
    Source
    GroupId string `json:"groupId"`           // Required
    UserId  string `json:"userId,omitempty"`  // Optional, included in message events
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `type` | string | Yes | Always "group" |
| `groupId` | string | Yes | Unique group identifier |
| `userId` | string | No | User ID (only present in message events) |

### RoomSource

Used when event originates from a multi-person chat room.

```json
{
  "type": "room",
  "roomId": "R1234567890abcdef1234567890abcdef",
  "userId": "U1234567890abcdef1234567890abcdef"
}
```

**Go Structure:**
```go
type RoomSource struct {
    Source
    UserId  string `json:"userId,omitempty"`  // Optional, only in message events
    RoomId  string `json:"roomId"`            // Required
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `type` | string | Yes | Always "room" |
| `roomId` | string | Yes | Unique room identifier |
| `userId` | string | No | User ID (only present in message events) |

## Group/Room Events

### JoinEvent

Triggered when the bot joins a group or room.

**Example JSON:**
```json
{
  "type": "join",
  "timestamp": 1462629479859,
  "source": {
    "type": "group",
    "groupId": "C1234567890abcdef1234567890abcdef"
  },
  "replyToken": "nHuyWiB7yP5Zw52FIkcQobQuGDXCTA",
  "mode": "active",
  "webhookEventId": "01FZ74A0TDDPYRVKNK77XKC3ZR",
  "deliveryContext": {
    "isRedelivery": false
  }
}
```

**Go Structure:**
```go
type JoinEvent struct {
    Event
    Source              SourceInterface  `json:"source,omitempty"`
    Timestamp           int64            `json:"timestamp"`
    Mode                EventMode        `json:"mode"`
    WebhookEventId      string           `json:"webhookEventId"`
    DeliveryContext     *DeliveryContext `json:"deliveryContext"`
    ReplyToken          string           `json:"replyToken"`
}
```

**Properties:**

| Field | Type | Description |
|-------|------|-------------|
| `type` | string | Always "join" |
| `source` | GroupSource or RoomSource | The group/room the bot joined |
| `replyToken` | string | Token to send reply messages |
| `timestamp` | int64 | When the bot joined (milliseconds) |
| `mode` | EventMode | Event mode |
| `webhookEventId` | string | Unique event ID |
| `deliveryContext` | DeliveryContext | Delivery information |

**Use Case:**
- Send welcome message when bot joins a group
- Initialize group-specific settings
- Log group membership

### MemberJoinedEvent

Triggered when one or more users join a group or room where the bot is already a member.

**Example JSON:**
```json
{
  "type": "memberJoined",
  "timestamp": 1462629479859,
  "source": {
    "type": "group",
    "groupId": "C1234567890abcdef1234567890abcdef"
  },
  "replyToken": "nHuyWiB7yP5Zw52FIkcQobQuGDXCTA",
  "mode": "active",
  "webhookEventId": "01FZ74A0TDDPYRVKNK77XKC3ZR",
  "deliveryContext": {
    "isRedelivery": false
  },
  "joined": {
    "members": [
      {
        "type": "user",
        "userId": "U1234567890abcdef1234567890abcdef"
      },
      {
        "type": "user",
        "userId": "U9876543210abcdef9876543210abcdef"
      }
    ]
  }
}
```

**Go Structure:**
```go
type MemberJoinedEvent struct {
    Event
    Source              SourceInterface  `json:"source,omitempty"`
    Timestamp           int64            `json:"timestamp"`
    Mode                EventMode        `json:"mode"`
    WebhookEventId      string           `json:"webhookEventId"`
    DeliveryContext     *DeliveryContext `json:"deliveryContext"`
    ReplyToken          string           `json:"replyToken"`
    Joined              *JoinedMembers   `json:"joined"`
}

type JoinedMembers struct {
    Members []UserSource `json:"members"`
}
```

**Properties:**

| Field | Type | Description |
|-------|------|-------------|
| `type` | string | Always "memberJoined" |
| `source` | GroupSource or RoomSource | The group/room where members joined |
| `replyToken` | string | Token to send reply messages |
| `joined` | JoinedMembers | Information about joined members |
| `joined.members` | UserSource[] | Array of users who joined |
| `timestamp` | int64 | When members joined (milliseconds) |
| `mode` | EventMode | Event mode |
| `webhookEventId` | string | Unique event ID |
| `deliveryContext` | DeliveryContext | Delivery information |

**Use Case:**
- Welcome new members to the group
- Update member count
- Send group rules or information to new members

### LeaveEvent

Triggered when the bot leaves a group or room (or is removed by admin).

**Example JSON:**
```json
{
  "type": "leave",
  "timestamp": 1462629479859,
  "source": {
    "type": "group",
    "groupId": "C1234567890abcdef1234567890abcdef"
  },
  "mode": "active",
  "webhookEventId": "01FZ74A0TDDPYRVKNK77XKC3ZR",
  "deliveryContext": {
    "isRedelivery": false
  }
}
```

**Go Structure:**
```go
type LeaveEvent struct {
    Event
    Source              SourceInterface  `json:"source,omitempty"`
    Timestamp           int64            `json:"timestamp"`
    Mode                EventMode        `json:"mode"`
    WebhookEventId      string           `json:"webhookEventId"`
    DeliveryContext     *DeliveryContext `json:"deliveryContext"`
}
```

**Properties:**

| Field | Type | Description |
|-------|------|-------------|
| `type` | string | Always "leave" |
| `source` | GroupSource or RoomSource | The group/room the bot left |
| `timestamp` | int64 | When the bot left (milliseconds) |
| `mode` | EventMode | Event mode |
| `webhookEventId` | string | Unique event ID |
| `deliveryContext` | DeliveryContext | Delivery information |

**Note:** No `replyToken` is available since the bot is no longer in the group.

**Use Case:**
- Clean up group-specific data
- Log group departures
- Update statistics

### MemberLeftEvent

Triggered when one or more users leave a group or room where the bot is a member.

**Example JSON:**
```json
{
  "type": "memberLeft",
  "timestamp": 1462629479859,
  "source": {
    "type": "group",
    "groupId": "C1234567890abcdef1234567890abcdef"
  },
  "mode": "active",
  "webhookEventId": "01FZ74A0TDDPYRVKNK77XKC3ZR",
  "deliveryContext": {
    "isRedelivery": false
  },
  "left": {
    "members": [
      {
        "type": "user",
        "userId": "U1234567890abcdef1234567890abcdef"
      }
    ]
  }
}
```

**Go Structure:**
```go
type MemberLeftEvent struct {
    Event
    Source              SourceInterface  `json:"source,omitempty"`
    Timestamp           int64            `json:"timestamp"`
    Mode                EventMode        `json:"mode"`
    WebhookEventId      string           `json:"webhookEventId"`
    DeliveryContext     *DeliveryContext `json:"deliveryContext"`
    Left                *LeftMembers     `json:"left"`
}

type LeftMembers struct {
    Members []UserSource `json:"members"`
}
```

**Properties:**

| Field | Type | Description |
|-------|------|-------------|
| `type` | string | Always "memberLeft" |
| `source` | GroupSource or RoomSource | The group/room where members left |
| `left` | LeftMembers | Information about departed members |
| `left.members` | UserSource[] | Array of users who left |
| `timestamp` | int64 | When members left (milliseconds) |
| `mode` | EventMode | Event mode |
| `webhookEventId` | string | Unique event ID |
| `deliveryContext` | DeliveryContext | Delivery information |

**Note:** No `replyToken` is available for this event.

**Use Case:**
- Update member count
- Clean up user-specific data in the group
- Log member departures

## Follow/Unfollow Events

### FollowEvent

Triggered when a user adds the bot as a friend or unblocks it.

**Example JSON:**
```json
{
  "type": "follow",
  "timestamp": 1462629479859,
  "source": {
    "type": "user",
    "userId": "U1234567890abcdef1234567890abcdef"
  },
  "replyToken": "nHuyWiB7yP5Zw52FIkcQobQuGDXCTA",
  "mode": "active",
  "webhookEventId": "01FZ74A0TDDPYRVKNK77XKC3ZR",
  "deliveryContext": {
    "isRedelivery": false
  },
  "follow": {
    "isUnblocked": false
  }
}
```

**Go Structure:**
```go
type FollowEvent struct {
    Event
    Source              SourceInterface  `json:"source,omitempty"`
    Timestamp           int64            `json:"timestamp"`
    Mode                EventMode        `json:"mode"`
    WebhookEventId      string           `json:"webhookEventId"`
    DeliveryContext     *DeliveryContext `json:"deliveryContext"`
    ReplyToken          string           `json:"replyToken"`
    Follow              *FollowDetail    `json:"follow"`
}

type FollowDetail struct {
    IsUnblocked bool `json:"isUnblocked"`
}
```

**Properties:**

| Field | Type | Description |
|-------|------|-------------|
| `type` | string | Always "follow" |
| `source` | UserSource | The user who followed |
| `replyToken` | string | Token to send reply messages |
| `follow` | FollowDetail | Follow details |
| `follow.isUnblocked` | boolean | True if user unblocked the bot, false if new follow |
| `timestamp` | int64 | When followed (milliseconds) |
| `mode` | EventMode | Event mode |
| `webhookEventId` | string | Unique event ID |
| `deliveryContext` | DeliveryContext | Delivery information |

**Use Case:**
- Welcome new followers
- Register user in database
- Send onboarding messages

### UnfollowEvent

Triggered when a user unfriends or blocks the bot.

**Example JSON:**
```json
{
  "type": "unfollow",
  "timestamp": 1462629479859,
  "source": {
    "type": "user",
    "userId": "U1234567890abcdef1234567890abcdef"
  },
  "mode": "active",
  "webhookEventId": "01FZ74A0TDDPYRVKNK77XKC3ZR",
  "deliveryContext": {
    "isRedelivery": false
  }
}
```

**Go Structure:**
```go
type UnfollowEvent struct {
    Event
    Source              SourceInterface  `json:"source,omitempty"`
    Timestamp           int64            `json:"timestamp"`
    Mode                EventMode        `json:"mode"`
    WebhookEventId      string           `json:"webhookEventId"`
    DeliveryContext     *DeliveryContext `json:"deliveryContext"`
}
```

**Properties:**

| Field | Type | Description |
|-------|------|-------------|
| `type` | string | Always "unfollow" |
| `source` | UserSource | The user who unfollowed |
| `timestamp` | int64 | When unfollowed (milliseconds) |
| `mode` | EventMode | Event mode |
| `webhookEventId` | string | Unique event ID |
| `deliveryContext` | DeliveryContext | Delivery information |

**Note:** No `replyToken` is available since the user blocked the bot.

**Use Case:**
- Mark user as inactive
- Clean up user data
- Log unfollow events

## Message Events

### MessageEvent

Triggered when a user sends a message to the bot.

**Example JSON (Text Message in Group):**
```json
{
  "type": "message",
  "timestamp": 1462629479859,
  "source": {
    "type": "group",
    "groupId": "C1234567890abcdef1234567890abcdef",
    "userId": "U1234567890abcdef1234567890abcdef"
  },
  "replyToken": "nHuyWiB7yP5Zw52FIkcQobQuGDXCTA",
  "mode": "active",
  "webhookEventId": "01FZ74A0TDDPYRVKNK77XKC3ZR",
  "deliveryContext": {
    "isRedelivery": false
  },
  "message": {
    "type": "text",
    "id": "325708",
    "text": "Hello, World!"
  }
}
```

**Go Structure:**
```go
type MessageEvent struct {
    Event
    Source          SourceInterface         `json:"source,omitempty"`
    Timestamp       int64                   `json:"timestamp"`
    Mode            EventMode               `json:"mode"`
    WebhookEventId  string                  `json:"webhookEventId"`
    DeliveryContext *DeliveryContext        `json:"deliveryContext"`
    ReplyToken      string                  `json:"replyToken"`
    Message         MessageContentInterface `json:"message"`
}
```

**Properties:**

| Field | Type | Description |
|-------|------|-------------|
| `type` | string | Always "message" |
| `source` | SourceInterface | Message source (UserSource, GroupSource, or RoomSource) |
| `replyToken` | string | Token to send reply messages |
| `message` | MessageContentInterface | Message content (varies by type) |
| `timestamp` | int64 | When message was sent (milliseconds) |
| `mode` | EventMode | Event mode |
| `webhookEventId` | string | Unique event ID |
| `deliveryContext` | DeliveryContext | Delivery information |

**Message Content Types:**
- `text` - Text messages (TextMessageContent)
- `image` - Image messages (ImageMessageContent)
- `audio` - Audio messages (AudioMessageContent)
- `video` - Video messages (VideoMessageContent)
- `file` - File messages (FileMessageContent)
- `location` - Location messages (LocationMessageContent)
- `sticker` - Sticker messages (StickerMessageContent)

**Use Case:**
- Respond to user messages
- Parse commands
- Process file uploads
- Handle location sharing

## Supporting Types

### JoinedMembers

Container for users who joined a group/room.

```go
type JoinedMembers struct {
    Members []UserSource `json:"members"`
}
```

| Field | Type | Description |
|-------|------|-------------|
| `members` | UserSource[] | Array of users who joined |

### LeftMembers

Container for users who left a group/room.

```go
type LeftMembers struct {
    Members []UserSource `json:"members"`
}
```

| Field | Type | Description |
|-------|------|-------------|
| `members` | UserSource[] | Array of users who left |

### FollowDetail

Additional information about follow events.

```go
type FollowDetail struct {
    IsUnblocked bool `json:"isUnblocked"`
}
```

| Field | Type | Description |
|-------|------|-------------|
| `isUnblocked` | boolean | True if user unblocked the bot, false if new follow |

## Webhook Parsing

### Basic Parsing with Signature Validation

```go
import "github.com/line/line-bot-sdk-go/v8/linebot/webhook"

// Parse incoming webhook request
callback, err := webhook.ParseRequest(channelSecret, httpRequest)
if err != nil {
    // Handle parsing error (invalid signature, malformed JSON, etc.)
    log.Printf("Error parsing webhook: %v", err)
    return
}

// Process events
for _, event := range callback.Events {
    switch e := event.(type) {
    case *webhook.JoinEvent:
        // Bot joined a group/room
        handleJoinEvent(e)
    case *webhook.MemberJoinedEvent:
        // Users joined a group/room
        handleMemberJoined(e)
    case *webhook.LeaveEvent:
        // Bot left a group/room
        handleLeaveEvent(e)
    case *webhook.MemberLeftEvent:
        // Users left a group/room
        handleMemberLeft(e)
    case *webhook.MessageEvent:
        // Message received
        handleMessage(e)
    }
}
```

### Parsing with Options

```go
// Parse with custom options
callback, err := webhook.ParseRequestWithOption(
    channelSecret,
    httpRequest,
    &webhook.ParseOption{
        // Skip signature validation (not recommended for production)
        SkipSignatureValidation: func() bool { return false },
    },
)
```

### Manual Signature Validation

```go
// Validate webhook signature manually
signature := httpRequest.Header.Get("X-Line-Signature")
body, _ := io.ReadAll(httpRequest.Body)

isValid := webhook.ValidateSignature(channelSecret, signature, body)
if !isValid {
    // Invalid signature - reject request
    return
}
```

### Using WebhookHandler

```go
import (
    "net/http"
    "github.com/line/line-bot-sdk-go/v8/linebot/webhook"
)

// Create webhook handler
handler, err := webhook.NewWebhookHandler(channelSecret)
if err != nil {
    log.Fatal(err)
}

// Register event handler
handler.HandleEvents(func(callback *webhook.CallbackRequest, r *http.Request) {
    for _, event := range callback.Events {
        switch e := event.(type) {
        case *webhook.JoinEvent:
            log.Printf("Bot joined: %v", e.Source)
        case *webhook.MemberJoinedEvent:
            log.Printf("Members joined: %d users", len(e.Joined.Members))
        case *webhook.MessageEvent:
            log.Printf("Message from %v: %v", e.Source, e.Message)
        }
    }
})

// Register error handler
handler.HandleError(func(err error, r *http.Request) {
    log.Printf("Webhook error: %v", err)
})

// Use as HTTP handler
http.Handle("/webhook", handler)
http.ListenAndServe(":8080", nil)
```

## Common Patterns

### Extracting Group ID from Events

```go
func getGroupID(source webhook.SourceInterface) (string, bool) {
    if groupSource, ok := source.(*webhook.GroupSource); ok {
        return groupSource.GroupId, true
    }
    return "", false
}

// Usage
if groupID, ok := getGroupID(event.Source); ok {
    log.Printf("Event from group: %s", groupID)
}
```

### Extracting User ID from Events

```go
func getUserID(source webhook.SourceInterface) (string, bool) {
    switch s := source.(type) {
    case *webhook.UserSource:
        return s.UserId, true
    case *webhook.GroupSource:
        return s.UserId, s.UserId != ""
    case *webhook.RoomSource:
        return s.UserId, s.UserId != ""
    }
    return "", false
}

// Usage
if userID, ok := getUserID(event.Source); ok {
    log.Printf("Event from user: %s", userID)
}
```

### Determining Chat Type

```go
func getChatType(source webhook.SourceInterface) string {
    switch source.(type) {
    case *webhook.UserSource:
        return "user"
    case *webhook.GroupSource:
        return "group"
    case *webhook.RoomSource:
        return "room"
    default:
        return "unknown"
    }
}

// Usage
chatType := getChatType(event.Source)
log.Printf("Message from %s chat", chatType)
```

## References

- [LINE Messaging API Official Documentation](https://developers.line.biz/en/docs/messaging-api/)
- [LINE Bot SDK for Go (v8)](https://github.com/line/line-bot-sdk-go)
- [Go Package Documentation](https://pkg.go.dev/github.com/line/line-bot-sdk-go/v8/linebot/webhook)
- [Webhook Event Objects Reference](https://developers.line.biz/en/reference/messaging-api/#webhook-event-objects)

## Notes

- Always validate webhook signatures in production to prevent unauthorized requests
- The `userId` field in GroupSource/RoomSource is only available for message events
- Events without `replyToken` cannot be replied to directly (LeaveEvent, UnfollowEvent, MemberLeftEvent)
- The `webhookEventId` is in ULID format and can be used for deduplication
- Check `deliveryContext.isRedelivery` to handle duplicate event processing
- Event mode can be "active" or "standby" depending on bot configuration
