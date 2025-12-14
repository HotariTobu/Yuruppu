# LINE Messaging API SDK for Go

> Official LINE Bot SDK for Go (v8+) that enables rapid bot development using the LINE Messaging API. Supports webhook event handling, message sending (reply/push/broadcast), rich menus, LIFF apps, and analytics. Requires Go 1.24+.

The LINE Messaging API SDK for Go simplifies building LINE bots by providing type-safe Go interfaces for all LINE platform features including messaging, webhooks, user management, rich menus, LIFF (LINE Front-end Framework), and analytics.

## Quick Start

- [Installation](#installation): Get started with the SDK
- [Basic Bot Example](#basic-bot-example): Echo bot implementation
- [Core Concepts](#core-concepts): Understanding the SDK architecture

## Core Features

- [Webhook Event Handling](#webhook-event-handling): Parse and handle LINE webhook events
- [Sending Messages](#sending-messages): Reply, push, broadcast, multicast, and narrowcast
- [Message Types](#message-types): Text, images, videos, templates, flex messages
- [User & Profile Management](#user-and-profile-management): Get user profiles and followers
- [Rich Menu Management](#rich-menu-management): Create and manage interactive rich menus
- [Group & Room Management](#group-and-room-management): Manage groups and multi-person chats

## Advanced Features

- [LIFF Apps](#liff-apps): Create and manage LINE Front-end Framework applications
- [Analytics & Insights](#analytics-and-insights): Access statistics and user demographics
- [Error Handling](#error-handling): Handle errors and access response headers
- [Custom Configuration](#custom-configuration): Configure HTTP clients and endpoints

## API Reference

- [Webhook Events](#webhook-events-reference): All event types and their properties
- [Message Actions](#message-actions-reference): Available action types for interactive messages
- [API Methods](#api-methods-reference): Complete method reference

## Optional

- [Official Documentation](https://developers.line.biz/en/docs/messaging-api/overview/): LINE Developer documentation
- [Go Package Reference](https://pkg.go.dev/github.com/line/line-bot-sdk-go/v8/linebot): Complete API documentation
- [GitHub Repository](https://github.com/line/line-bot-sdk-go): Source code and examples
- [FAQ](https://developers.line.biz/en/faq/): Frequently asked questions
- [News](https://developers.line.biz/en/news/): Latest updates

---

## Installation

Install the SDK using Go modules:

```bash
go get -u github.com/line/line-bot-sdk-go/v8/linebot
```

**Requirements:** Go 1.24 or later

### Package Imports

The SDK provides multiple sub-packages for different functionalities:

```go
import (
    "github.com/line/line-bot-sdk-go/v8/linebot"
    "github.com/line/line-bot-sdk-go/v8/linebot/messaging_api"
    "github.com/line/line-bot-sdk-go/v8/linebot/webhook"
    "github.com/line/line-bot-sdk-go/v8/linebot/liff"
    "github.com/line/line-bot-sdk-go/v8/linebot/insight"
    "github.com/line/line-bot-sdk-go/v8/linebot/channel_access_token"
    "github.com/line/line-bot-sdk-go/v8/linebot/manage_audience"
    "github.com/line/line-bot-sdk-go/v8/linebot/module"
    "github.com/line/line-bot-sdk-go/v8/linebot/module_attach"
    "github.com/line/line-bot-sdk-go/v8/linebot/shop"
)
```

## Basic Bot Example

A simple echo bot that replies to user messages:

```go
package main

import (
    "errors"
    "fmt"
    "log"
    "net/http"
    "os"

    "github.com/line/line-bot-sdk-go/v8/linebot/messaging_api"
    "github.com/line/line-bot-sdk-go/v8/linebot/webhook"
)

func main() {
    channelSecret := os.Getenv("LINE_CHANNEL_SECRET")
    bot, err := messaging_api.NewMessagingApiAPI(
        os.Getenv("LINE_CHANNEL_TOKEN"),
    )
    if err != nil {
        log.Fatal(err)
    }

    // Webhook handler
    http.HandleFunc("/callback", func(w http.ResponseWriter, req *http.Request) {
        cb, err := webhook.ParseRequest(channelSecret, req)
        if err != nil {
            if errors.Is(err, webhook.ErrInvalidSignature) {
                w.WriteHeader(400)
            } else {
                w.WriteHeader(500)
            }
            return
        }

        // Handle events
        for _, event := range cb.Events {
            switch e := event.(type) {
            case webhook.MessageEvent:
                switch message := e.Message.(type) {
                case webhook.TextMessageContent:
                    // Echo the message
                    bot.ReplyMessage(
                        &messaging_api.ReplyMessageRequest{
                            ReplyToken: e.ReplyToken,
                            Messages: []messaging_api.MessageInterface{
                                messaging_api.TextMessage{
                                    Text: message.Text,
                                },
                            },
                        },
                    )
                }
            }
        }
    })

    port := os.Getenv("PORT")
    if port == "" {
        port = "5000"
    }
    http.ListenAndServe(":"+port, nil)
}
```

**Environment Variables:**
- `LINE_CHANNEL_SECRET`: Your channel's webhook signature key
- `LINE_CHANNEL_TOKEN`: Your channel access token
- `PORT`: Server port (defaults to 5000)

## Core Concepts

### Client Initialization

Create API clients with your channel token:

```go
// Messaging API client
messagingClient, err := messaging_api.NewMessagingApiAPI(
    os.Getenv("LINE_CHANNEL_TOKEN"),
)

// Blob API client (for content operations)
blobClient := messaging_api.NewMessagingApiBlobAPI(
    os.Getenv("LINE_CHANNEL_TOKEN"),
)

// LIFF API client
liffClient, err := liff.NewLiffAPI(
    os.Getenv("LINE_CHANNEL_TOKEN"),
)

// Insight API client
insightClient, err := insight.NewInsightAPI(
    os.Getenv("LINE_CHANNEL_TOKEN"),
)
```

### Webhook Flow

1. **Receive webhook request** from LINE platform
2. **Validate signature** using channel secret
3. **Parse events** using `webhook.ParseRequest()`
4. **Handle events** with type switching
5. **Send responses** using reply token or user ID

### Message Sending Patterns

**Reply Messages:** Use reply tokens (available for 30 seconds after event)
```go
bot.ReplyMessage(&messaging_api.ReplyMessageRequest{
    ReplyToken: replyToken,
    Messages:   []messaging_api.MessageInterface{...},
})
```

**Push Messages:** Send directly to users using their ID
```go
bot.PushMessage(&messaging_api.PushMessageRequest{
    To:       userID,
    Messages: []messaging_api.MessageInterface{...},
}, "")
```

**Broadcast:** Send to all followers
```go
bot.Broadcast(&messaging_api.BroadcastRequest{
    Messages: []messaging_api.MessageInterface{...},
}, "")
```

## Webhook Event Handling

### Parsing Webhook Requests

Use `webhook.ParseRequest()` to parse and validate incoming webhooks:

```go
import "github.com/line/line-bot-sdk-go/v8/linebot/webhook"

cb, err := webhook.ParseRequest(channelSecret, req)
if err != nil {
    if errors.Is(err, webhook.ErrInvalidSignature) {
        // Invalid signature
        w.WriteHeader(400)
    } else {
        // Other error
        w.WriteHeader(500)
    }
    return
}

// Process events
for _, event := range cb.Events {
    handleEvent(event)
}
```

### Event Type Switching

Handle different event types using type assertions:

```go
func handleEvent(event webhook.EventInterface) {
    switch e := event.(type) {
    case webhook.MessageEvent:
        // User sent a message
        handleMessage(e)

    case webhook.FollowEvent:
        // User added bot as friend
        handleFollow(e)

    case webhook.UnfollowEvent:
        // User blocked bot
        handleUnfollow(e)

    case webhook.JoinEvent:
        // Bot joined group/room
        handleJoin(e)

    case webhook.LeaveEvent:
        // Bot left group/room
        handleLeave(e)

    case webhook.PostbackEvent:
        // User performed postback action
        handlePostback(e)

    case webhook.BeaconEvent:
        // Beacon detection
        handleBeacon(e)

    case webhook.MemberJoinedEvent:
        // User joined group with bot
        handleMemberJoined(e)

    case webhook.MemberLeftEvent:
        // User left group with bot
        handleMemberLeft(e)

    case webhook.VideoPlayCompleteEvent:
        // Video playback completed
        handleVideoComplete(e)

    default:
        log.Printf("Unsupported event type: %T\n", event)
    }
}
```

### Message Content Type Handling

Handle different message types within a MessageEvent:

```go
func handleMessage(e webhook.MessageEvent) {
    switch message := e.Message.(type) {
    case webhook.TextMessageContent:
        log.Printf("Text: %s", message.Text)

    case webhook.ImageMessageContent:
        log.Printf("Image ID: %s", message.Id)

    case webhook.VideoMessageContent:
        log.Printf("Video ID: %s", message.Id)

    case webhook.AudioMessageContent:
        log.Printf("Audio ID: %s, duration: %d", message.Id, message.Duration)

    case webhook.FileMessageContent:
        log.Printf("File: %s (%d bytes)", message.FileName, message.FileSize)

    case webhook.LocationMessageContent:
        log.Printf("Location: %f, %f - %s",
            message.Latitude, message.Longitude, message.Address)

    case webhook.StickerMessageContent:
        log.Printf("Sticker ID: %s, type: %s",
            message.StickerId, message.StickerResourceType)

    default:
        log.Printf("Unsupported message type: %T", message)
    }
}
```

### Event Properties

All webhook events contain:

- **Type**: Event type identifier (string)
- **Timestamp**: Event time in milliseconds (int64)
- **WebhookEventId**: Unique ULID identifier (string)
- **Mode**: "active" or "standby" (EventMode)
- **DeliveryContext**: Contains `IsRedelivery` flag (bool)
- **Source**: SourceInterface identifying sender (user/group/room)
- **ReplyToken**: Token for sending replies (when applicable)

### Extracting Source Information

Get user, group, or room IDs from events:

```go
// User ID
if e.Source.UserId != "" {
    userID := e.Source.UserId
}

// Group ID
if e.Source.GroupId != "" {
    groupID := e.Source.GroupId
}

// Room ID
if e.Source.RoomId != "" {
    roomID := e.Source.RoomId
}
```

### Using WebhookHandler

Alternative webhook handling with handler pattern:

```go
handler, err := webhook.NewWebhookHandler(channelSecret)
if err != nil {
    log.Fatal(err)
}

handler.HandleEvents(func(cb *webhook.CallbackRequest, r *http.Request) {
    for _, event := range cb.Events {
        handleEvent(event)
    }
})

handler.HandleError(func(err error, r *http.Request) {
    log.Printf("Webhook error: %v", err)
})

http.Handle("/webhook", handler)
```

## Sending Messages

### Reply Messages

Reply to webhook events using reply tokens (valid for 30 seconds):

```go
bot.ReplyMessage(
    &messaging_api.ReplyMessageRequest{
        ReplyToken: e.ReplyToken,
        Messages: []messaging_api.MessageInterface{
            messaging_api.TextMessage{
                Text: "Hello, world!",
            },
        },
    },
)
```

### Push Messages

Send messages directly to users, groups, or rooms:

```go
bot.PushMessage(
    &messaging_api.PushMessageRequest{
        To: "U1234567890abcdef1234567890abcdef", // User ID
        Messages: []messaging_api.MessageInterface{
            messaging_api.TextMessage{
                Text: "Direct message",
            },
        },
    },
    "", // x-line-retry-key (optional)
)
```

### Broadcast Messages

Send to all followers:

```go
bot.Broadcast(
    &messaging_api.BroadcastRequest{
        Messages: []messaging_api.MessageInterface{
            messaging_api.TextMessage{
                Text: "Announcement to all followers",
            },
        },
    },
    "", // x-line-retry-key (optional)
)
```

### Multicast Messages

Send to multiple specific users:

```go
bot.Multicast(
    &messaging_api.MulticastRequest{
        To: []string{
            "U1234567890abcdef1234567890abcdef",
            "U0987654321fedcba0987654321fedcba",
        },
        Messages: []messaging_api.MessageInterface{
            messaging_api.TextMessage{
                Text: "Message to selected users",
            },
        },
    },
    "", // x-line-retry-key (optional)
)
```

### Narrowcast Messages

Send to users matching demographic or filter criteria:

```go
bot.Narrowcast(
    &messaging_api.NarrowcastRequest{
        Messages: []messaging_api.MessageInterface{
            messaging_api.TextMessage{
                Text: "Targeted message",
            },
        },
        Filter: &messaging_api.Filter{
            Demographic: &messaging_api.DemographicFilter{
                Type: "gender",
                OneOf: []string{"male"},
            },
        },
    },
    "", // x-line-retry-key (optional)
)
```

### Sending Multiple Messages

Send up to 5 messages in a single request:

```go
bot.ReplyMessage(
    &messaging_api.ReplyMessageRequest{
        ReplyToken: e.ReplyToken,
        Messages: []messaging_api.MessageInterface{
            messaging_api.TextMessage{
                Text: "First message",
            },
            messaging_api.TextMessage{
                Text: "Second message",
            },
            messaging_api.StickerMessage{
                PackageId: "446",
                StickerId: "1988",
            },
        },
    },
)
```

### Message Validation

Validate messages before sending:

```go
err := bot.ValidateReply(
    &messaging_api.ValidateMessageRequest{
        Messages: []messaging_api.MessageInterface{
            messaging_api.TextMessage{
                Text: "Test message",
            },
        },
    },
)
if err != nil {
    log.Printf("Invalid message: %v", err)
}
```

## Message Types

### Text Messages

Basic text message:

```go
messaging_api.TextMessage{
    Text: "Hello, world!",
}
```

Text message with emoji substitution:

```go
messaging_api.TextMessageV2{
    Text: "Hello! {smile}",
    Substitution: map[string]messaging_api.SubstitutionObjectInterface{
        "smile": &messaging_api.EmojiSubstitutionObject{
            ProductId: "5ac1bfd5040ab15980c9b435",
            EmojiId:   "002",
        },
    },
}
```

### Image Messages

```go
messaging_api.ImageMessage{
    OriginalContentUrl: "https://example.com/image.jpg",
    PreviewImageUrl:    "https://example.com/preview.jpg",
}
```

### Video Messages

```go
messaging_api.VideoMessage{
    OriginalContentUrl: "https://example.com/video.mp4",
    PreviewImageUrl:    "https://example.com/preview.jpg",
}
```

### Audio Messages

```go
messaging_api.AudioMessage{
    OriginalContentUrl: "https://example.com/audio.m4a",
    Duration:           60000, // milliseconds
}
```

### Location Messages

```go
messaging_api.LocationMessage{
    Title:     "My Location",
    Address:   "1-6-1 Yotsuya, Shinjuku-ku, Tokyo",
    Latitude:  35.687574,
    Longitude: 139.72922,
}
```

### Sticker Messages

```go
messaging_api.StickerMessage{
    PackageId: "446",
    StickerId: "1988",
}
```

### Template Messages

**Buttons Template:**

```go
messaging_api.TemplateMessage{
    AltText: "Please select",
    Template: &messaging_api.ButtonsTemplate{
        Text: "Please choose an option",
        Actions: []messaging_api.ActionInterface{
            &messaging_api.MessageAction{
                Label: "Yes",
                Text:  "yes",
            },
            &messaging_api.MessageAction{
                Label: "No",
                Text:  "no",
            },
        },
    },
}
```

**Confirm Template:**

```go
messaging_api.TemplateMessage{
    AltText: "Confirm",
    Template: &messaging_api.ConfirmTemplate{
        Text: "Are you sure?",
        Actions: []messaging_api.ActionInterface{
            &messaging_api.MessageAction{
                Label: "Yes",
                Text:  "yes",
            },
            &messaging_api.MessageAction{
                Label: "No",
                Text:  "no",
            },
        },
    },
}
```

**Carousel Template:**

```go
messaging_api.TemplateMessage{
    AltText: "Products",
    Template: &messaging_api.CarouselTemplate{
        Columns: []messaging_api.CarouselColumn{
            {
                ThumbnailImageUrl: "https://example.com/product1.jpg",
                Title:             "Product 1",
                Text:              "Description",
                Actions: []messaging_api.ActionInterface{
                    &messaging_api.UriAction{
                        Label: "View",
                        Uri:   "https://example.com/products/1",
                    },
                },
            },
            {
                ThumbnailImageUrl: "https://example.com/product2.jpg",
                Title:             "Product 2",
                Text:              "Description",
                Actions: []messaging_api.ActionInterface{
                    &messaging_api.UriAction{
                        Label: "View",
                        Uri:   "https://example.com/products/2",
                    },
                },
            },
        },
    },
}
```

### Flex Messages

Flex messages provide rich, customizable layouts:

```go
messaging_api.FlexMessage{
    AltText: "Flex message",
    Contents: &messaging_api.FlexBubble{
        Hero: &messaging_api.FlexImage{
            Url:  "https://example.com/hero.jpg",
            Size: "full",
        },
        Body: &messaging_api.FlexBox{
            Layout: "vertical",
            Contents: []messaging_api.FlexComponentInterface{
                &messaging_api.FlexText{
                    Text:   "Title",
                    Weight: "bold",
                    Size:   "xl",
                },
                &messaging_api.FlexText{
                    Text: "Description text",
                    Wrap: true,
                },
            },
        },
        Footer: &messaging_api.FlexBox{
            Layout: "vertical",
            Contents: []messaging_api.FlexComponentInterface{
                &messaging_api.FlexButton{
                    Action: &messaging_api.UriAction{
                        Label: "Open",
                        Uri:   "https://example.com",
                    },
                },
            },
        },
    },
}
```

### Imagemap Messages

Interactive image maps with clickable areas:

```go
messaging_api.ImagemapMessage{
    BaseUrl: "https://example.com/imagemap/1040",
    AltText: "Imagemap",
    BaseSize: &messaging_api.ImagemapBaseSize{
        Width:  1040,
        Height: 1040,
    },
    Actions: []messaging_api.ImagemapActionInterface{
        &messaging_api.MessageImagemapAction{
            Area: &messaging_api.ImagemapArea{
                X:      0,
                Y:      0,
                Width:  520,
                Height: 520,
            },
            Text: "Upper left",
        },
        &messaging_api.UriImagemapAction{
            LinkUri: "https://example.com",
            Area: &messaging_api.ImagemapArea{
                X:      520,
                Y:      0,
                Width:  520,
                Height: 520,
            },
        },
    },
}
```

## User and Profile Management

### Get User Profile

Retrieve user profile information:

```go
profile, err := bot.GetProfile(userID)
if err != nil {
    log.Fatal(err)
}

log.Printf("Display name: %s", profile.DisplayName)
log.Printf("User ID: %s", profile.UserId)
log.Printf("Picture URL: %s", profile.PictureUrl)
log.Printf("Status message: %s", profile.StatusMessage)
```

### Get Followers

Get list of users who have added your bot:

```go
followers, err := bot.GetFollowers("", 100) // start token, limit
if err != nil {
    log.Fatal(err)
}

for _, userId := range followers.UserIds {
    log.Printf("Follower: %s", userId)
}

// Paginate with next token
if followers.Next != "" {
    nextPage, _ := bot.GetFollowers(followers.Next, 100)
}
```

### Issue Link Token

Generate a link token for account linking:

```go
response, err := bot.IssueLinkToken(userID)
if err != nil {
    log.Fatal(err)
}

linkToken := response.LinkToken
```

### Get Bot Info

Retrieve your bot's basic information:

```go
botInfo, err := bot.GetBotInfo()
if err != nil {
    log.Fatal(err)
}

log.Printf("Bot name: %s", botInfo.DisplayName)
log.Printf("Bot ID: %s", botInfo.UserId)
log.Printf("Picture URL: %s", botInfo.PictureUrl)
```

## Rich Menu Management

### Create Rich Menu

Define and create a rich menu:

```go
richMenu := &messaging_api.RichMenuRequest{
    Size: &messaging_api.RichMenuSize{
        Width:  2500,
        Height: 1686,
    },
    Selected: true,
    Name:     "Main Menu",
    ChatBarText: "Menu",
    Areas: []messaging_api.RichMenuArea{
        {
            Bounds: &messaging_api.RichMenuBounds{
                X:      0,
                Y:      0,
                Width:  1250,
                Height: 1686,
            },
            Action: &messaging_api.MessageAction{
                Label: "Option 1",
                Text:  "option1",
            },
        },
        {
            Bounds: &messaging_api.RichMenuBounds{
                X:      1250,
                Y:      0,
                Width:  1250,
                Height: 1686,
            },
            Action: &messaging_api.UriAction{
                Label: "Website",
                Uri:   "https://example.com",
            },
        },
    },
}

response, err := bot.CreateRichMenu(richMenu)
if err != nil {
    log.Fatal(err)
}

richMenuID := response.RichMenuId
```

### Upload Rich Menu Image

Upload an image for the rich menu:

```go
imageData, err := os.Open("richmenu.png")
if err != nil {
    log.Fatal(err)
}
defer imageData.Close()

err = blobClient.SetRichMenuImage(richMenuID, "image/png", imageData)
if err != nil {
    log.Fatal(err)
}
```

### Set Default Rich Menu

Set a rich menu as the default for all users:

```go
err := bot.SetDefaultRichMenu(richMenuID)
if err != nil {
    log.Fatal(err)
}
```

### Link Rich Menu to User

Link a rich menu to a specific user:

```go
err := bot.LinkRichMenuIdToUser(userID, richMenuID)
if err != nil {
    log.Fatal(err)
}
```

### Link Rich Menu to Multiple Users

Bulk link a rich menu to multiple users:

```go
err := bot.LinkRichMenuIdToUsers(
    &messaging_api.RichMenuBulkLinkRequest{
        RichMenuId: richMenuID,
        UserIds: []string{
            "U1234567890abcdef1234567890abcdef",
            "U0987654321fedcba0987654321fedcba",
        },
    },
)
```

### Get User's Rich Menu

Get the rich menu linked to a user:

```go
response, err := bot.GetRichMenuIdOfUser(userID)
if err != nil {
    log.Fatal(err)
}

richMenuID := response.RichMenuId
```

### Unlink Rich Menu from User

Remove a rich menu from a user:

```go
err := bot.UnlinkRichMenuIdFromUser(userID)
if err != nil {
    log.Fatal(err)
}
```

### List All Rich Menus

Get all rich menus:

```go
response, err := bot.GetRichMenuList()
if err != nil {
    log.Fatal(err)
}

for _, richMenu := range response.RichMenus {
    log.Printf("Rich menu: %s - %s", richMenu.RichMenuId, richMenu.Name)
}
```

### Delete Rich Menu

Remove a rich menu:

```go
err := bot.DeleteRichMenu(richMenuID)
if err != nil {
    log.Fatal(err)
}
```

### Rich Menu Aliases

Create reusable rich menu aliases:

```go
// Create alias
err := bot.CreateRichMenuAlias(
    &messaging_api.CreateRichMenuAliasRequest{
        RichMenuAliasId: "main-menu",
        RichMenuId:      richMenuID,
    },
)

// Update alias
err = bot.UpdateRichMenuAlias(
    "main-menu",
    &messaging_api.UpdateRichMenuAliasRequest{
        RichMenuId: newRichMenuID,
    },
)

// Delete alias
err = bot.DeleteRichMenuAlias("main-menu")
```

## Group and Room Management

### Get Group Summary

Get group information:

```go
summary, err := bot.GetGroupSummary(groupID)
if err != nil {
    log.Fatal(err)
}

log.Printf("Group name: %s", summary.GroupName)
log.Printf("Group ID: %s", summary.GroupId)
log.Printf("Picture URL: %s", summary.PictureUrl)
```

### Get Group Member Count

```go
count, err := bot.GetGroupMemberCount(groupID)
if err != nil {
    log.Fatal(err)
}

log.Printf("Member count: %d", count.Count)
```

### Get Group Member Profile

Get profile of a specific group member:

```go
profile, err := bot.GetGroupMemberProfile(groupID, userID)
if err != nil {
    log.Fatal(err)
}

log.Printf("Display name: %s", profile.DisplayName)
log.Printf("Picture URL: %s", profile.PictureUrl)
```

### List Group Member IDs

Get all member IDs in a group:

```go
members, err := bot.GetGroupMembersIds(groupID, "")
if err != nil {
    log.Fatal(err)
}

for _, userId := range members.MemberIds {
    log.Printf("Member: %s", userId)
}

// Paginate
if members.Next != "" {
    nextPage, _ := bot.GetGroupMembersIds(groupID, members.Next)
}
```

### Leave Group

Make the bot leave a group:

```go
err := bot.LeaveGroup(groupID)
if err != nil {
    log.Fatal(err)
}
```

### Room Management

Similar methods exist for multi-person rooms:

```go
// Get room member count
count, err := bot.GetRoomMemberCount(roomID)

// Get room member profile
profile, err := bot.GetRoomMemberProfile(roomID, userID)

// List room member IDs
members, err := bot.GetRoomMembersIds(roomID, "")

// Leave room
err = bot.LeaveRoom(roomID)
```

## LIFF Apps

### Create LIFF App

Create a LINE Front-end Framework application:

```go
liffClient, err := liff.NewLiffAPI(channelToken)
if err != nil {
    log.Fatal(err)
}

response, err := liffClient.AddLIFFApp(
    &liff.AddLiffAppRequest{
        View: &liff.LiffView{
            Type: "full", // "compact", "tall", or "full"
            Url:  "https://example.com/liff",
        },
        Description: "My LIFF App",
        Features: &liff.LiffFeatures{
            Ble:  true,
            Qrcode: true,
        },
        Scope: []liff.LiffScope{
            "openid",
            "profile",
            "chat_message.write",
        },
        BotPrompt: "normal", // "normal", "aggressive", or "none"
    },
)
if err != nil {
    log.Fatal(err)
}

liffID := response.LiffId
log.Printf("LIFF ID: %s", liffID)
```

### Get All LIFF Apps

List all LIFF apps:

```go
apps, err := liffClient.GetAllLIFFApps()
if err != nil {
    log.Fatal(err)
}

for _, app := range apps.Apps {
    log.Printf("LIFF ID: %s, URL: %s", app.LiffId, app.View.Url)
}
```

### Update LIFF App

Modify an existing LIFF app:

```go
_, err := liffClient.UpdateLIFFApp(
    liffID,
    &liff.UpdateLiffAppRequest{
        View: &liff.LiffView{
            Type: "tall",
            Url:  "https://example.com/new-liff",
        },
        Description: "Updated LIFF App",
    },
)
if err != nil {
    log.Fatal(err)
}
```

### Delete LIFF App

Remove a LIFF app:

```go
_, err := liffClient.DeleteLIFFApp(liffID)
if err != nil {
    log.Fatal(err)
}
```

## Analytics and Insights

### Get Friends Demographics

Retrieve demographic information about your followers:

```go
insightClient, err := insight.NewInsightAPI(channelToken)
if err != nil {
    log.Fatal(err)
}

demographics, err := insightClient.GetFriendsDemographics()
if err != nil {
    log.Fatal(err)
}

// Gender distribution
for _, gender := range demographics.Genders {
    log.Printf("%s: %.2f%%", gender.Gender, gender.Percentage)
}

// Age distribution
for _, age := range demographics.Ages {
    log.Printf("%s: %.2f%%", age.Age, age.Percentage)
}

// Area distribution
for _, area := range demographics.Areas {
    log.Printf("%s: %.2f%%", area.Area, area.Percentage)
}

// OS distribution
for _, appType := range demographics.AppTypes {
    log.Printf("%s: %.2f%%", appType.AppType, appType.Percentage)
}
```

### Get Follower Count

Get follower statistics for a specific date:

```go
followers, err := insightClient.GetNumberOfFollowers("20251214")
if err != nil {
    log.Fatal(err)
}

if followers.Status == "ready" {
    log.Printf("Followers: %d", followers.Followers)
    log.Printf("Targeted reaches: %d", followers.TargetedReaches)
    log.Printf("Blocks: %d", followers.Blocks)
}
```

### Get Message Delivery Statistics

Get message delivery counts by type:

```go
deliveries, err := insightClient.GetNumberOfMessageDeliveries("20251214")
if err != nil {
    log.Fatal(err)
}

if deliveries.Status == "ready" {
    log.Printf("Broadcast: %d", deliveries.Broadcast)
    log.Printf("Targeting: %d", deliveries.Targeting)
    log.Printf("API Push: %d", deliveries.ApiPush)
    log.Printf("API Reply: %d", deliveries.ApiReply)
}
```

### Get Message Event Statistics

Get detailed statistics for a specific message:

```go
stats, err := insightClient.GetMessageEvent(requestID)
if err != nil {
    log.Fatal(err)
}

log.Printf("Delivered: %d", stats.Overview.Delivered)
log.Printf("Unique impressions: %d", stats.Overview.UniqueImpression)
log.Printf("Unique clicks: %d", stats.Overview.UniqueClick)
log.Printf("Unique media plays: %d", stats.Overview.UniqueMediaPlayed)
```

### Get Statistics Per Unit

Get aggregated statistics by custom unit:

```go
stats, err := insightClient.GetStatisticsPerUnit(
    "custom-unit-id",
    "20251201", // from date
    "20251214", // to date
)
if err != nil {
    log.Fatal(err)
}

log.Printf("Messages sent: %d", stats.Overview.UniqueImpression)
```

## Error Handling

### Basic Error Handling

```go
response, err := bot.ReplyMessage(request)
if err != nil {
    log.Printf("Error sending message: %v", err)
    return
}
```

### Getting Response Headers

Use `WithHttpInfo` methods to access response headers:

```go
response, httpResp, err := bot.ReplyMessageWithHttpInfo(
    &messaging_api.ReplyMessageRequest{
        ReplyToken: replyToken,
        Messages: []messaging_api.MessageInterface{
            messaging_api.TextMessage{
                Text: "Hello",
            },
        },
    },
)

if err == nil {
    requestID := httpResp.Header.Get("x-line-request-id")
    log.Printf("Request ID: %s", requestID)
    log.Printf("Status code: %d", httpResp.StatusCode)
}
```

### Parsing Error Responses

Extract detailed error information from API responses:

```go
response, httpResp, err := bot.ReplyMessageWithHttpInfo(request)
if err != nil && httpResp.StatusCode >= 400 && httpResp.StatusCode < 500 {
    decoder := json.NewDecoder(httpResp.Body)
    errorResponse := &messaging_api.ErrorResponse{}
    if err := decoder.Decode(&errorResponse); err == nil {
        log.Printf("Error: %s", errorResponse.Message)
        for _, detail := range errorResponse.Details {
            log.Printf("Detail: %s", detail.Message)
        }
    }
}
```

### Webhook Signature Validation Errors

```go
cb, err := webhook.ParseRequest(channelSecret, req)
if err != nil {
    if errors.Is(err, webhook.ErrInvalidSignature) {
        log.Println("Invalid signature - possible unauthorized request")
        w.WriteHeader(400)
    } else {
        log.Printf("Parse error: %v", err)
        w.WriteHeader(500)
    }
    return
}
```

## Custom Configuration

### Custom HTTP Client

Configure a custom HTTP client:

```go
customClient := &http.Client{
    Timeout: 30 * time.Second,
    Transport: &http.Transport{
        MaxIdleConns:        100,
        MaxIdleConnsPerHost: 10,
    },
}

bot, err := messaging_api.NewMessagingApiAPI(
    channelToken,
    messaging_api.WithHTTPClient(customClient),
)
```

### Custom API Endpoint

Use a custom API endpoint:

```go
bot, err := messaging_api.NewMessagingApiAPI(
    channelToken,
    messaging_api.WithEndpoint("https://custom-endpoint.example.com"),
)
```

### Custom Blob Endpoint

Configure separate endpoint for blob operations:

```go
blobClient := messaging_api.NewMessagingApiBlobAPI(
    channelToken,
    messaging_api.WithBlobHTTPClient(customClient),
    messaging_api.WithBlobEndpoint("https://blob-endpoint.example.com"),
)
```

### Context Support

Use context for request cancellation and timeouts:

```go
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

bot.WithContext(ctx).ReplyMessage(request)
```

### Skip Signature Validation (Development)

For local development, optionally skip signature validation:

```go
opt := &webhook.ParseOption{
    SkipSignatureValidation: func() bool {
        return os.Getenv("SKIP_SIGNATURE") == "true"
    },
}

cb, err := webhook.ParseRequestWithOption(channelSecret, req, opt)
```

## Webhook Events Reference

### Event Types

| Event Type | Description | Key Properties |
|------------|-------------|----------------|
| **MessageEvent** | User sends a message | Message (content interface), ReplyToken |
| **FollowEvent** | User adds bot as friend | ReplyToken |
| **UnfollowEvent** | User blocks/removes bot | - |
| **JoinEvent** | Bot joins group/room | ReplyToken |
| **LeaveEvent** | Bot leaves group/room | - |
| **PostbackEvent** | User performs postback action | Postback.Data, Postback.Params |
| **BeaconEvent** | Beacon detection | Beacon.Hwid, Beacon.Type |
| **MemberJoinedEvent** | User joins group with bot | Joined.Members |
| **MemberLeftEvent** | User leaves group with bot | Left.Members |
| **ThingsEvent** | IoT device communication | Things (content interface) |
| **AccountLinkEvent** | Account linking result | Link.Result, Link.Nonce |
| **VideoPlayCompleteEvent** | Video playback completed | VideoPlayComplete.TrackingId |
| **ActivatedEvent** | Bot activated in chat | - |
| **DeactivatedEvent** | Bot deactivated in chat | - |
| **BotSuspendedEvent** | Bot suspended | - |
| **BotResumedEvent** | Bot resumed | - |

### Message Content Types

| Type | Description | Key Properties |
|------|-------------|----------------|
| **TextMessageContent** | Text message | Text, Emojis, Mention |
| **ImageMessageContent** | Image | Id, ContentProvider, ImageSet |
| **VideoMessageContent** | Video | Id, Duration, ContentProvider |
| **AudioMessageContent** | Audio | Id, Duration, ContentProvider |
| **FileMessageContent** | File | Id, FileName, FileSize |
| **LocationMessageContent** | Location | Title, Address, Latitude, Longitude |
| **StickerMessageContent** | Sticker | StickerId, PackageId, StickerResourceType |

### Source Types

| Type | Properties |
|------|------------|
| **UserSource** | UserId |
| **GroupSource** | GroupId, UserId (optional) |
| **RoomSource** | RoomId, UserId (optional) |

## Message Actions Reference

### Action Types

| Action Type | Description | Properties |
|-------------|-------------|------------|
| **MessageAction** | Sends text message | Label, Text |
| **UriAction** | Opens URI/URL | Label, Uri |
| **PostbackAction** | Sends postback data | Label, Data, DisplayText |
| **DatetimePickerAction** | Date/time selection | Label, Data, Mode, Initial, Max, Min |
| **CameraAction** | Camera capture | Label |
| **CameraRollAction** | Camera roll access | Label |
| **LocationAction** | Location request | Label |
| **ClipboardAction** | Copy to clipboard | Label, ClipboardText |
| **RichMenuSwitchAction** | Switch rich menu | Label, RichMenuAliasId, Data |

### Action Usage in Templates

Actions can be used in:
- Template messages (Buttons, Confirm, Carousel)
- Flex messages (Buttons, Images)
- Rich menus
- Imagemap messages

## API Methods Reference

### Message Sending

```go
// Reply to events
ReplyMessage(request *ReplyMessageRequest) (*ReplyMessageResponse, error)

// Push to users
PushMessage(request *PushMessageRequest, retryKey string) (*PushMessageResponse, error)

// Broadcast to all followers
Broadcast(request *BroadcastRequest, retryKey string) error

// Multicast to multiple users
Multicast(request *MulticastRequest, retryKey string) error

// Narrowcast with targeting
Narrowcast(request *NarrowcastRequest, retryKey string) (*NarrowcastProgressResponse, error)

// Push by phone number
PushMessagesByPhone(request *PnpMessagesRequest, deliveryTag string) error

// Validate messages
ValidateReply(request *ValidateMessageRequest) error
ValidatePush(request *ValidateMessageRequest) error
ValidateBroadcast(request *ValidateMessageRequest) error
ValidateMulticast(request *ValidateMessageRequest) error
ValidateNarrowcast(request *ValidateMessageRequest) error
```

### User Profile

```go
// Get user profile
GetProfile(userId string) (*UserProfileResponse, error)

// Get bot info
GetBotInfo() (*BotInfoResponse, error)

// Get followers
GetFollowers(start string, limit int32) (*GetFollowersResponse, error)

// Issue link token
IssueLinkToken(userId string) (*IssueLinkTokenResponse, error)
```

### Rich Menu

```go
// Create rich menu
CreateRichMenu(request *RichMenuRequest) (*RichMenuIdResponse, error)

// Get rich menu
GetRichMenu(richMenuId string) (*RichMenuResponse, error)

// Delete rich menu
DeleteRichMenu(richMenuId string) error

// List rich menus
GetRichMenuList() (*RichMenuListResponse, error)

// Set default rich menu
SetDefaultRichMenu(richMenuId string) error

// Get default rich menu ID
GetDefaultRichMenuId() (*RichMenuIdResponse, error)

// Cancel default rich menu
CancelDefaultRichMenu() error

// Link to user
LinkRichMenuIdToUser(userId, richMenuId string) error

// Unlink from user
UnlinkRichMenuIdFromUser(userId string) error

// Get user's rich menu
GetRichMenuIdOfUser(userId string) (*RichMenuIdResponse, error)

// Bulk link
LinkRichMenuIdToUsers(request *RichMenuBulkLinkRequest) error

// Bulk unlink
UnlinkRichMenuIdFromUsers(request *RichMenuBulkUnlinkRequest) error
```

### Group & Room

```go
// Group management
GetGroupSummary(groupId string) (*GroupSummaryResponse, error)
GetGroupMemberCount(groupId string) (*GroupMemberCountResponse, error)
GetGroupMemberProfile(groupId, userId string) (*GroupUserProfileResponse, error)
GetGroupMembersIds(groupId, start string) (*MembersIdsResponse, error)
LeaveGroup(groupId string) error

// Room management
GetRoomMemberCount(roomId string) (*RoomMemberCountResponse, error)
GetRoomMemberProfile(roomId, userId string) (*RoomUserProfileResponse, error)
GetRoomMembersIds(roomId, start string) (*MembersIdsResponse, error)
LeaveRoom(roomId string) error
```

### Message Content

```go
// Get message content
GetMessageContent(messageId string) (*http.Response, error)

// Get message preview
GetMessageContentPreview(messageId string) (*http.Response, error)

// Get transcoding status
GetMessageContentTranscodingByMessageId(messageId string) (*GetMessageContentTranscodingResponse, error)

// Mark as read
MarkMessagesAsRead(request *MarkMessagesAsReadRequest) error
MarkMessagesAsReadByToken(request *MarkMessagesAsReadByTokenRequest) error
```

### Statistics

```go
// Message quota
GetMessageQuota() (*MessageQuotaResponse, error)
GetMessageQuotaConsumption() (*QuotaConsumptionResponse, error)

// Message statistics
GetNumberOfSentPushMessages(date string) (*NumberOfMessagesResponse, error)
GetNumberOfSentReplyMessages(date string) (*NumberOfMessagesResponse, error)
GetNumberOfSentMulticastMessages(date string) (*NumberOfMessagesResponse, error)
GetNumberOfSentBroadcastMessages(date string) (*NumberOfMessagesResponse, error)
```

### LIFF

```go
// Create LIFF app
AddLIFFApp(request *AddLiffAppRequest) (*AddLiffAppResponse, error)

// Get all LIFF apps
GetAllLIFFApps() (*GetAllLiffAppsResponse, error)

// Update LIFF app
UpdateLIFFApp(liffId string, request *UpdateLiffAppRequest) (struct{}, error)

// Delete LIFF app
DeleteLIFFApp(liffId string) (struct{}, error)
```

### Insight

```go
// Get demographics
GetFriendsDemographics() (*GetFriendsDemographicsResponse, error)

// Get follower count
GetNumberOfFollowers(date string) (*GetNumberOfFollowersResponse, error)

// Get delivery statistics
GetNumberOfMessageDeliveries(date string) (*GetNumberOfMessageDeliveriesResponse, error)

// Get message event statistics
GetMessageEvent(requestId string) (*GetMessageEventResponse, error)

// Get statistics per unit
GetStatisticsPerUnit(customAggregationUnit, from, to string) (*GetStatisticsPerUnitResponse, error)
```

---

## Best Practices

### Security

1. **Always validate webhook signatures** using `webhook.ParseRequest()`
2. **Store channel secret and token securely** (environment variables, not hardcoded)
3. **Use HTTPS** for production webhooks
4. **Validate user input** before processing

### Performance

1. **Use `WithHttpInfo` methods** when you need response headers for debugging
2. **Implement retry logic** with exponential backoff for failed requests
3. **Use context with timeouts** to prevent hanging requests
4. **Cache rich menu IDs** instead of querying repeatedly

### Error Handling

1. **Check HTTP status codes** from `WithHttpInfo` responses
2. **Log request IDs** (`x-line-request-id`) for debugging
3. **Handle rate limits** (implement backoff when receiving 429 responses)
4. **Validate messages** before sending to avoid quota waste

### Message Design

1. **Provide alt text** for all template and flex messages
2. **Keep messages concise** (max 5 messages per request)
3. **Use quick reply** for simple user selections
4. **Test on multiple devices** (compact/tall/full view sizes)

### Webhook Handling

1. **Respond quickly** to webhook requests (< 30 seconds)
2. **Process events asynchronously** for long-running operations
3. **Store reply tokens** if processing takes time (valid for 30 seconds)
4. **Handle redelivery** gracefully (check `DeliveryContext.IsRedelivery`)

---

This documentation provides comprehensive coverage of the LINE Messaging API SDK for Go, including installation, core concepts, all major features, complete API reference, and best practices for building robust LINE bots.
